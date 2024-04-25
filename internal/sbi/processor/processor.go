package processor

import (
	nssf_context "github.com/free5gc/nssf/internal/context"
	"github.com/free5gc/nssf/pkg/factory"
)

// TODO: Resolve the circular dependency between internal/sbi/processor/processor.go and internal/sbi/nssf.go
type Nssf interface {
	Config() *factory.Config
	Context() *nssf_context.NSSFContext
	Processor() *Processor
}

type Processor struct {
	Nssf
}

func NewProcessor(nssf Nssf) *Processor {
	return &Processor{
		Nssf: nssf,
	}
}
