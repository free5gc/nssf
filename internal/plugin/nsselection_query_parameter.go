/*
 * NSSF Plugin
 */

package plugin

import (
	"github.com/free5gc/openapi/models"
)

type NsselectionQueryParameter struct {
	NfType *models.Nrf_NFMgmt_NFType `json:"nf-type"`

	NfId string `json:"nf-id"`

	SliceInfoRequestForRegistration *models.Nssf_NSSel_SliceInfoForRegistration `json:"slice-info-request-for-registration,omitempty"` //nolint:lll

	SliceInfoRequestForPduSession *models.Nssf_NSSel_SliceInfoForPDUSession `json:"slice-info-request-for-pdu-session,omitempty"` //nolint:lll

	HomePlmnId *models.PlmnId `json:"home-plmn-id,omitempty"`

	Tai *models.Tai `json:"tai,omitempty"`

	SupportedFeatures string `json:"supported-features,omitempty"`
}
