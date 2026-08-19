package consumer

import (
	"github.com/free5gc/nssf/pkg/app"
	"github.com/free5gc/openapi/nrf/NFMgmt"
	sbi_metrics "github.com/free5gc/util/metrics/sbi"
)

type Consumer struct {
	app.NssfApp

	*NrfService
}

func NewConsumer(nssf app.NssfApp) *Consumer {
	configuration := NFMgmt.NewConfiguration()
	configuration.SetBasePath(nssf.Context().NrfUri)
	configuration.SetMetrics(sbi_metrics.SbiMetricHook)
	nrfService := &NrfService{
		nrfNfMgmtClient: NFMgmt.NewAPIClient(configuration),
	}

	return &Consumer{
		NssfApp:    nssf,
		NrfService: nrfService,
	}
}
