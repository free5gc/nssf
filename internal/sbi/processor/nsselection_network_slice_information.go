/*
 * NSSF NS Selection
 *
 * NSSF Network Slice Selection Service
 *
 * API version: 1.0.0
 * Generated by: OpenAPI Generator (https://openapi-generator.tech)
 */

package processor

import (
	"fmt"
	"math/rand"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/free5gc/nssf/internal/logger"
	"github.com/free5gc/nssf/internal/util"
	"github.com/free5gc/openapi"
	"github.com/free5gc/openapi/models"
)

type NetworkSliceInformationGetQuery struct {
	// nolint: lll
	NfType models.NrfNfManagementNfType `form:"nf-type" binding:"required,oneof=NRF UDM AMF SMF AUSF NEF PCF SMSF NSSF UDR LMF GMLC 5G_EIR SEPP UPF N3IWF AF UDSF BSF CHF NWDAF PCSCF CBCF HSS UCMF SOR_AF SPAF MME SCSAS SCEF SCP NSSAAF ICSCF SCSCF DRA IMS_AS AANF 5G_DDNMF NSACF MFAF EASDF DCCF MB_SMF TSCTSF ADRF GBA_BSF CEF MB_UPF NSWOF PKMF MNPF SMS_GMSC SMS_IWMSC MBSF MBSTF PANF IP_SM_GW SMS_ROUTER"`

	NfId string `form:"nf-id" binding:"required,uuid"`

	// nolint: lll
	SliceInfoRequestForRegistration *models.SliceInfoForRegistration `form:"slice-info-request-for-registration" binding:"omitempty"`

	// nolint: lll
	SliceInfoRequestForPduSession *models.SliceInfoForPduSession `form:"slice-info-request-for-pdu-session" binding:"omitempty"`

	// nolint: lll
	SliceInfoRequestForUeConfigurationUpdate *models.SliceInfoForUeConfigurationUpdate `form:"slice-info-request-for-ue-configuration-update" binding:"omitempty"`

	HomePlmnId        *models.PlmnId `form:"home-plmn-id" binding:"required_without=Tai,omitempty"`
	Tai               *models.Tai    `form:"tai" binding:"required_without=HomePlmnId,omitempty"`
	SupportedFeatures string         `form:"supported-features"`
}

// Check if the NF service consumer is authorized
// TODO: Check if the NF service consumer is legal with local configuration, or possibly after querying NRF through
// `nf-id` e.g. Whether the V-NSSF is authorized
func checkNfServiceConsumer(nfType models.NrfNfManagementNfType) error {
	if nfType != models.NrfNfManagementNfType_AMF && nfType != models.NrfNfManagementNfType_NSSF {
		return fmt.Errorf("`nf-type`:'%s' is not authorized to retrieve the slice selection information", string(nfType))
	}

	return nil
}

func (p *Processor) NSSelectionSliceInformationGet(
	c *gin.Context,
	param NetworkSliceInformationGetQuery,
) {
	var (
		status         int
		response       *models.AuthorizedNetworkSliceInfo
		problemDetails *models.ProblemDetails
	)

	// TODO: Record request times of the NF service consumer and response with ProblemDetails of 429 Too Many Requests
	//       if the consumer has sent too many requests in a configured amount of time
	// TODO: Check URI length and response with ProblemDetails of 414 URI Too Long if URI is too long

	// Check permission of NF service consumer
	err := checkNfServiceConsumer(param.NfType)
	if err != nil {
		problemDetails = &models.ProblemDetails{
			Title:  util.UNAUTHORIZED_CONSUMER,
			Status: http.StatusForbidden,
			Detail: err.Error(),
		}
		util.GinProblemJson(c, problemDetails)
		return
	}

	if param.Tai == nil && param.HomePlmnId == nil {
		problemDetails = &models.ProblemDetails{
			Title:  util.MANDATORY_IE_MISSING,
			Status: http.StatusBadRequest,
			Cause:  "MANDATORY_IE_MISSING",
			Detail: "Either `tai` or `home-plmn-id` should be provided",
			InvalidParams: []models.InvalidParam{
				{
					Param: "tai",
				},
				{
					Param: "home-plmn-id",
				},
			},
		}

		util.GinProblemJson(c, problemDetails)
		return
	}

	if param.SliceInfoRequestForRegistration != nil {
		// Network slice information is requested during the Registration procedure
		status, response, problemDetails = nsselectionForRegistration(param)
	} else {
		// Network slice information is requested during the PDU session establishment procedure
		status, response, problemDetails = nsselectionForPduSession(param)
	}

	// TODO: Handle `SliceInfoRequestForUeConfigurationUpdate`

	if problemDetails != nil {
		util.GinProblemJson(c, problemDetails)
		return
	}

	if response == nil {
		util.GinProblemJson(c, &models.ProblemDetails{
			Title:  util.INTERNAL_ERROR,
			Status: http.StatusInternalServerError,
		})
		return
	}

	c.JSON(status, response)
}

