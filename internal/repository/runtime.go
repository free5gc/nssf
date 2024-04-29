package repository

import (
	nssf_context "github.com/free5gc/nssf/internal/context"
	"github.com/free5gc/nssf/pkg/factory"
)

type RuntimeRepository struct {
	config  *factory.Config
	nssfCtx *nssf_context.NSSFContext
}

func NewRuntimeRepository(cfg *factory.Config) *RuntimeRepository {
	return &RuntimeRepository{
		config:  cfg,
		nssfCtx: nssf_context.GetSelf(),
	}
}

func (rr RuntimeRepository) Config() *factory.Config {
	return rr.config
}

func (rr RuntimeRepository) Context() *nssf_context.NSSFContext {
	return rr.nssfCtx
}
