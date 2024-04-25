package sbi

import (
	"context"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"

	nssf_context "github.com/free5gc/nssf/internal/context"
	"github.com/free5gc/nssf/internal/logger"
	"github.com/free5gc/nssf/internal/sbi/processor"
	"github.com/free5gc/nssf/pkg/factory"
	"github.com/free5gc/util/httpwrapper"
	logger_util "github.com/free5gc/util/logger"
)

type Nssf interface {
	Config() *factory.Config
	Context() *nssf_context.NSSFContext
}

type Server struct {
	Nssf

	httpServer *http.Server
	router     *gin.Engine
	processor  *processor.Processor
}

func NewServer(nssf Nssf, tlsKeyLogPath string) *Server {
	s := &Server{
		Nssf:      nssf,
		processor: processor.NewProcessor(nssf),
	}

	s.router = newRouter(s)

	server, err := bindRouter(nssf, s.router, tlsKeyLogPath)
	s.httpServer = server

	if err != nil {
		logger.SBILog.Errorf("bind Router Error: %+v", err)
		panic("Server initialization failed")
	}

	return s
}

func (s *Server) Processor() *processor.Processor {
	return s.processor
}

func (s *Server) Run(wg *sync.WaitGroup) {
	logger.SBILog.Info("Starting server...")

	wg.Add(1)
	go func() {
		defer wg.Done()

		err := s.serve()
		if err != http.ErrServerClosed {
			logger.SBILog.Panicf("HTTP server setup failed: %+v", err)
		}
	}()
}

func (s *Server) Shutdown() {
	s.shutdownHttpServer()
}

func (s *Server) shutdownHttpServer() {
	const shutdownTimeout time.Duration = 2 * time.Second

	if s.httpServer == nil {
		return
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
	defer cancel()

	err := s.httpServer.Shutdown(shutdownCtx)
	if err != nil {
		logger.SBILog.Errorf("HTTP server shutdown failed: %+v", err)
	}
}

func bindRouter(nssf Nssf, router *gin.Engine, tlsKeyLogPath string) (*http.Server, error) {
	sbiConfig := nssf.Config().Configuration.Sbi
	bindAddr := fmt.Sprintf("%s:%d", sbiConfig.BindingIPv4, sbiConfig.Port)

	return httpwrapper.NewHttp2Server(bindAddr, tlsKeyLogPath, router)
}

func newRouter(s *Server) *gin.Engine {
	router := logger_util.NewGinWithLogrus(logger.GinLog)

	nssaiAvailabilityGroup := router.Group(factory.NssfNssaiavailResUriPrefix)
	nssaiAvailabilityRoutes := s.getNssaiAvailabilityRoutes()
	AddService(nssaiAvailabilityGroup, nssaiAvailabilityRoutes)

	nsSelectionGroup := router.Group(factory.NssfNsselectResUriPrefix)
	nsSelectionRoutes := s.getNsSelectionRoutes()
	AddService(nsSelectionGroup, nsSelectionRoutes)

	return router
}

func (s *Server) unsecureServe() error {
	return s.httpServer.ListenAndServe()
}

func (s *Server) secureServe() error {
	sbiConfig := s.Nssf.Config().Configuration.Sbi

	pemPath := sbiConfig.Tls.Pem
	if pemPath == "" {
		pemPath = factory.NssfDefaultCertPemPath
	}

	keyPath := sbiConfig.Tls.Key
	if keyPath == "" {
		keyPath = factory.NssfDefaultPrivateKeyPath
	}

	return s.httpServer.ListenAndServeTLS(pemPath, keyPath)
}

func (s *Server) serve() error {
	sbiConfig := s.Nssf.Config().Configuration.Sbi

	switch sbiConfig.Scheme {
	case "http":
		return s.unsecureServe()
	case "https":
		return s.secureServe()
	default:
		return fmt.Errorf("invalid SBI scheme: %s", sbiConfig.Scheme)
	}
}
