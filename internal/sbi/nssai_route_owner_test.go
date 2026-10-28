package sbi

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"

	"github.com/free5gc/nssf/internal/sbi/processor"
	"github.com/free5gc/nssf/internal/util"
	"github.com/free5gc/nssf/pkg/factory"
	"github.com/free5gc/openapi/models"
)

type testCallerContext struct{}

func (testCallerContext) IsOAuth2Required() bool {
	return true
}

func (testCallerContext) AuthorizationCheck(
	token string,
	service models.Nrf_NFMgmt_ServiceName,
) error {
	if token != "Bearer test-amf-a" {
		return errors.New("invalid token")
	}
	return nil
}

func (testCallerContext) VerifiedCallerNFID(
	token string,
	service models.Nrf_NFMgmt_ServiceName,
) (string, error) {
	if token != "Bearer test-amf-a" {
		return "", errors.New("invalid token")
	}
	return "11111111-1111-4111-8111-111111111111", nil
}

func TestNSSAIRouteOwnerAuthorization(t *testing.T) {
	gin.SetMode(gin.TestMode)

	const (
		amfA = "11111111-1111-4111-8111-111111111111"
		amfB = "22222222-2222-4222-8222-222222222222"
	)

	oldConfig := factory.NssfConfig
	factory.NssfConfig = &factory.Config{
		Configuration: &factory.Configuration{
			AmfList: []factory.AmfConfig{
				{NfId: amfA},
				{NfId: amfB},
			},
		},
	}
	t.Cleanup(func() {
		factory.NssfConfig = oldConfig
	})

	server := &Server{
		processor: processor.NewProcessor(nil),
	}

	router := gin.New()

	group := router.Group(factory.NssfNssaiavailResUriPrefix)
	group.Use(func(c *gin.Context) {
		util.NewRouterAuthorizationCheck(
			models.Nrf_NFMgmt_ServiceName_NNSSF_NSSAIAVAILABILITY,
		).Check(c, testCallerContext{})
	})

	group.DELETE("/nssai-availability/:nfId",
		server.NSSAIAvailabilityDelete)

	request := func(target string) int {
		w := httptest.NewRecorder()
		req := httptest.NewRequestWithContext(
			context.Background(),
			http.MethodDelete,
			factory.NssfNssaiavailResUriPrefix+
				"/nssai-availability/"+target,
			nil,
		)
		req.Header.Set("Authorization", "Bearer test-amf-a")
		router.ServeHTTP(w, req)
		return w.Code
	}

	// AMF-A cannot delete AMF-B's record.
	if status := request(amfB); status != http.StatusForbidden {
		t.Fatalf("cross-NF DELETE: expected 403, got %d", status)
	}

	if len(factory.NssfConfig.Configuration.AmfList) != 2 {
		t.Fatal("AMF-B data was modified")
	}

	// AMF-A can delete its own record.
	if status := request(amfA); status != http.StatusNoContent {
		t.Fatalf("own DELETE: expected 204, got %d", status)
	}

	list := factory.NssfConfig.Configuration.AmfList
	if len(list) != 1 || list[0].NfId != amfB {
		t.Fatal("unexpected AMF data after authorized DELETE")
	}
}

type oauthDisabledTestCallerContext struct{}

func (oauthDisabledTestCallerContext) IsOAuth2Required() bool {
	return false
}

func (oauthDisabledTestCallerContext) AuthorizationCheck(
	_ string,
	_ models.Nrf_NFMgmt_ServiceName,
) error {
	return nil
}

func (oauthDisabledTestCallerContext) VerifiedCallerNFID(
	_ string,
	_ models.Nrf_NFMgmt_ServiceName,
) (string, error) {
	return "", errors.New("NF identity verification should be skipped")
}

func TestNSSAIRouteOAuth2DisabledDelete(t *testing.T) {
	gin.SetMode(gin.TestMode)

	const amfB = "22222222-2222-4222-8222-222222222222"

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

	server := &Server{
		processor: processor.NewProcessor(nil),
	}

	router := gin.New()
	group := router.Group(factory.NssfNssaiavailResUriPrefix)

	group.Use(func(c *gin.Context) {
		util.NewRouterAuthorizationCheck(
			models.Nrf_NFMgmt_ServiceName_NNSSF_NSSAIAVAILABILITY,
		).Check(c, oauthDisabledTestCallerContext{})
	})

	group.DELETE(
		"/nssai-availability/:nfId",
		server.NSSAIAvailabilityDelete,
	)

	w := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(
		context.Background(),
		http.MethodDelete,
		factory.NssfNssaiavailResUriPrefix+
			"/nssai-availability/"+amfB,
		nil,
	)

	// No Authorization header: OAuth2 is disabled.
	router.ServeHTTP(w, req)

	if w.Code != http.StatusNoContent {
		t.Fatalf("expected HTTP 204, got %d: %s",
			w.Code, w.Body.String())
	}

	if len(factory.NssfConfig.Configuration.AmfList) != 0 {
		t.Fatal("NSSAI availability resource was not deleted")
	}
}

func TestNSSAIRouteOAuth2DisabledUnsubscribe(t *testing.T) {
	gin.SetMode(gin.TestMode)

	oldConfig := factory.NssfConfig
	factory.NssfConfig = &factory.Config{
		Configuration: &factory.Configuration{},
		Subscriptions: []factory.Subscription{
			{
				SubscriptionId: "subscription-b",
				OwnerNfId:      "amf-b",
			},
		},
	}
	t.Cleanup(func() {
		factory.NssfConfig = oldConfig
	})

	server := &Server{
		processor: processor.NewProcessor(nil),
	}

	router := gin.New()
	group := router.Group(factory.NssfNssaiavailResUriPrefix)

	group.Use(func(c *gin.Context) {
		util.NewRouterAuthorizationCheck(
			models.Nrf_NFMgmt_ServiceName_NNSSF_NSSAIAVAILABILITY,
		).Check(c, oauthDisabledTestCallerContext{})
	})

	group.DELETE(
		"/nssai-availability/subscriptions/:subscriptionId",
		server.NSSAIAvailabilityUnsubscribeDelete,
	)

	w := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(
		context.Background(),
		http.MethodDelete,
		factory.NssfNssaiavailResUriPrefix+
			"/nssai-availability/subscriptions/subscription-b",
		nil,
	)

	// No Authorization header because OAuth2 is disabled.
	router.ServeHTTP(w, req)

	if w.Code != http.StatusNoContent {
		t.Fatalf("expected HTTP 204, got %d: %s",
			w.Code, w.Body.String())
	}

	if len(factory.NssfConfig.Subscriptions) != 0 {
		t.Fatal("subscription was not deleted")
	}
}
