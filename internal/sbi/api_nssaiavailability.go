package sbi

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/free5gc/nssf/internal/logger"
	"github.com/free5gc/nssf/internal/plugin"
	"github.com/free5gc/nssf/internal/util"
	"github.com/free5gc/openapi"
	"github.com/free5gc/openapi/models"
)

func (s *Server) getNssaiAvailabilityRoutes() []Route {
	return []Route{
		{
			"Health Check",
			strings.ToUpper("Get"),
			"/",
			func(ctx *gin.Context) {
				ctx.JSON(http.StatusOK, gin.H{"status": "Service Available"})
			},
		},

		{
			"NSSAIAvailabilityDelete",
			strings.ToUpper("Delete"),
			"/nssai-availability/:nfId",
			s.NSSAIAvailabilityDelete,
		},

		{
			"NSSAIAvailabilityPatch",
			strings.ToUpper("Patch"),
			"/nssai-availability/:nfId",
			s.NSSAIAvailabilityPatch,
		},

		{
			"NSSAIAvailabilityPut",
			strings.ToUpper("Put"),
			"/nssai-availability/:nfId",
			s.NSSAIAvailabilityPut,
		},

		// Regular expressions for route matching should be unique in Gin package
		// 'subscriptions' would conflict with existing wildcard ':nfId'
		// Simply replace 'subscriptions' with ':nfId' and check if ':nfId' is 'subscriptions' in handler function
		{
			"NSSAIAvailabilityUnsubscribe",
			strings.ToUpper("Delete"),
			// "/nssai-availability/subscriptions/:subscriptionId",
			"/nssai-availability/:nfId/:subscriptionId",
			s.NSSAIAvailabilityUnsubscribeDelete,
		},

		{
			"NSSAIAvailabilityPost",
			strings.ToUpper("Post"),
			"/nssai-availability/subscriptions",
			s.NSSAIAvailabilityPost,
		},

		{
			"NSSAIAvailabilityPatchSubscriptions",
			http.MethodPatch,
			"/nssai-availability/subscriptions/:subscriptionId",
			s.NSSAIAvailabilitySubscriptionPatch,
		},

		{
			"NSSAIAvailabilityDiscoverOptions",
			http.MethodOptions,
			"/nssai-availability",
			s.NSSAIAvailabilityOptions,
		},
	}
}

// NSSAIAvailabilityDelete - Deletes an already existing S-NSSAIs per TA
// provided by the NF service consumer (e.g AMF)
func (s *Server) NSSAIAvailabilityDelete(c *gin.Context) {
	logger.NssaiavailLog.Infof("Handle NSSAIAvailabilityDelete")

	nfId := c.Params.ByName("nfId")

	if nfId == "" {
		problemDetails := &models.ProblemDetails{
			Status: http.StatusBadRequest,
			Cause:  "UNSPECIFIED", // TODO: Check if this is the correct cause
		}

		util.GinProblemJson(c, problemDetails)
		return
	}

	s.Processor().NssaiAvailabilityNfInstanceDelete(c, nfId)
}

// NSSAIAvailabilityPatch - Updates an already existing S-NSSAIs per TA
// provided by the NF service consumer (e.g AMF)
func (s *Server) NSSAIAvailabilityPatch(c *gin.Context) {
	logger.NssaiavailLog.Infof("Handle NSSAIAvailabilityPatch")

	nfId := c.Params.ByName("nfId")

	if nfId == "" {
		problemDetails := &models.ProblemDetails{
			Status: http.StatusBadRequest,
			Cause:  "UNSPECIFIED", // TODO: Check if this is the correct cause
		}

		util.GinProblemJson(c, problemDetails)
		return
	}

	var patchDocument plugin.PatchDocument

	requestBody, err := c.GetRawData()
	if err != nil {
		problemDetails := &models.ProblemDetails{
			Status: http.StatusInternalServerError,
			Cause:  "SYSTEM_FAILURE",
		}

		util.GinProblemJson(c, problemDetails)
		return
	}

	if err = openapi.Deserialize(&patchDocument, requestBody, "application/json"); err != nil {
		problemDetails := &models.ProblemDetails{
			Status: http.StatusBadRequest,
			Cause:  "UNSPECIFIED", // TODO: Check if this is the correct cause
		}

		logger.SBILog.Errorf("Error deserializing patch document: %+v", err)
		util.GinProblemJson(c, problemDetails)
		return
	}

	// TODO: Request NfProfile of NfId from NRF
	//       Check if NfId is valid AMF and obtain AMF Set ID
	//       If NfId is invalid, return ProblemDetails with code 404 Not Found
	//       If NF consumer is not authorized to update NSSAI availability, return ProblemDetails with code 403 Forbidden

	s.Processor().NssaiAvailabilityNfInstancePatch(c, patchDocument, nfId)
}