// Set Allowed NSSAI with Subscribed S-NSSAI(s) which are marked as default S-NSSAI(s)
func useDefaultSubscribedSnssai(
	param NetworkSliceInformationGetQuery, authorizedNetworkSliceInfo *models.AuthorizedNetworkSliceInfo,
) {
	var mappingOfSnssai []models.MappingOfSnssai
	if param.HomePlmnId != nil {
		// Find mapping of Subscribed S-NSSAI of UE's HPLMN to S-NSSAI in Serving PLMN from NSSF configuration
		mappingOfSnssai = util.GetMappingOfPlmnFromConfig(*param.HomePlmnId)

		if mappingOfSnssai == nil {
			logger.NsselLog.Warnf("No S-NSSAI mapping of UE's HPLMN %+v in NSSF configuration", *param.HomePlmnId)
			return
		}
	}

	for _, subscribedSnssai := range param.SliceInfoRequestForRegistration.SubscribedNssai {
		if subscribedSnssai.DefaultIndication {
			// Subscribed S-NSSAI is marked as default S-NSSAI

			var mappingOfSubscribedSnssai models.Snssai
			// TODO: Compared with Restricted S-NSSAI list in configuration under roaming scenario
			if param.HomePlmnId != nil && !util.CheckStandardSnssai(*subscribedSnssai.SubscribedSnssai) {
				targetMapping, found := util.FindMappingWithHomeSnssai(*subscribedSnssai.SubscribedSnssai, mappingOfSnssai)

				if !found {
					logger.NsselLog.Warnf("No mapping of Subscribed S-NSSAI %+v in PLMN %+v in NSSF configuration",
						*subscribedSnssai.SubscribedSnssai,
						*param.HomePlmnId)
					continue
				} else {
					mappingOfSubscribedSnssai = *targetMapping.ServingSnssai
				}
			} else {
				mappingOfSubscribedSnssai = *subscribedSnssai.SubscribedSnssai
			}

			if param.Tai != nil && !util.CheckSupportedSnssaiInTa(mappingOfSubscribedSnssai, *param.Tai) {
				continue
			}

			var allowedSnssaiElement models.AllowedSnssai
			allowedSnssaiElement.AllowedSnssai = new(models.Snssai)
			*allowedSnssaiElement.AllowedSnssai = mappingOfSubscribedSnssai
			nsiInformationList := util.GetNsiInformationListFromConfig(mappingOfSubscribedSnssai)
			if nsiInformationList != nil {
				// TODO: `NsiInformationList` should be slice in `AllowedSnssai` instead of pointer of slice
				allowedSnssaiElement.NsiInformationList = append(
					allowedSnssaiElement.NsiInformationList,
					nsiInformationList...)
			}
			if param.HomePlmnId != nil && !util.CheckStandardSnssai(*subscribedSnssai.SubscribedSnssai) {
				allowedSnssaiElement.MappedHomeSnssai = new(models.Snssai)
				*allowedSnssaiElement.MappedHomeSnssai = *subscribedSnssai.SubscribedSnssai
			}

			// Default Access Type is set to 3GPP Access if no TAI is provided
			// TODO: Depend on operator implementation, it may also return S-NSSAIs in all valid Access Type if
			//       UE's Access Type could not be identified
			accessType := models.AccessType__3_GPP_ACCESS
			if param.Tai != nil {
				accessType = util.GetAccessTypeFromConfig(*param.Tai)
			}

			util.AddAllowedSnssai(allowedSnssaiElement, accessType, authorizedNetworkSliceInfo)
		}
	}
}

