/*
 * NSSF NSSAI Availability
 *
 * NSSF NSSAI Availability Service
 */

package processor

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"reflect"

	jsonpatch "github.com/evanphx/json-patch"
	"github.com/gin-gonic/gin"

	"github.com/free5gc/nssf/internal/logger"
	"github.com/free5gc/nssf/internal/plugin"
	"github.com/free5gc/nssf/internal/util"
	"github.com/free5gc/nssf/pkg/factory"
	"github.com/free5gc/openapi/models"
	"github.com/free5gc/util/metrics/sbi"
)

func validateSupportedNssaiAvailabilityDataList(
	c *gin.Context,
	supportedNssaiAvailabilityData []models.Nssf_NSSAIAvail_SupportedNssaiAvailabilityData,
) bool {
	for _, s := range supportedNssaiAvailabilityData {
		if s.Tai == nil || s.Tai.PlmnId == nil {
			problemDetails := &models.ProblemDetails{
				Title:  util.MANDATORY_IE_MISSING,
				Status: http.StatusBadRequest,
				Detail: "tai or tai.plmnId is missing in supportedNssaiAvailabilityData",
			}
			c.Set(sbi.IN_PB_DETAILS_CTX_STR, problemDetails.Title)
			util.GinProblemJson(c, problemDetails)
			return false
		}

		if !util.CheckSupportedNssaiInPlmn(s.SupportedSnssaiList, *s.Tai.PlmnId) {
			problemDetails := &models.ProblemDetails{
				Title:  util.UNSUPPORTED_RESOURCE,
				Status: http.StatusForbidden,
				Detail: "S-NSSAI in Requested NSSAI is not supported in PLMN",
				Cause:  "SNSSAI_NOT_SUPPORTED",
			}
			c.Set(sbi.IN_PB_DETAILS_CTX_STR, problemDetails.Cause)
			util.GinProblemJson(c, problemDetails)
			return false
		}
	}

	return true
}

func (p *Processor) NssaiAvailabilityNfInstanceDelete(c *gin.Context, nfId string) {
	var problemDetails *models.ProblemDetails

	factory.NssfConfig.Lock()
	for i, amfConfig := range factory.NssfConfig.Configuration.AmfList {
		if amfConfig.NfId == nfId {
			factory.NssfConfig.Configuration.AmfList = append(
				factory.NssfConfig.Configuration.AmfList[:i],
				factory.NssfConfig.Configuration.AmfList[i+1:]...)

			factory.NssfConfig.Unlock()
			c.Status(http.StatusNoContent)
			return
		}
	}

	factory.NssfConfig.Unlock()
	problemDetails = &models.ProblemDetails{
		Title:  util.UNSUPPORTED_RESOURCE,
		Status: http.StatusNotFound,
		Detail: fmt.Sprintf("AMF ID '%s' does not exist", nfId),
	}
	c.Set(sbi.IN_PB_DETAILS_CTX_STR, problemDetails.Title)
	util.GinProblemJson(c, problemDetails)
}

