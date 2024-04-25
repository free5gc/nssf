/*
 * NSSF Service
 */

package service

import (
	"fmt"
	"io"
	"os"
	"os/signal"
	"runtime/debug"
	"sync"
	"syscall"

	"github.com/sirupsen/logrus"

	nssf_context "github.com/free5gc/nssf/internal/context"
	"github.com/free5gc/nssf/internal/logger"
	"github.com/free5gc/nssf/internal/sbi"
	"github.com/free5gc/nssf/internal/sbi/consumer"
	"github.com/free5gc/nssf/pkg/factory"
)

type App interface {
	Config() *factory.Config
	Context() *nssf_context.NSSFContext
}

type NssfApp struct {
	cfg     *factory.Config
	nssfCtx *nssf_context.NSSFContext

	wg        sync.WaitGroup
	sbiServer *sbi.Server
}

var _ App = &NssfApp{}

func NewApp(cfg *factory.Config, tlsKeyLogPath string) (*NssfApp, error) {
	nssf := &NssfApp{cfg: cfg, wg: sync.WaitGroup{}}
	nssf.SetLogEnable(cfg.GetLogEnable())
	nssf.SetLogLevel(cfg.GetLogLevel())
	nssf.SetReportCaller(cfg.GetLogReportCaller())

	sbiServer := sbi.NewServer(nssf, tlsKeyLogPath)
	nssf.sbiServer = sbiServer

	nssf_context.Init()
	nssf.nssfCtx = nssf_context.GetSelf()
	return nssf, nil
}

func (a *NssfApp) Config() *factory.Config {
	return a.cfg
}

func (a *NssfApp) Context() *nssf_context.NSSFContext {
	return a.nssfCtx
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

func (a *NssfApp) registerToNrf() error {
	nssfContext := a.nssfCtx

	profile, err := consumer.BuildNFProfile(nssfContext)
	if err != nil {
		return fmt.Errorf("failed to build NSSF profile")
	}

	_, nssfContext.NfId, err = consumer.SendRegisterNFInstance(nssfContext.NrfUri, nssfContext.NfId, profile)
	if err != nil {
		return fmt.Errorf("failed to register NSSF to NRF: %s", err.Error())
	}

	return nil
}

func (a *NssfApp) deregisterFromNrf() {
	problemDetails, err := consumer.SendDeregisterNFInstance()
	if problemDetails != nil {
		logger.InitLog.Errorf("Deregister NF instance Failed Problem[%+v]", problemDetails)
	} else if err != nil {
		logger.InitLog.Errorf("Deregister NF instance Error[%+v]", err)
	} else {
		logger.InitLog.Infof("Deregister from NRF successfully")
	}
}

func (a *NssfApp) addSigTermHandler() {
	signalChannel := make(chan os.Signal, 1)
	signal.Notify(signalChannel, os.Interrupt, syscall.SIGTERM)
	go func() {
		defer func() {
			if p := recover(); p != nil {
				// Print stack for panic to log. Fatalf() will let program exit.
				logger.InitLog.Fatalf("panic: %v\n%s", p, string(debug.Stack()))
			}
		}()

		<-signalChannel
		a.Terminate()
		os.Exit(0)
	}()
}

func (a *NssfApp) Start(tlsKeyLogPath string) {
	logger.InitLog.Infoln("Server started")

	err := a.registerToNrf()
	if err != nil {
		logger.InitLog.Errorf("Register to NRF failed: %+v", err)
	}

	// Graceful deregister when panic
	defer func() {
		if p := recover(); p != nil {
			logger.InitLog.Errorf("panic: %v\n%s", p, string(debug.Stack()))
			a.deregisterFromNrf()
		}
	}()

	a.sbiServer.Run(&a.wg)
	a.addSigTermHandler()
}

func (nssf *NssfApp) Terminate() {
	logger.InitLog.Infof("Terminating NSSF...")
	nssf.deregisterFromNrf()
	nssf.sbiServer.Shutdown()
	logger.InitLog.Infof("NSSF terminated")
}
