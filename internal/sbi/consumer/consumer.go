package consumer

import (
	"github.com/free5gc/nssf/pkg/app"
	"github.com/free5gc/openapi/Nnrf_NFManagement"
)

type Consumer struct {
	app.NssfApp

	*NrfService
}

func NewConsumer(nssf app.NssfApp) *Consumer {
	configuration := Nnrf_NFManagement.NewConfiguration()
	configuration.SetBasePath(nssf.Context().NrfUri)
	nrfService := &NrfService{
		nrfNfMgmtClient: Nnrf_NFManagement.NewAPIClient(configuration),
	}

	return &Consumer{
		NssfApp:    nssf,
		NrfService: nrfService,
	}
}
