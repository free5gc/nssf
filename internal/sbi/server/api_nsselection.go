package server

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

func (s *Server) getNsSelectionRoutes() []Route {
	return []Route{
		{
			"Helth Check",
			strings.ToUpper("Get"),
			"/",
			func(ctx *gin.Context) {
				ctx.JSON(http.StatusOK, gin.H{"status": "Service Available"})
			},
		},

		{
			"NSSelectionGet",
			strings.ToUpper("Get"),
			"/network-slice-information",
			s.Processor().HandleNetworkSliceInformationGet,
		},
	}
}