// Set Configured NSSAI with S-NSSAI(s) in Requested NSSAI which are marked as Default Configured NSSAI
func useDefaultConfiguredNssai(
	param NetworkSliceInformationGetQuery, authorizedNetworkSliceInfo *models.AuthorizedNetworkSliceInfo,
) {
	for _, requestedSnssai := range param.SliceInfoRequestForRegistration.RequestedNssai {
		// Check whether the Default Configured S-NSSAI is standard, which could be commonly decided by all roaming partners
		if !util.CheckStandardSnssai(requestedSnssai) {
			logger.NsselLog.Infof("S-NSSAI %+v in Requested NSSAI which based on Default Configured NSSAI is not standard",
				requestedSnssai)
			continue
		}

		// Check whether the Default Configured S-NSSAI is subscribed
		for _, subscribedSnssai := range param.SliceInfoRequestForRegistration.SubscribedNssai {
			if openapi.SnssaiEqualFold(requestedSnssai, *subscribedSnssai.SubscribedSnssai) {
				var configuredSnssai models.ConfiguredSnssai
				configuredSnssai.ConfiguredSnssai = new(models.Snssai)
				*configuredSnssai.ConfiguredSnssai = requestedSnssai

				authorizedNetworkSliceInfo.ConfiguredNssai = append(
					authorizedNetworkSliceInfo.ConfiguredNssai,
					configuredSnssai)
				break
			}
		}
	}
}

// Set Configured NSSAI with Subscribed S-NSSAI(s)
func setConfiguredNssai(
	param NetworkSliceInformationGetQuery, authorizedNetworkSliceInfo *models.AuthorizedNetworkSliceInfo,
) {
	var mappingOfSnssai []models.MappingOfSnssai
	if param.HomePlmnId != nil {
		// Find mapping of Subscribed S-NSSAI of UE's HPLMN to S-NSSAI in Serving PLMN from NSSF configuration
		mappingOfSnssai = util.GetMappingOfPlmnFromConfig(*param.HomePlmnId)

		if mappingOfSnssai == nil {
			logger.NsselLog.Warnf("No S-NSSAI mapping of UE's HPLMN %+v in NSSF configuration", *param.HomePlmnId)
			return
		}
	}

	for _, subscribedSnssai := range param.SliceInfoRequestForRegistration.SubscribedNssai {
		var mappingOfSubscribedSnssai models.Snssai
		if param.HomePlmnId != nil && !util.CheckStandardSnssai(*subscribedSnssai.SubscribedSnssai) {
			targetMapping, found := util.FindMappingWithHomeSnssai(*subscribedSnssai.SubscribedSnssai, mappingOfSnssai)

			if !found {
				logger.NsselLog.Warnf("No mapping of Subscribed S-NSSAI %+v in PLMN %+v in NSSF configuration",
					*subscribedSnssai.SubscribedSnssai,
					*param.HomePlmnId)
				continue
			} else {
				mappingOfSubscribedSnssai = *targetMapping.ServingSnssai
			}
		} else {
			mappingOfSubscribedSnssai = *subscribedSnssai.SubscribedSnssai
		}

		if util.CheckSupportedSnssaiInPlmn(mappingOfSubscribedSnssai, *param.Tai.PlmnId) {
			var configuredSnssai models.ConfiguredSnssai
			configuredSnssai.ConfiguredSnssai = new(models.Snssai)
			*configuredSnssai.ConfiguredSnssai = mappingOfSubscribedSnssai
			if param.HomePlmnId != nil && !util.CheckStandardSnssai(*subscribedSnssai.SubscribedSnssai) {
				configuredSnssai.MappedHomeSnssai = new(models.Snssai)
				*configuredSnssai.MappedHomeSnssai = *subscribedSnssai.SubscribedSnssai
			}

			authorizedNetworkSliceInfo.ConfiguredNssai = append(
				authorizedNetworkSliceInfo.ConfiguredNssai,
				configuredSnssai)
		}
	}
}