func (p *Processor) NssaiAvailabilityNfInstancePatch(
	c *gin.Context,
	nssaiAvailabilityUpdateInfo plugin.PatchDocument, nfId string,
) {
	response := &models.Nssf_NSSAIAvail_AuthorizedNssaiAvailabilityInfo{}
	var updatedSupportedNssaiAvailabilityData []models.Nssf_NSSAIAvail_SupportedNssaiAvailabilityData
	var problemDetails *models.ProblemDetails

	// Protect the entire read-modify-write operation.
	func() {
		factory.NssfConfig.Lock()
		defer factory.NssfConfig.Unlock()

		amfIdx := -1
		for i, amfConfig := range factory.NssfConfig.Configuration.AmfList {
			if amfConfig.NfId == nfId {
				amfIdx = i
				break
			}
		}

		if amfIdx == -1 {
			problemDetails = &models.ProblemDetails{
				Title:  util.UNSUPPORTED_RESOURCE,
				Status: http.StatusNotFound,
				Detail: fmt.Sprintf("AMF ID '%s' does not exist", nfId),
			}
			return
		}

		// Make a copy to avoid modifying shared data during JSON processing.
		data := factory.NssfConfig.Configuration.AmfList[amfIdx].SupportedNssaiAvailabilityData

		var temp []models.Nssf_NSSAIAvail_SupportedNssaiAvailabilityData
		if data != nil {
			temp = make([]models.Nssf_NSSAIAvail_SupportedNssaiAvailabilityData, len(data))
			copy(temp, data)
		}

		for i := range temp {
			temp[i].SupportedSnssaiList = append(
				[]models.ExtSnssai(nil),
				data[i].SupportedSnssaiList...,
			)
		}

		// Preserve the existing handling of optional SD fields.
		const dummyString = "DUMMY"
		for i := range temp {
			for j := range temp[i].SupportedSnssaiList {
				if temp[i].SupportedSnssaiList[j].Sd == "" {
					temp[i].SupportedSnssaiList[j].Sd = dummyString
				}
			}
		}

		original, err := json.Marshal(temp)
		if err != nil {
			logger.NssaiavailLog.Errorf(
				"Marshal error in NSSAIAvailabilityPatchProcedure: %+v", err,
			)
			problemDetails = &models.ProblemDetails{
				Title:  util.INTERNAL_ERROR,
				Status: http.StatusInternalServerError,
				Detail: "Failed to encode NSSAI availability data",
			}
			return
		}
		original = bytes.ReplaceAll(original, []byte(dummyString), []byte(""))

		// Preserve existing JSON Patch input processing.
		for i, patchItem := range nssaiAvailabilityUpdateInfo {
			if reflect.ValueOf(patchItem.Value).Kind() == reflect.Map {
				_, exist := patchItem.Value.(map[string]interface{})["sst"]
				_, notExist := patchItem.Value.(map[string]interface{})["sd"]
				if exist && !notExist {
					nssaiAvailabilityUpdateInfo[i].Value.(map[string]interface{})["sd"] = ""
				}
			}
		}

		patchJSON, err := json.Marshal(nssaiAvailabilityUpdateInfo)
		if err != nil {
			logger.NssaiavailLog.Errorf(
				"Marshal error in NSSAIAvailabilityPatchProcedure: %+v", err,
			)
			problemDetails = &models.ProblemDetails{
				Title:  util.MALFORMED_REQUEST,
				Status: http.StatusBadRequest,
				Detail: err.Error(),
			}
			return
		}

		patch, err := jsonpatch.DecodePatch(patchJSON)
		if err != nil {
			problemDetails = &models.ProblemDetails{
				Title:  util.MALFORMED_REQUEST,
				Status: http.StatusBadRequest,
				Detail: err.Error(),
			}
			return
		}

		modified, err := patch.Apply(original)
		if err != nil {
			problemDetails = &models.ProblemDetails{
				Title:  util.INVALID_REQUEST,
				Status: http.StatusConflict,
				Detail: err.Error(),
			}
			return
		}

		err = json.Unmarshal(modified, &updatedSupportedNssaiAvailabilityData)
		if err != nil {
			problemDetails = &models.ProblemDetails{
				Title:  util.INVALID_REQUEST,
				Status: http.StatusBadRequest,
				Detail: err.Error(),
			}
			return
		}

		// Validate while holding the write lock.
		// Use the lock-aware helper to avoid acquiring RLock again.
		for _, s := range updatedSupportedNssaiAvailabilityData {
			if s.Tai == nil || s.Tai.PlmnId == nil {
				problemDetails = &models.ProblemDetails{
					Title:  util.MANDATORY_IE_MISSING,
					Status: http.StatusBadRequest,
					Detail: "tai or tai.plmnId is missing in supportedNssaiAvailabilityData",
				}
				return
			}

			if !util.CheckSupportedNssaiInPlmnLocked(
				s.SupportedSnssaiList, *s.Tai.PlmnId,
			) {
				problemDetails = &models.ProblemDetails{
					Title:  util.UNSUPPORTED_RESOURCE,
					Status: http.StatusForbidden,
					Detail: "S-NSSAI in Requested NSSAI is not supported in PLMN",
					Cause:  "SNSSAI_NOT_SUPPORTED",
				}
				return
			}
		}

		// Commit the update before releasing the lock.
		factory.NssfConfig.Configuration.AmfList[amfIdx].
			SupportedNssaiAvailabilityData = updatedSupportedNssaiAvailabilityData
	}()

	// Send errors after releasing the lock.
	if problemDetails != nil {
		value := problemDetails.Title
		if problemDetails.Cause != "" {
			value = problemDetails.Cause
		}
		c.Set(sbi.IN_PB_DETAILS_CTX_STR, value)
		util.GinProblemJson(c, problemDetails)
		return
	}

	// Build the response from the data committed by this PATCH.
	// A concurrent DELETE cannot change this request's response snapshot.
	for _, s := range updatedSupportedNssaiAvailabilityData {
		tai := *s.Tai

		response.AuthorizedNssaiAvailabilityData = append(
			response.AuthorizedNssaiAvailabilityData,
			models.Nssf_NSSAIAvail_AuthorizedNssaiAvailabilityData{
				Tai:                  &tai,
				SupportedSnssaiList:  s.SupportedSnssaiList,
				RestrictedSnssaiList: util.GetRestrictedSnssaiListFromConfig(tai),
			},
		)
	}

	c.JSON(http.StatusOK, response)
}