type NssaiAvailabilityPutParams struct {
	NfId string `uri:"nfId" binding:"required,uuid"`
}

// NSSAIAvailabilityPut - Updates/replaces the NSSF
// with the S-NSSAIs the NF service consumer (e.g AMF) supports per TA
func (s *Server) NSSAIAvailabilityPut(c *gin.Context) {
	logger.NssaiavailLog.Infof("Handle NSSAIAvailabilityPut")

	// nfId := c.Params.ByName("nfId")
	var params NssaiAvailabilityPutParams
	if err := c.ShouldBindUri(&params); err != nil {
		problemDetails := &models.ProblemDetails{
			Title:         "Malformed Request",
			Status:        http.StatusBadRequest,
			Cause:         "MALFORMED_REQUEST",
			InvalidParams: util.BindErrorInvalidParamsMessages(err),
		}

		util.GinProblemJson(c, problemDetails)
		return
	}

	var nssaiAvailabilityInfo models.NssaiAvailabilityInfo
	data, err := c.GetRawData()
	if err != nil {
		problemDetails := &models.ProblemDetails{
			Status: http.StatusInternalServerError,
			Cause:  "SYSTEM_FAILURE",
		}

		util.GinProblemJson(c, problemDetails)
		return
	}

	if err = openapi.Deserialize(&nssaiAvailabilityInfo, data, "application/json"); err != nil {
		problemDetails := &models.ProblemDetails{
			Status: http.StatusBadRequest,
			Cause:  "UNSPECIFIED", // TODO: Check if this is the correct cause
		}

		logger.SBILog.Errorf("Error deserializing NSSAI availability info: %+v", err)
		util.GinProblemJson(c, problemDetails)
		return
	}

	s.Processor().NssaiAvailabilityNfInstanceUpdate(c, nssaiAvailabilityInfo, params.NfId)
}

func (s *Server) NSSAIAvailabilitySubscriptionPatch(c *gin.Context) {
	c.Status(http.StatusNotImplemented)
}

func (s *Server) NSSAIAvailabilityPost(c *gin.Context) {
	var createData models.NssfEventSubscriptionCreateData

	requestBody, err := c.GetRawData()
	if err != nil {
		problemDetail := &models.ProblemDetails{
			Title:  "System failure",
			Status: http.StatusInternalServerError,
			Detail: err.Error(),
			Cause:  "SYSTEM_FAILURE",
		}
		logger.NssaiavailLog.Errorf("Get Request Body error: %+v", err)

		util.GinProblemJson(c, problemDetail)
		return
	}

	err = openapi.Deserialize(&createData, requestBody, "application/json")
	if err != nil {
		problemDetail := "[Request Body] " + err.Error()
		rsp := &models.ProblemDetails{
			Title:  "Malformed request syntax",
			Status: http.StatusBadRequest,
			Detail: problemDetail,
		}
		logger.NssaiavailLog.Errorln(problemDetail)

		util.GinProblemJson(c, rsp)
		return
	}

	s.Processor().NssaiAvailabilitySubscriptionCreate(c, createData)
}

func (s *Server) NSSAIAvailabilityOptions(c *gin.Context) {
	c.Status(http.StatusNotImplemented)
}

func (s *Server) NSSAIAvailabilityUnsubscribeDelete(c *gin.Context) {
	// Due to conflict of route matching, 'subscriptions' in the route is replaced with the existing wildcard ':nfId'
	nfID := c.Param("nfId")
	if nfID != "subscriptions" {
		c.JSON(http.StatusNotFound, gin.H{})
		logger.NssaiavailLog.Infof("404 Not Found")
		return
	}

	subscriptionId := c.Params.ByName("subscriptionId")
	if subscriptionId == "" {
		problemDetails := &models.ProblemDetails{
			Status: http.StatusBadRequest,
			Cause:  "UNSPECIFIED", // TODO: Check if this is the correct cause
		}

		util.GinProblemJson(c, problemDetails)
		return
	}

	s.Processor().NssaiAvailabilitySubscriptionUnsubscribe(c, subscriptionId)
}