// Network slice selection for registration
// The function is executed when the IE, `slice-info-request-for-registration`, is provided in query parameters
func nsselectionForRegistration(param NetworkSliceInformationGetQuery) (
	int, *models.AuthorizedNetworkSliceInfo, *models.ProblemDetails,
) {
	authorizedNetworkSliceInfo := &models.AuthorizedNetworkSliceInfo{}

	var status int
	if param.HomePlmnId != nil {
		// Check whether UE's Home PLMN is supported when UE is a roamer
		if !util.CheckSupportedHplmn(*param.HomePlmnId) {
			authorizedNetworkSliceInfo.RejectedNssaiInPlmn = append(
				authorizedNetworkSliceInfo.RejectedNssaiInPlmn,
				param.SliceInfoRequestForRegistration.RequestedNssai...)

			status = http.StatusOK
			return status, authorizedNetworkSliceInfo, nil
		}
	}

	if param.Tai != nil {
		// Check whether UE's current TA is supported when UE provides TAI
		if !util.CheckSupportedTa(*param.Tai) {
			authorizedNetworkSliceInfo.RejectedNssaiInTa = append(
				authorizedNetworkSliceInfo.RejectedNssaiInTa,
				param.SliceInfoRequestForRegistration.RequestedNssai...)

			status = http.StatusOK
			return status, authorizedNetworkSliceInfo, nil
		}
	}

	if param.SliceInfoRequestForRegistration.RequestMapping {
		// Based on TS 29.531 v15.2.0, when `requestMapping` is set to true, the NSSF shall return the VPLMN specific
		// mapped S-NSSAI values for the S-NSSAI values in `subscribedNssai`. But also `sNssaiForMapping` shall be
		// provided if `requestMapping` is set to true. In the implementation, the NSSF would return mapped S-NSSAIs
		// for S-NSSAIs in both `sNssaiForMapping` and `subscribedSnssai` if present

		if param.HomePlmnId == nil {
			detail := "[Query Parameter] `home-plmn-id` should be provided" +
				" when requesting VPLMN specific mapped S-NSSAI values"
			problemDetails := &models.ProblemDetails{
				Title:  util.INVALID_REQUEST,
				Status: http.StatusBadRequest,
				Detail: detail,
				InvalidParams: []models.InvalidParam{
					{
						Param:  "home-plmn-id",
						Reason: detail,
					},
				},
			}

			status = http.StatusBadRequest
			return status, nil, problemDetails
		}

		mappingOfSnssai := util.GetMappingOfPlmnFromConfig(*param.HomePlmnId)

		if mappingOfSnssai != nil {
			// Find mappings for S-NSSAIs in `subscribedSnssai`
			for _, subscribedSnssai := range param.SliceInfoRequestForRegistration.SubscribedNssai {
				if util.CheckStandardSnssai(*subscribedSnssai.SubscribedSnssai) {
					continue
				}

				targetMapping, found := util.FindMappingWithHomeSnssai(*subscribedSnssai.SubscribedSnssai, mappingOfSnssai)

				if !found {
					logger.NsselLog.Warnf("No mapping of Subscribed S-NSSAI %+v in PLMN %+v in NSSF configuration",
						*subscribedSnssai.SubscribedSnssai,
						*param.HomePlmnId)
					continue
				} else {
					// Add mappings to Allowed NSSAI list
					var allowedSnssaiElement models.AllowedSnssai
					allowedSnssaiElement.AllowedSnssai = new(models.Snssai)
					*allowedSnssaiElement.AllowedSnssai = *targetMapping.ServingSnssai
					allowedSnssaiElement.MappedHomeSnssai = new(models.Snssai)
					*allowedSnssaiElement.MappedHomeSnssai = *subscribedSnssai.SubscribedSnssai

					// Default Access Type is set to 3GPP Access if no TAI is provided
					// TODO: Depend on operator implementation, it may also return S-NSSAIs in all valid Access Type if
					//       UE's Access Type could not be identified
					accessType := models.AccessType__3_GPP_ACCESS
					if param.Tai != nil {
						accessType = util.GetAccessTypeFromConfig(*param.Tai)
					}

					util.AddAllowedSnssai(allowedSnssaiElement, accessType, authorizedNetworkSliceInfo)
				}
			}

			// Find mappings for S-NSSAIs in `sNssaiForMapping`
			for _, snssai := range param.SliceInfoRequestForRegistration.SNssaiForMapping {
				if util.CheckStandardSnssai(snssai) {
					continue
				}

				targetMapping, found := util.FindMappingWithHomeSnssai(snssai, mappingOfSnssai)

				if !found {
					logger.NsselLog.Warnf("No mapping of Subscribed S-NSSAI %+v in PLMN %+v in NSSF configuration",
						snssai,
						*param.HomePlmnId)
					continue
				} else {
					// Add mappings to Allowed NSSAI list
					var allowedSnssaiElement models.AllowedSnssai
					allowedSnssaiElement.AllowedSnssai = new(models.Snssai)
					*allowedSnssaiElement.AllowedSnssai = *targetMapping.ServingSnssai
					allowedSnssaiElement.MappedHomeSnssai = new(models.Snssai)
					*allowedSnssaiElement.MappedHomeSnssai = snssai

					// Default Access Type is set to 3GPP Access if no TAI is provided
					// TODO: Depend on operator implementation, it may also return S-NSSAIs in all valid Access Type if
					//       UE's Access Type could not be identified
					accessType := models.AccessType__3_GPP_ACCESS
					if param.Tai != nil {
						accessType = util.GetAccessTypeFromConfig(*param.Tai)
					}

					util.AddAllowedSnssai(allowedSnssaiElement, accessType, authorizedNetworkSliceInfo)
				}
			}

			status = http.StatusOK
			return status, authorizedNetworkSliceInfo, nil
		} else {
			logger.NsselLog.Warnf("No S-NSSAI mapping of UE's HPLMN %+v in NSSF configuration", *param.HomePlmnId)

			status = http.StatusOK
			return status, authorizedNetworkSliceInfo, nil
		}
	}

	checkInvalidRequestedNssai := false
	if len(param.SliceInfoRequestForRegistration.RequestedNssai) != 0 {
		// Requested NSSAI is provided
		// Verify which S-NSSAI(s) in the Requested NSSAI are permitted based on comparing the Subscribed S-NSSAI(s)

		if param.Tai != nil &&
			!util.CheckSupportedNssaiInPlmn(param.SliceInfoRequestForRegistration.RequestedNssai, *param.Tai.PlmnId) {
			// Return ProblemDetails indicating S-NSSAI is not supported
			// TODO: Based on TS 23.501 V15.2.0, if the Requested NSSAI includes an S-NSSAI that is not valid in the
			//       Serving PLMN, the NSSF may derive the Configured NSSAI for Serving PLMN
			problemDetails := &models.ProblemDetails{
				Title:  util.UNSUPPORTED_RESOURCE,
				Status: http.StatusForbidden,
				Detail: "S-NSSAI in Requested NSSAI is not supported in PLMN",
				Cause:  "SNSSAI_NOT_SUPPORTED",
			}

			status = http.StatusForbidden
			return status, nil, problemDetails
		}

		// Check if any Requested S-NSSAIs is present in Subscribed S-NSSAIs
		checkIfRequestAllowed := false

		for _, requestedSnssai := range param.SliceInfoRequestForRegistration.RequestedNssai {
			if param.Tai != nil && !util.CheckSupportedSnssaiInTa(requestedSnssai, *param.Tai) {
				// Requested S-NSSAI does not supported in UE's current TA
				// Add it to Rejected NSSAI in TA
				authorizedNetworkSliceInfo.RejectedNssaiInTa = append(
					authorizedNetworkSliceInfo.RejectedNssaiInTa,
					requestedSnssai)
				continue
			}

			var mappingOfRequestedSnssai models.Snssai
			// TODO: Compared with Restricted S-NSSAI list in configuration under roaming scenario
			if param.HomePlmnId != nil && !util.CheckStandardSnssai(requestedSnssai) {
				// Standard S-NSSAIs are supported to be commonly decided by all roaming partners
				// Only non-standard S-NSSAIs are required to find mappings
				targetMapping, found := util.FindMappingWithServingSnssai(requestedSnssai,
					param.SliceInfoRequestForRegistration.MappingOfNssai)

				if !found {
					// No mapping of Requested S-NSSAI to HPLMN S-NSSAI is provided by UE
					// TODO: Search for local configuration if there is no provided mapping from UE, and update UE's
					//       Configured NSSAI
					checkInvalidRequestedNssai = true
					authorizedNetworkSliceInfo.RejectedNssaiInPlmn = append(
						authorizedNetworkSliceInfo.RejectedNssaiInPlmn,
						requestedSnssai)
					continue
				} else {
					// TODO: Check if mappings of S-NSSAIs are correct
					//       If not, update UE's Configured NSSAI
					mappingOfRequestedSnssai = *targetMapping.HomeSnssai
				}
			} else {
				mappingOfRequestedSnssai = requestedSnssai
			}

			hitSubscription := false
			for _, subscribedSnssai := range param.SliceInfoRequestForRegistration.SubscribedNssai {
				if openapi.SnssaiEqualFold(mappingOfRequestedSnssai, *subscribedSnssai.SubscribedSnssai) {
					// Requested S-NSSAI matches one of Subscribed S-NSSAI
					// Add it to Allowed NSSAI list
					hitSubscription = true

					var allowedSnssaiElement models.AllowedSnssai
					allowedSnssaiElement.AllowedSnssai = new(models.Snssai)
					*allowedSnssaiElement.AllowedSnssai = requestedSnssai
					nsiInformationList := util.GetNsiInformationListFromConfig(requestedSnssai)
					if nsiInformationList != nil {
						// TODO: `NsiInformationList` should be slice in `AllowedSnssai` instead of pointer of slice
						allowedSnssaiElement.NsiInformationList = append(
							allowedSnssaiElement.NsiInformationList,
							nsiInformationList...)
					}
					if param.HomePlmnId != nil && !util.CheckStandardSnssai(requestedSnssai) {
						allowedSnssaiElement.MappedHomeSnssai = new(models.Snssai)
						*allowedSnssaiElement.MappedHomeSnssai = *subscribedSnssai.SubscribedSnssai
					}

					// Default Access Type is set to 3GPP Access if no TAI is provided
					// TODO: Depend on operator implementation, it may also return S-NSSAIs in all valid Access Type if
					//       UE's Access Type could not be identified
					accessType := models.AccessType__3_GPP_ACCESS
					if param.Tai != nil {
						accessType = util.GetAccessTypeFromConfig(*param.Tai)
					}

					util.AddAllowedSnssai(allowedSnssaiElement, accessType, authorizedNetworkSliceInfo)

					checkIfRequestAllowed = true
					break
				}
			}

			if !hitSubscription {
				// Requested S-NSSAI does not match any Subscribed S-NSSAI
				// Add it to Rejected NSSAI in PLMN
				checkInvalidRequestedNssai = true
				authorizedNetworkSliceInfo.RejectedNssaiInPlmn = append(
					authorizedNetworkSliceInfo.RejectedNssaiInPlmn,
					requestedSnssai)
			}
		}

		if !checkIfRequestAllowed {
			// No S-NSSAI from Requested NSSAI is present in Subscribed S-NSSAIs
			// Subscribed S-NSSAIs marked as default are used
			useDefaultSubscribedSnssai(param, authorizedNetworkSliceInfo)
		}
	} else {
		// No Requested NSSAI is provided
		// Subscribed S-NSSAIs marked as default are used
		checkInvalidRequestedNssai = true
		useDefaultSubscribedSnssai(param, authorizedNetworkSliceInfo)
	}

	if param.Tai != nil &&
		!util.CheckAllowedNssaiInAmfTa(authorizedNetworkSliceInfo.AllowedNssaiList, param.NfId, *param.Tai) {
		util.AddAmfInformation(*param.Tai, authorizedNetworkSliceInfo)
	}

	if param.SliceInfoRequestForRegistration.DefaultConfiguredSnssaiInd {
		// Default Configured NSSAI Indication is received from AMF
		// Determine the Configured NSSAI based on the Default Configured NSSAI
		useDefaultConfiguredNssai(param, authorizedNetworkSliceInfo)
	} else if checkInvalidRequestedNssai {
		// No Requested NSSAI is provided or the Requested NSSAI includes an S-NSSAI that is not valid
		// Determine the Configured NSSAI based on the subscription
		// Configure available NSSAI for UE in its PLMN
		// If TAI is not provided, then unable to check if S-NSSAIs is supported in the PLMN
		if param.Tai != nil {
			setConfiguredNssai(param, authorizedNetworkSliceInfo)
		}
	}

	status = http.StatusOK
	return status, authorizedNetworkSliceInfo, nil
}

