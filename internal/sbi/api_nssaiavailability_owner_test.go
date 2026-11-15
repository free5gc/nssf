package sbi

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"

	"github.com/free5gc/nssf/internal/sbi/processor"
	"github.com/free5gc/nssf/pkg/factory"
	"github.com/free5gc/openapi/models"
)

func TestNSSAIAvailabilityWriteRejectsCrossNF(t *testing.T) {
	gin.SetMode(gin.TestMode)

	const (
		amfA = "11111111-1111-4111-8111-111111111111"
		amfB = "22222222-2222-4222-8222-222222222222"
	)

	tests := []struct {
		name   string
		method string
		body   string
	}{
		{
			name:   "PUT",
			method: http.MethodPut,
			body:   `{"supportedNssaiAvailabilityData":[]}`,
		},
		{
			name:   "PATCH",
			method: http.MethodPatch,
			body:   `[]`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			oldConfig := factory.NssfConfig

			factory.NssfConfig = &factory.Config{
				Configuration: &factory.Configuration{
					AmfList: []factory.AmfConfig{
						{NfId: amfB},
					},
				},
			}

			t.Cleanup(func() {
				factory.NssfConfig = oldConfig
			})

			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)

			c.Request = httptest.NewRequestWithContext(
				context.Background(),
				tt.method,
				"/nnssf-nssaiavailability/v1/nssai-availability/"+amfB,
				strings.NewReader(tt.body),
			)

			c.Request.Header.Set(
				"Content-Type", "application/json",
			)

			c.Params = gin.Params{
				{Key: "nfId", Value: amfB},
			}

			// Simulate authenticated AMF-A.
			c.Set("nssfVerifiedCallerNfID", amfA)

			server := &Server{
				processor: processor.NewProcessor(nil),
			}

			switch tt.method {
			case http.MethodPut:
				server.NSSAIAvailabilityPut(c)
			case http.MethodPatch:
				server.NSSAIAvailabilityPatch(c)
			}

			// Cross-NF modification must be forbidden.
			if w.Code != http.StatusForbidden {
				t.Fatalf("expected HTTP 403, got %d", w.Code)
			}

			var problemDetails models.ProblemDetails

			if err := json.Unmarshal(w.Body.Bytes(), &problemDetails); err != nil {
				t.Fatalf("invalid ProblemDetails JSON: %v", err)
			}

			if problemDetails.Status != http.StatusForbidden {
				t.Errorf("expected status 403, got %d", problemDetails.Status)
			}

			if problemDetails.Cause != "NOT_AUTHORIZED" {
				t.Errorf("expected NOT_AUTHORIZED, got %q", problemDetails.Cause)
			}

			if problemDetails.Detail == "" {
				t.Error("expected non-empty ProblemDetails detail")
			}

			// AMF-B's record must remain unchanged.
			amfList := factory.NssfConfig.Configuration.AmfList

			if len(amfList) != 1 || amfList[0].NfId != amfB {
				t.Error("AMF-B record was modified or removed")
			}
		})
	}
}

func TestNSSAIAvailabilityMissingVerifiedCaller(t *testing.T) {
	gin.SetMode(gin.TestMode)

	const nfID = "11111111-1111-4111-8111-111111111111"

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	c.Request = httptest.NewRequestWithContext(
		context.Background(),
		http.MethodDelete,
		"/nnssf-nssaiavailability/v1/nssai-availability/"+nfID,
		nil,
	)

	c.Params = gin.Params{
		{Key: "nfId", Value: nfID},
	}

	// Simulate a handler request without a verified caller identity.
	// The OAuth2-disabled skip flag is intentionally absent.
	server := &Server{}
	server.NSSAIAvailabilityDelete(c)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected HTTP 401, got %d", w.Code)
	}

	var problemDetails models.ProblemDetails

	if err := json.Unmarshal(w.Body.Bytes(), &problemDetails); err != nil {
		t.Fatalf("invalid ProblemDetails JSON: %v", err)
	}

	if problemDetails.Status != http.StatusUnauthorized {
		t.Errorf("expected status 401, got %d", problemDetails.Status)
	}

	if problemDetails.Detail == "" {
		t.Error("expected non-empty ProblemDetails detail")
	}

	if !c.IsAborted() {
		t.Error("request was not aborted")
	}
}
