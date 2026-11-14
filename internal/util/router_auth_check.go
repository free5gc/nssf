package util

import (
	"net/http"

	"github.com/gin-gonic/gin"

	nssf_context "github.com/free5gc/nssf/internal/context"
	"github.com/free5gc/nssf/internal/logger"
	"github.com/free5gc/openapi/models"
)

type RouterAuthorizationCheck struct {
	serviceName models.Nrf_NFMgmt_ServiceName
}

func NewRouterAuthorizationCheck(serviceName models.Nrf_NFMgmt_ServiceName) *RouterAuthorizationCheck {
	return &RouterAuthorizationCheck{
		serviceName: serviceName,
	}
}

func (rac *RouterAuthorizationCheck) Check(
	c *gin.Context,
	nssfContext nssf_context.NFContext,
) {
	token := c.Request.Header.Get("Authorization")

	// Step 1: Validate the existing OAuth2 authorization.
	err := nssfContext.AuthorizationCheck(token, rac.serviceName)
	if err != nil {
		logger.UtilLog.Debugf(
			"RouterAuthorizationCheck: Unauthorized: %s",
			err.Error(),
		)
		problemDetails := &models.ProblemDetails{
			Status: http.StatusUnauthorized,
			Detail: "Unauthorized",
		}
		GinProblemJson(c, problemDetails)
		c.Abort()
		return
	}

	// Step 2: Obtain the authenticated caller identity
	// for NSSAI Availability write operations.
	if rac.serviceName == models.Nrf_NFMgmt_ServiceName_NNSSF_NSSAIAVAILABILITY {
		switch c.Request.Method {
		case http.MethodPut, http.MethodPatch,
			http.MethodDelete, http.MethodPost:

			if !nssfContext.IsOAuth2Required() {
				c.Set("nssfSkipOwnershipCheck", true)
				break
			}

			callerID, err := nssfContext.VerifiedCallerNFID(
				token,
				rac.serviceName,
			)
			if err != nil {
				logger.UtilLog.Debugf(
					"Failed to verify NF identity: %s",
					err.Error(),
				)
				problemDetails := &models.ProblemDetails{
					Status: http.StatusUnauthorized,
					Detail: "Verified NF identity required",
				}
				GinProblemJson(c, problemDetails)
				c.Abort()
				return
			}

			// Only the authentication middleware sets this value.
			c.Set("nssfVerifiedCallerNfID", callerID)
		}
	}

	logger.UtilLog.Debugf(
		"RouterAuthorizationCheck: Check Authorized",
	)
}