// NSSAIAvailability PUT method
func (p *Processor) NssaiAvailabilityNfInstanceUpdate(
	c *gin.Context,
	nssaiAvailabilityInfo models.Nssf_NSSAIAvail_NssaiAvailabilityInfo, nfId string,
) {
	var (
		response = &models.Nssf_NSSAIAvail_AuthorizedNssaiAvailabilityInfo{}
		hitAmf   bool
	)

	if !validateSupportedNssaiAvailabilityDataList(c, nssaiAvailabilityInfo.SupportedNssaiAvailabilityData) {
		return
	}

	// TODO: Currently authorize all the provided S-NSSAIs
	//       Take some issue into consideration e.g. operator policies

	// Find AMF configuration of given NfId
	// If found, then update the SupportedNssaiAvailabilityData
	factory.NssfConfig.Lock()

	for i, amfConfig := range factory.NssfConfig.Configuration.AmfList {
		if amfConfig.NfId == nfId {
			factory.NssfConfig.Configuration.AmfList[i].SupportedNssaiAvailabilityData = nssaiAvailabilityInfo.
				SupportedNssaiAvailabilityData

			hitAmf = true
			break
		}
	}
	// If no AMF record is found, create a new one
	if !hitAmf {
		var amfConfig factory.AmfConfig
		amfConfig.NfId = nfId
		amfConfig.SupportedNssaiAvailabilityData = nssaiAvailabilityInfo.SupportedNssaiAvailabilityData

		factory.NssfConfig.Configuration.AmfList = append(
			factory.NssfConfig.Configuration.AmfList,
			amfConfig)
	}

	factory.NssfConfig.Unlock()

	// Return all authorized NSSAI availability information
	// a.AuthorizedNssaiAvailabilityData, _ = authorizeOfAmfFromConfig(nfId)

	// Return authorized NSSAI availability information of updated TAI only
	for _, s := range nssaiAvailabilityInfo.SupportedNssaiAvailabilityData {
		tai := *s.Tai

		response.AuthorizedNssaiAvailabilityData = append(
			response.AuthorizedNssaiAvailabilityData,
			models.Nssf_NSSAIAvail_AuthorizedNssaiAvailabilityData{
				Tai:                  &tai,
				SupportedSnssaiList:  s.SupportedSnssaiList,
				RestrictedSnssaiList: util.GetRestrictedSnssaiListFromConfig(tai),
			},
		)
	}

	c.JSON(http.StatusOK, response)
}
