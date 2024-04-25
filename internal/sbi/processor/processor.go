package processor

import (
	nssf_context "github.com/free5gc/nssf/internal/context"
	"github.com/free5gc/nssf/pkg/factory"
)

type Nssf interface {
	Config() *factory.Config
	Context() *nssf_context.NSSFContext
}

type Processor struct {
	Nssf
}

func NewProcessor(nssf Nssf) *Processor {
	return &Processor{
		Nssf: nssf,
	}
}
