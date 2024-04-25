package server

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
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
			s.Processor().HandleNSSAIAvailabilityDelete,
		},

		{
			"NSSAIAvailabilityPatch",
			strings.ToUpper("Patch"),
			"/nssai-availability/:nfId",
			s.Processor().HandleNSSAIAvailabilityPatch,
		},

		{
			"NSSAIAvailabilityPut",
			strings.ToUpper("Put"),
			"/nssai-availability/:nfId",
			s.Processor().HandleNSSAIAvailabilityPut,
		},

		// Regular expressions for route matching should be unique in Gin package
		// 'subscriptions' would conflict with existing wildcard ':nfId'
		// Simply replace 'subscriptions' with ':nfId' and check if ':nfId' is 'subscriptions' in handler function
		{
			"NSSAIAvailabilityUnsubscribe",
			strings.ToUpper("Delete"),
			// "/nssai-availability/subscriptions/:subscriptionId",
			"/nssai-availability/:nfId/:subscriptionId",
			s.Processor().HTTPNSSAIAvailabilityUnsubscribe,
		},

		{
			"NSSAIAvailabilityPost",
			strings.ToUpper("Post"),
			"/nssai-availability/subscriptions",
			s.Processor().HTTPNSSAIAvailabilityPost,
		},
	}
}