func selectNsiInformation(nsiInformationList []models.NsiInformation) models.NsiInformation {
	// TODO: Algorithm to select Network Slice Instance
	//       Take roaming indication into consideration

	// Randomly select a Network Slice Instance
	idx := rand.Intn(len(nsiInformationList))
	return nsiInformationList[idx]
}

// Network slice selection for PDU session
// The function is executed when the IE, `slice-info-for-pdu-session`, is provided in query parameters
func nsselectionForPduSession(param NetworkSliceInformationGetQuery) (
	int, *models.AuthorizedNetworkSliceInfo, *models.ProblemDetails,
) {
	var status int
	authorizedNetworkSliceInfo := &models.AuthorizedNetworkSliceInfo{}

	if param.HomePlmnId != nil {
		// Check whether UE's Home PLMN is supported when UE is a roamer
		if !util.CheckSupportedHplmn(*param.HomePlmnId) {
			authorizedNetworkSliceInfo.RejectedNssaiInPlmn = append(
				authorizedNetworkSliceInfo.RejectedNssaiInPlmn,
				*param.SliceInfoRequestForPduSession.SNssai)

			status = http.StatusOK
			return status, authorizedNetworkSliceInfo, nil
		}
	}

	if param.Tai != nil {
		// Check whether UE's current TA is supported when UE provides TAI
		if !util.CheckSupportedTa(*param.Tai) {
			authorizedNetworkSliceInfo.RejectedNssaiInTa = append(
				authorizedNetworkSliceInfo.RejectedNssaiInTa,
				*param.SliceInfoRequestForPduSession.SNssai)

			status = http.StatusOK
			return status, authorizedNetworkSliceInfo, nil
		}
	}

	if param.Tai != nil &&
		!util.CheckSupportedSnssaiInPlmn(*param.SliceInfoRequestForPduSession.SNssai, *param.Tai.PlmnId) {
		// Return ProblemDetails indicating S-NSSAI is not supported
		// TODO: Based on TS 23.501 V15.2.0, if the Requested NSSAI includes an S-NSSAI that is not valid in the
		//       Serving PLMN, the NSSF may derive the Configured NSSAI for Serving PLMN

		problemDetails := &models.ProblemDetails{
			Title:  util.UNSUPPORTED_RESOURCE,
			Status: http.StatusForbidden,
			Detail: "S-NSSAI in Requested NSSAI is not supported in PLMN",
			Cause:  "SNSSAI_NOT_SUPPORTED",
		}

		status = http.StatusForbidden
		return status, nil, problemDetails
	}

	if param.HomePlmnId != nil {
		if param.SliceInfoRequestForPduSession.RoamingIndication == models.RoamingIndication_NON_ROAMING {
			detail := "`home-plmn-id` is provided, which contradicts `roamingIndication`:'NON_ROAMING'"
			problemDetails := &models.ProblemDetails{
				Title:  util.INVALID_REQUEST,
				Status: http.StatusBadRequest,
				Detail: detail,
				InvalidParams: []models.InvalidParam{
					{
						Param:  "home-plmn-id",
						Reason: detail,
					},
				},
			}

			status = http.StatusBadRequest
			return status, nil, problemDetails
		}
	} else {
		if param.SliceInfoRequestForPduSession.RoamingIndication != models.RoamingIndication_NON_ROAMING {
			detail := fmt.Sprintf("`home-plmn-id` is not provided, which contradicts `roamingIndication`:'%s'",
				string(param.SliceInfoRequestForPduSession.RoamingIndication))
			problemDetails := &models.ProblemDetails{
				Title:  util.INVALID_REQUEST,
				Status: http.StatusBadRequest,
				Detail: detail,
				InvalidParams: []models.InvalidParam{
					{
						Param:  "home-plmn-id",
						Reason: detail,
					},
				},
			}

			status = http.StatusBadRequest
			return status, nil, problemDetails
		}
	}

	if param.Tai != nil && !util.CheckSupportedSnssaiInTa(*param.SliceInfoRequestForPduSession.SNssai, *param.Tai) {
		// Requested S-NSSAI does not supported in UE's current TA
		// Add it to Rejected NSSAI in TA
		authorizedNetworkSliceInfo.RejectedNssaiInTa = append(
			authorizedNetworkSliceInfo.RejectedNssaiInTa,
			*param.SliceInfoRequestForPduSession.SNssai)
		status = http.StatusOK
		return status, authorizedNetworkSliceInfo, nil
	}

	nsiInformationList := util.GetNsiInformationListFromConfig(*param.SliceInfoRequestForPduSession.SNssai)

	if len(nsiInformationList) == 0 {
		*authorizedNetworkSliceInfo = models.AuthorizedNetworkSliceInfo{}
	} else {
		nsiInformation := selectNsiInformation(nsiInformationList)
		authorizedNetworkSliceInfo.NsiInformation = new(models.NsiInformation)
		*authorizedNetworkSliceInfo.NsiInformation = nsiInformation
	}

	logger.NsselLog.Infof("authorizedNetworkSliceInfo: %+v", authorizedNetworkSliceInfo)

	return http.StatusOK, authorizedNetworkSliceInfo, nil
}
