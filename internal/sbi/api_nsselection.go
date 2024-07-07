package sbi

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"

	"github.com/free5gc/nssf/internal/logger"
	"github.com/free5gc/nssf/internal/sbi/processor"
	"github.com/free5gc/nssf/internal/util"
	"github.com/free5gc/openapi/models"
)

func (s *Server) getNsSelectionRoutes() []Route {
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
			"NSSelectionGet",
			strings.ToUpper("Get"),
			"/network-slice-information",
			s.NetworkSliceInformationGet,
		},
	}
}

func BindErrorInvalidParams(err error) []models.InvalidParam {
	var invalidParams []models.InvalidParam
	if errs, ok := err.(validator.ValidationErrors); ok {
		for _, e := range errs {
			ip := models.InvalidParam{
				Param: e.Field(),
			}

			switch e.Tag() {
			case "required":
				ip.Reason = fmt.Sprintf("The `%s` field is required.", e.Field())
			case "oneof":
				ip.Reason = fmt.Sprintf("The `%s` field must be one of '%s'.", e.Field(), e.Param())
			case "required_with":
				ip.Reason = fmt.Sprintf("The `%s` field is required when `%s` is present.", e.Field(), e.Param())
			case "required_without":
				ip.Reason = fmt.Sprintf("The `%s` field is required when `%s` is not present.", e.Field(), e.Param())
			default:
				ip.Reason = fmt.Sprintf("Failed on the `%s` tag.", e.Tag())
			}

			invalidParams = append(invalidParams, ip)
		}
	} else {
		logger.NsselLog.Errorf("Unknown error type: %+v", err)
	}

	return invalidParams
}

func (s *Server) NetworkSliceInformationGet(c *gin.Context) {
	logger.NsselLog.Infof("Handle NSSelectionGet")

	var query processor.NetworkSliceInformationGetQuery
	if err := c.ShouldBindQuery(&query); err != nil {
		logger.NsselLog.Errorf("BindQuery failed: %+v", err)
		problemDetail := &models.ProblemDetails{
			Title:         "Malformed Request",
			Status:        http.StatusBadRequest,
			Detail:        err.Error(),
			Instance:      "",
			InvalidParams: BindErrorInvalidParams(err),
		}
		util.GinProblemJson(c, problemDetail)
		return
	}

	// query := c.Request.URL.Query()
	s.Processor().NSSelectionSliceInformationGet(c, query)
}
