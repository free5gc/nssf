/*
 * NSSF Service
 */

package service

import (
	"context"
	"fmt"
	"io"
	"os"
	"runtime/debug"
	"sync"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/sirupsen/logrus"

	nssf_context "github.com/free5gc/nssf/internal/context"
	"github.com/free5gc/nssf/internal/logger"
	"github.com/free5gc/nssf/internal/sbi"
	"github.com/free5gc/nssf/internal/sbi/consumer"
	"github.com/free5gc/nssf/internal/sbi/processor"
	"github.com/free5gc/nssf/pkg/app"
	"github.com/free5gc/nssf/pkg/factory"
	"github.com/free5gc/util/metrics"
	"github.com/free5gc/util/metrics/utils"
)

type NssfApp struct {
	cfg     *factory.Config
	nssfCtx *nssf_context.NSSFContext

	ctx           context.Context
	cancel        context.CancelFunc
	wg            sync.WaitGroup
	sbiServer     *sbi.Server
	metricsServer *metrics.Server
	processor     *processor.Processor
	consumer      *consumer.Consumer
	registered    bool
}

var _ app.NssfApp = &NssfApp{}

func NewApp(ctx context.Context, cfg *factory.Config, tlsKeyLogPath string) (*NssfApp, error) {
	nssf_context.InitNssfContext()

	nssf := &NssfApp{
		cfg:     cfg,
		wg:      sync.WaitGroup{},
		nssfCtx: nssf_context.GetSelf(),
	}
	nssf.SetLogEnable(cfg.GetLogEnable())
	nssf.SetLogLevel(cfg.GetLogLevel())
	nssf.SetReportCaller(cfg.GetLogReportCaller())

	nssf.ctx, nssf.cancel = context.WithCancel(ctx)

	processor := processor.NewProcessor(nssf)
	nssf.processor = processor

	consumer, err := consumer.NewConsumer(nssf)
	if err != nil {
		return nil, err
	}
	nssf.consumer = consumer

	sbiServer := sbi.NewServer(nssf, tlsKeyLogPath)
	nssf.sbiServer = sbiServer

	features := map[utils.MetricTypeEnabled]bool{utils.SBI: true}
	customMetrics := make(map[utils.MetricTypeEnabled][]prometheus.Collector)
	if cfg.AreMetricsEnabled() {
		var err error
		if nssf.metricsServer, err = metrics.NewServer(
			getInitMetrics(cfg, features, customMetrics), tlsKeyLogPath, logger.InitLog); err != nil {
			return nil, err
		}
	}

	return nssf, nil
}

func getInitMetrics(
	cfg *factory.Config,
	features map[utils.MetricTypeEnabled]bool,
	customMetrics map[utils.MetricTypeEnabled][]prometheus.Collector,
) metrics.InitMetrics {
	metricsInfo := metrics.Metrics{
		BindingIPv4: cfg.GetMetricsBindingAddr(),
		Scheme:      cfg.GetMetricsScheme(),
		Namespace:   cfg.GetMetricsNamespace(),
		Port:        cfg.GetMetricsPort(),
		Tls: metrics.Tls{
			Key: cfg.GetMetricsCertKeyPath(),
			Pem: cfg.GetMetricsCertPemPath(),
		},
	}

	return metrics.NewInitMetrics(metricsInfo, "NSSF", features, customMetrics)
}

func (a *NssfApp) Config() *factory.Config {
	return a.cfg
}

func (a *NssfApp) Context() *nssf_context.NSSFContext {
	return a.nssfCtx
}

func (a *NssfApp) Processor() *processor.Processor {
	return a.processor
}

func (a *NssfApp) Consumer() *consumer.Consumer {
	return a.consumer
}

func (a *NssfApp) SetLogEnable(enable bool) {
	logger.MainLog.Infof("Log enable is set to [%v]", enable)
	if enable && logger.Log.Out == os.Stderr {
		return
	} else if !enable && logger.Log.Out == io.Discard {
		return
	}

	a.cfg.SetLogEnable(enable)
	if enable {
		logger.Log.SetOutput(os.Stderr)
	} else {
		logger.Log.SetOutput(io.Discard)
	}
}

func (a *NssfApp) SetLogLevel(level string) {
	lvl, err := logrus.ParseLevel(level)
	if err != nil {
		logger.MainLog.Warnf("Log level [%s] is invalid", level)
		return
	}

	logger.MainLog.Infof("Log level is set to [%s]", level)
	if lvl == logger.Log.GetLevel() {
		return
	}

	a.cfg.SetLogLevel(level)
	logger.Log.SetLevel(lvl)
}

func (a *NssfApp) SetReportCaller(reportCaller bool) {
	logger.MainLog.Infof("Report Caller is set to [%v]", reportCaller)
	if reportCaller == logger.Log.ReportCaller {
		return
	}

	a.cfg.SetLogReportCaller(reportCaller)
	logger.Log.SetReportCaller(reportCaller)
}

func (a *NssfApp) registerToNrf(ctx context.Context) error {
	if err := a.consumer.SendRegisterNFInstance(ctx, true); err != nil {
		return fmt.Errorf("failed to register NSSF to NRF: %s", err.Error())
	}

	return nil
}

func (a *NssfApp) deregisterFromNrf() {
	if !a.registered {
		return
	}
	problemDetails, err := a.consumer.SendDeregisterNFInstance(a.nssfCtx.NfId)
	if problemDetails != nil {
		logger.InitLog.Errorf("Deregister NF instance Failed Problem[%+v]", problemDetails)
	} else if err != nil {
		logger.InitLog.Errorf("Deregister NF instance Error[%+v]", err)
	} else {
		logger.InitLog.Infof("Deregister from NRF successfully")
	}
}

func (a *NssfApp) Start() {
	err := a.registerToNrf(a.ctx)
	a.registered = err == nil
	if err != nil {
		logger.MainLog.Errorf("register to NRF failed: %+v", err)
	} else {
		logger.MainLog.Infoln("register to NRF successfully")
	}

	// Graceful deregister when panic
	defer func() {
		if p := recover(); p != nil {
			a.deregisterFromNrf()
			logger.InitLog.Fatalf("panic: %v\n%s", p, string(debug.Stack()))
		}
	}()

	if a.registered {
		// Only a registered profile has something to keep alive.
		a.consumer.StartHeartbeat(a.ctx, &a.wg)
	}

	a.sbiServer.Run(&a.wg)

	if a.cfg.AreMetricsEnabled() && a.metricsServer != nil {
		go func() {
			a.metricsServer.Run(&a.wg)
		}()
	}

	go a.listenShutdown(a.ctx)
	a.Wait()
}

func (a *NssfApp) listenShutdown(ctx context.Context) {
	<-ctx.Done()
	a.terminateProcedure()
}

func (a *NssfApp) Terminate() {
	a.cancel()
}

func (a *NssfApp) terminateProcedure() {
	logger.MainLog.Infof("Terminating NSSF...")

	// no heartbeat PATCH or re-registration PUT may land after the deregistration
	a.consumer.WaitHeartbeatStopped()

	a.deregisterFromNrf()
	a.sbiServer.Shutdown()
	if a.metricsServer != nil {
		a.metricsServer.Stop()
		logger.MainLog.Infof("NSSF Metrics Server terminated")
	}
}

func (a *NssfApp) Wait() {
	a.wg.Wait()
	logger.MainLog.Infof("NSSF terminated")
}
