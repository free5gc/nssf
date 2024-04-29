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

	"github.com/sirupsen/logrus"

	"github.com/free5gc/nssf/internal/logger"
	"github.com/free5gc/nssf/internal/repository"
	"github.com/free5gc/nssf/internal/sbi"
	"github.com/free5gc/nssf/internal/sbi/consumer"
)

type NssfApp struct {
	*repository.RuntimeRepository

	wg        sync.WaitGroup
	sbiServer *sbi.Server
}

func NewApp(runtimeRepo *repository.RuntimeRepository, tlsKeyLogPath string) (*NssfApp, error) {
	nssf := &NssfApp{
		RuntimeRepository: runtimeRepo,
		wg:                sync.WaitGroup{},
	}

	nssf.SetLogEnable(runtimeRepo.Config().GetLogEnable())
	nssf.SetLogLevel(runtimeRepo.Config().GetLogLevel())
	nssf.SetReportCaller(runtimeRepo.Config().GetLogReportCaller())

	sbiServer := sbi.NewServer(runtimeRepo, tlsKeyLogPath)
	nssf.sbiServer = sbiServer

	return nssf, nil
}

func (a *NssfApp) SetLogEnable(enable bool) {
	logger.MainLog.Infof("Log enable is set to [%v]", enable)
	if enable && logger.Log.Out == os.Stderr {
		return
	} else if !enable && logger.Log.Out == io.Discard {
		return
	}

	a.Config().SetLogEnable(enable)
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

	a.Config().SetLogLevel(level)
	logger.Log.SetLevel(lvl)
}

func (a *NssfApp) SetReportCaller(reportCaller bool) {
	logger.MainLog.Infof("Report Caller is set to [%v]", reportCaller)
	if reportCaller == logger.Log.ReportCaller {
		return
	}

	a.Config().SetLogReportCaller(reportCaller)
	logger.Log.SetReportCaller(reportCaller)
}

func (a *NssfApp) registerToNrf() error {
	nssfContext := a.Context()

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

func (a *NssfApp) Start(ctx context.Context) {
	err := a.registerToNrf()
	if err != nil {
		logger.MainLog.Errorf("Register to NRF failed: %+v", err)
	} else {
		logger.MainLog.Infoln("Register to NRF successfully")
	}

	// Graceful deregister when panic
	defer func() {
		if p := recover(); p != nil {
			logger.InitLog.Errorf("panic: %v\n%s", p, string(debug.Stack()))
			a.deregisterFromNrf()
		}
	}()

	a.sbiServer.Run(&a.wg)
	go a.listenShutdown(ctx)
}

func (a *NssfApp) listenShutdown(ctx context.Context) {
	<-ctx.Done()
	a.Terminate()
}

func (a *NssfApp) Terminate() {
	logger.MainLog.Infof("Terminating NSSF...")
	a.deregisterFromNrf()
	a.sbiServer.Shutdown()
	a.Wait()
}

func (a *NssfApp) Wait() {
	a.wg.Wait()
	logger.MainLog.Infof("NSSF terminated")
}
