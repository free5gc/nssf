package util

import (
	"github.com/gin-gonic/gin"
	"github.com/pkg/errors"
	"net/http"
	"net/http/httptest"
	"testing"
)

const (
	Valid   = "valid"
	Invalid = "invalid"
)

type mockNSSFContext struct{}

func newMockNSSFContext() *mockNSSFContext {
	return &mockNSSFContext{}
}

func (m *mockNSSFContext) AuthorizationCheck(token string, serviceName string) error {
	if token == Valid {
		return nil
	}

	return errors.New("invalid token")
}

func TestRouterAuthorizationCheck_Check(t *testing.T) {
	// Mock gin.Context
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request, _ = http.NewRequest("GET", "/", nil)

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
			c.Request, _ = http.NewRequest("GET", "/", nil)
			c.Request.Header.Set("Authorization", tt.args.token)

			rac := NewRouterAuthorizationCheck("testService")
			rac.Check(c, newMockNSSFContext())
			if w.Code != tt.want.statusCode {
				t.Errorf("StatusCode should be %d, but got %d", tt.want.statusCode, w.Code)
			}
		})
	}
}
