package context

import (
	"slices"

	"github.com/free5gc/openapi/models"
)

var servicePolicies = map[models.Nrf_NFMgmt_ServiceName][]models.Nrf_NFMgmt_NFType{
	models.Nrf_NFMgmt_ServiceName_NNSSF_NSSELECTION: {
		models.Nrf_NFMgmt_NFType_AMF,
		models.Nrf_NFMgmt_NFType_NSSF,
		models.Nrf_NFMgmt_NFType_SMF,
		models.Nrf_NFMgmt_NFType_NWDAF,
	},
	models.Nrf_NFMgmt_ServiceName_NNSSF_NSSAIAVAILABILITY: {
		models.Nrf_NFMgmt_NFType_AMF,
	},
}

func AllowedNfTypesForService(serviceName models.Nrf_NFMgmt_ServiceName) (
	[]models.Nrf_NFMgmt_NFType, bool,
) {
	allowed, known := servicePolicies[serviceName]
	return slices.Clone(allowed), known
}
