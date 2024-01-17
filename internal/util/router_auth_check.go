package util

import (
	nssf_context "github.com/free5gc/nssf/internal/context"
	"github.com/free5gc/nssf/internal/logger"
	"github.com/gin-gonic/gin"
	"net/http"
)

type NFContextGetter func() *nssf_context.NSSFContext

type RouterAuthorizationCheck struct {
	serviceName string
}

func NewRouterAuthorizationCheck(serviceName string) *RouterAuthorizationCheck {
	return &RouterAuthorizationCheck{
		serviceName: serviceName,
	}
}

func (rac *RouterAuthorizationCheck) Check(c *gin.Context, nssfContext nssf_context.NFContext) {
	token := c.Request.Header.Get("Authorization")
	err := nssfContext.AuthorizationCheck(token, rac.serviceName)

	if err != nil {
		logger.UtilLog.Debugf("RouterAuthorizationCheck: Check Unauthorized: %s", err)
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		c.Abort()
		return
	}

	logger.UtilLog.Debugf("RouterAuthorizationCheck: Check Authorized")
}
