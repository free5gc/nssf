package processor

import (
	"github.com/free5gc/nssf/internal/repository"
)

type Processor struct {
	*repository.RuntimeRepository
}

func NewProcessor(runtimeRepo *repository.RuntimeRepository) *Processor {
	return &Processor{RuntimeRepository: runtimeRepo}
}
