package util_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/pkg/errors"

	"github.com/free5gc/nssf/internal/util"
	"github.com/free5gc/openapi/models"
)

const (
	Valid   = "valid"
	Invalid = "invalid"
)

type mockNSSFContext struct{}

func newMockNSSFContext() *mockNSSFContext {
	return &mockNSSFContext{}
}

func (m *mockNSSFContext) AuthorizationCheck(token string, serviceName models.Nrf_NFMgmt_ServiceName) error {
	if token == Valid {
		return nil
	}

	return errors.New("invalid token")
}

func (m *mockNSSFContext) VerifiedCallerNFID(
	token string,
	serviceName models.Nrf_NFMgmt_ServiceName,
) (string, error) {
	if token == Valid {
		return "11111111-1111-4111-8111-111111111111", nil
	}

	return "", errors.New("invalid NF identity")
}

func TestRouterAuthorizationCheck_Check(t *testing.T) {
	// Mock gin.Context
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	var err error
	c.Request, err = http.NewRequestWithContext(context.Background(), "GET", "/", nil)
	if err != nil {
		t.Errorf("error on http request: %+v", err)
	}

	type Args struct {
		token string
	}
	type Want struct {
		statusCode int
	}

	tests := []struct {
		name string
		args Args
		want Want
	}{
		{
			name: "Valid Token",
			args: Args{
				token: Valid,
			},
			want: Want{
				statusCode: http.StatusOK,
			},
		},
		{
			name: "Invalid Token",
			args: Args{
				token: Invalid,
			},
			want: Want{
				statusCode: http.StatusUnauthorized,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w = httptest.NewRecorder()
			c, _ = gin.CreateTestContext(w)
			c.Request, err = http.NewRequestWithContext(context.Background(), "GET", "/", nil)
			if err != nil {
				t.Errorf("error on http request: %+v", err)
			}
			c.Request.Header.Set("Authorization", tt.args.token)

			var testService models.Nrf_NFMgmt_ServiceName = "testService"

			rac := util.NewRouterAuthorizationCheck(testService)
			rac.Check(c, newMockNSSFContext())
			if w.Code != tt.want.statusCode {
				t.Errorf("StatusCode should be %d, but got %d", tt.want.statusCode, w.Code)
			}
		})
	}
}

func TestNSSAIAvailabilityMiddlewareVerifiedCaller(t *testing.T) {
	gin.SetMode(gin.TestMode)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	c.Request = httptest.NewRequestWithContext(
		context.Background(),
		http.MethodDelete,
		"/nssai-availability/22222222-2222-4222-8222-222222222222",
		nil,
	)
	c.Request.Header.Set("Authorization", Valid)

	rac := util.NewRouterAuthorizationCheck(
		models.Nrf_NFMgmt_ServiceName_NNSSF_NSSAIAVAILABILITY,
	)

	rac.Check(c, newMockNSSFContext())

	if c.IsAborted() {
		t.Fatal("valid caller was rejected")
	}

	callerID, exists := c.Get("nssfVerifiedCallerNfID")
	if !exists {
		t.Fatal("verified NF identity not found in Gin Context")
	}

	expected := "11111111-1111-4111-8111-111111111111"
	if callerID != expected {
		t.Errorf("expected caller %s, got %v", expected, callerID)
	}
}
