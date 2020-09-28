package logger

import (
	"os"
	"time"

	formatter "github.com/antonfisher/nested-logrus-formatter"
	"github.com/sirupsen/logrus"

	"free5gc/lib/logger_conf"
	"free5gc/lib/logger_util"
)

var log *logrus.Logger
var AppLog *logrus.Entry
var ContextLog *logrus.Entry
var FactoryLog *logrus.Entry
var HandlerLog *logrus.Entry
var InitLog *logrus.Entry
var Nsselection *logrus.Entry
var Nssaiavailability *logrus.Entry
var Util *logrus.Entry
var GinLog *logrus.Entry

func init() {
	log = logrus.New()
	log.SetReportCaller(false)

	log.Formatter = &formatter.Formatter{
		TimestampFormat: time.RFC3339,
		TrimMessages:    true,
		NoFieldsSpace:   true,
		HideKeys:        true,
		FieldsOrder:     []string{"component", "category"},
	}

	free5gcLogHook, err := logger_util.NewFileHook(logger_conf.Free5gcLogFile, os.O_CREATE|os.O_APPEND|os.O_RDWR, 0666)
	if err == nil {
		log.Hooks.Add(free5gcLogHook)
	}

	selfLogHook, err := logger_util.NewFileHook(logger_conf.NfLogDir+"nssf.log", os.O_CREATE|os.O_APPEND|os.O_RDWR, 0666)
	if err == nil {
		log.Hooks.Add(selfLogHook)
	}

	AppLog = log.WithFields(logrus.Fields{"component": "NSSF", "category": "App"})
	ContextLog = log.WithFields(logrus.Fields{"component": "NSSF", "category": "Context"})
	FactoryLog = log.WithFields(logrus.Fields{"component": "NSSF", "category": "Factory"})
	HandlerLog = log.WithFields(logrus.Fields{"component": "NSSF", "category": "Handler"})
	InitLog = log.WithFields(logrus.Fields{"component": "NSSF", "category": "Init"})
	Nsselection = log.WithFields(logrus.Fields{"component": "NSSF", "category": "NsSelect"})
	Nssaiavailability = log.WithFields(logrus.Fields{"component": "NSSF", "category": "NssaiAvail"})
	Util = log.WithFields(logrus.Fields{"component": "NSSF", "category": "Util"})
	GinLog = log.WithFields(logrus.Fields{"component": "NSSF", "category": "GIN"})
}

func SetLogLevel(level logrus.Level) {
	log.SetLevel(level)
}

func SetReportCaller(bool bool) {
	log.SetReportCaller(bool)
}
