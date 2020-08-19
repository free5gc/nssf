/*
 * NSSF NS Selection
 *
 * NSSF Network Slice Selection Service
 */

package producer

import (
	"fmt"
	"math/rand"
	"net/http"

	. "free5gc/lib/openapi/models"
	. "free5gc/src/nssf/plugin"
	"free5gc/src/nssf/util"
)

func selectNsiInformation(nsiInformationList []NsiInformation) NsiInformation {
	// TODO: Algorithm to select Network Slice Instance
	//       Take roaming indication into consideration

	// Randomly select a Network Slice Instance
	idx := rand.Intn(len(nsiInformationList))
	return nsiInformationList[idx]
}

// Network slice selection for PDU session
// The function is executed when the IE, `slice-info-for-pdu-session`, is provided in query parameters
func nsselectionForPduSession(param NsselectionQueryParameter,
	authorizedNetworkSliceInfo *AuthorizedNetworkSliceInfo,
	problemDetails *ProblemDetails) (status int) {
	if param.HomePlmnId != nil {
		// Check whether UE's Home PLMN is supported when UE is a roamer
		if !util.CheckSupportedHplmn(*param.HomePlmnId) {
			authorizedNetworkSliceInfo.RejectedNssaiInPlmn = append(authorizedNetworkSliceInfo.RejectedNssaiInPlmn, *param.SliceInfoRequestForPduSession.SNssai)

			status = http.StatusOK
			return
		}
	}

	if param.Tai != nil {
		// Check whether UE's current TA is supported when UE provides TAI
		if !util.CheckSupportedTa(*param.Tai) {
			authorizedNetworkSliceInfo.RejectedNssaiInTa = append(authorizedNetworkSliceInfo.RejectedNssaiInTa, *param.SliceInfoRequestForPduSession.SNssai)

			status = http.StatusOK
			return
		}
	}

	if param.Tai != nil && !util.CheckSupportedSnssaiInPlmn(*param.SliceInfoRequestForPduSession.SNssai, *param.Tai.PlmnId) {
		// Return ProblemDetails indicating S-NSSAI is not supported
		// TODO: Based on TS 23.501 V15.2.0, if the Requested NSSAI includes an S-NSSAI that is not valid in the
		//       Serving PLMN, the NSSF may derive the Configured NSSAI for Serving PLMN
		*problemDetails = ProblemDetails{
			Title:  util.UNSUPPORTED_RESOURCE,
			Status: http.StatusForbidden,
			Detail: "S-NSSAI in Requested NSSAI is not supported in PLMN",
			Cause:  "SNSSAI_NOT_SUPPORTED",
		}

		status = http.StatusForbidden
		return
	}

	if param.HomePlmnId != nil {
		if param.SliceInfoRequestForPduSession.RoamingIndication == RoamingIndication_NON_ROAMING {
			problemDetail := "`home-plmn-id` is provided, which contradicts `roamingIndication`:'NON_ROAMING'"
			*problemDetails = ProblemDetails{
				Title:  util.INVALID_REQUEST,
				Status: http.StatusBadRequest,
				Detail: problemDetail,
				InvalidParams: []InvalidParam{
					{
						Param:  "home-plmn-id",
						Reason: problemDetail,
					},
				},
			}

			status = http.StatusBadRequest
			return
		}
	} else {
		if param.SliceInfoRequestForPduSession.RoamingIndication != RoamingIndication_NON_ROAMING {
			problemDetail := fmt.Sprintf("`home-plmn-id` is not provided, which contradicts `roamingIndication`:'%s'",
				string(param.SliceInfoRequestForPduSession.RoamingIndication))
			*problemDetails = ProblemDetails{
				Title:  util.INVALID_REQUEST,
				Status: http.StatusBadRequest,
				Detail: problemDetail,
				InvalidParams: []InvalidParam{
					{
						Param:  "home-plmn-id",
						Reason: problemDetail,
					},
				},
			}

			status = http.StatusBadRequest
			return
		}
	}

	if param.Tai != nil && !util.CheckSupportedSnssaiInTa(*param.SliceInfoRequestForPduSession.SNssai, *param.Tai) {
		// Requested S-NSSAI does not supported in UE's current TA
		// Add it to Rejected NSSAI in TA
		authorizedNetworkSliceInfo.RejectedNssaiInTa = append(authorizedNetworkSliceInfo.RejectedNssaiInTa, *param.SliceInfoRequestForPduSession.SNssai)
		status = http.StatusOK
		return
	}

	nsiInformationList := util.GetNsiInformationListFromConfig(*param.SliceInfoRequestForPduSession.SNssai)

	nsiInformation := selectNsiInformation(nsiInformationList)

	authorizedNetworkSliceInfo.NsiInformation = new(NsiInformation)
	*authorizedNetworkSliceInfo.NsiInformation = nsiInformation

	return http.StatusOK
}
