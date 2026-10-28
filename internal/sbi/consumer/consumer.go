package consumer

import (
	"github.com/free5gc/nssf/internal/logger"
	"github.com/free5gc/nssf/pkg/app"
	"github.com/free5gc/openapi/nrf/NFMgmt"
	sbi_metrics "github.com/free5gc/util/metrics/sbi"
	"github.com/free5gc/util/nfheartbeat"
)

type Consumer struct {
	app.NssfApp

	*NrfService
}

func NewConsumer(nssf app.NssfApp) (*Consumer, error) {
	configuration := NFMgmt.NewConfiguration()
	configuration.SetBasePath(nssf.Context().NrfUri)
	configuration.SetMetrics(sbi_metrics.SbiMetricHook)
	nrfService := &NrfService{
		nrfNfMgmtClient: NFMgmt.NewAPIClient(configuration),
	}

	c := &Consumer{
		NssfApp:    nssf,
		NrfService: nrfService,
	}

	heartbeat, err := nfheartbeat.NewRunner(
		nrfRegistrar{nrfService},
		func() int32 { return c.Config().GetNfHeartBeatTimer() },
		logger.ConsumerLog,
	)
	if err != nil {
		return nil, err
	}
	nrfService.heartbeat = heartbeat

	return c, nil
}
