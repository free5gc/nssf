package consumer

import (
	"context"
	"sync"

	"github.com/free5gc/openapi/models"
)

// nrfRegistrar adapts NrfService to nfheartbeat.Registrar.
type nrfRegistrar struct {
	s *NrfService
}

func (r nrfRegistrar) UpdateNFInstance(ctx context.Context, patchItems []models.PatchItem) (
	models.Nrf_NFMgmt_NFProfile, *models.ProblemDetails, error,
) {
	return r.s.SendUpdateNFInstance(ctx, patchItems)
}

func (r nrfRegistrar) RegisterNFInstance(ctx context.Context) (int32, error) {
	if err := r.s.SendRegisterNFInstance(ctx, false); err != nil {
		return 0, err
	}
	return r.s.heartbeatTimer, nil
}

// StartHeartbeat launches the periodic NF heartbeat toward the NRF.
// It must be called after a successful NF registration.
func (ns *NrfService) StartHeartbeat(ctx context.Context, wg *sync.WaitGroup) {
	ns.heartbeat.Start(ctx, wg, ns.heartbeatTimer)
}

// WaitHeartbeatStopped blocks until the heartbeat goroutine exits, or returns at once if it never started.
func (ns *NrfService) WaitHeartbeatStopped() {
	ns.heartbeat.Wait()
}
