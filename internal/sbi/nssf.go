package sbi

import (
	nssf_context "github.com/free5gc/nssf/internal/context"
	"github.com/free5gc/nssf/internal/sbi/processor"
	"github.com/free5gc/nssf/pkg/factory"
)

type Nssf interface {
	Config() *factory.Config
	Context() *nssf_context.NSSFContext
	Processor() *processor.Processor
}
