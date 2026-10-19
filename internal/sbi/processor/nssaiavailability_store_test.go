package processor_test

import (
	tlsContext "context"
	tlsX509 "crypto/x509"
	"encoding/json"
	tlsErrors "errors"
	"fmt"
	tlsIO "io"
	tlsLog "log"
	"net/http"
	"net/http/httptest"
	tlsAtomic "sync/atomic"
	"testing"

	"github.com/free5gc/nssf/internal/sbi/processor"
	"github.com/free5gc/nssf/internal/util"
	"github.com/free5gc/nssf/pkg/app"
	"github.com/free5gc/nssf/pkg/factory"
	tlsOpenAPI "github.com/free5gc/openapi"
	"github.com/free5gc/openapi/models"
	tlsNFM "github.com/free5gc/openapi/nrf/NFMgmt"
	"github.com/gin-gonic/gin"
	"go.uber.org/mock/gomock"
	tlsYAML "gopkg.in/yaml.v2"
)

func setup() {
	// Set the default values for the factory.NssfConfig
	factory.NssfConfig = &factory.Config{
		Configuration: &factory.Configuration{},
	}
}

func TestMain(m *testing.M) {
	// Run the tests
	setup()
	m.Run()
}

func TestNfInstanceDelete(t *testing.T) {
	mockNssfApp := app.NewMockNssfApp(gomock.NewController(t))
	processor := processor.NewProcessor(mockNssfApp)
	httpRecorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(httpRecorder)

	// Create a sample AMF list
	amfList := []factory.AmfConfig{
		{
			NfId: "nf1",
		},
		{
			NfId: "nf2",
		},
		{
			NfId: "nf3",
		},
	}

	// Set the sample AMF list in the factory.NssfConfig.Configuration
	factory.NssfConfig.Configuration.AmfList = amfList

	// Test case 1: Delete an existing NF instance
	nfIdToDelete := "nf2"
	processor.NssaiAvailabilityNfInstanceDelete(c, nfIdToDelete)
	if c.Writer.Status() != http.StatusNoContent {
		t.Errorf("Expected status code %d, got: %d", http.StatusNoContent, httpRecorder.Code)
	}

	// Verify that the NF instance is deleted from the AMF list
	for _, amfConfig := range factory.NssfConfig.Configuration.AmfList {
		if amfConfig.NfId == nfIdToDelete {
			t.Errorf("Expected NF instance '%s' to be deleted, but it still exists", nfIdToDelete)
		}
	}

	// Test case 2: Delete a non-existing NF instance
	nfIdToDelete = "nf4"
	expectedDetail := fmt.Sprintf("AMF ID '%s' does not exist", nfIdToDelete)
	processor.NssaiAvailabilityNfInstanceDelete(c, nfIdToDelete)
	if httpRecorder.Code != http.StatusNotFound {
		t.Errorf("Expected status code %d, got: %d", http.StatusNotFound, httpRecorder.Code)
	}

	var problemDetails models.ProblemDetails
	if err := json.Unmarshal(httpRecorder.Body.Bytes(), &problemDetails); err != nil {
		t.Errorf("Error unmarshalling response body: %v", err)
	}
	if problemDetails.Title != util.UNSUPPORTED_RESOURCE {
		t.Errorf("Expected problemDetails.Title to be '%s', got: '%s'", util.UNSUPPORTED_RESOURCE, problemDetails.Title)
	}
	if problemDetails.Status != http.StatusNotFound {
		t.Errorf("Expected problemDetails.Status to be %d, got: %d", http.StatusNotFound, problemDetails.Status)
	}
	if problemDetails.Detail != expectedDetail {
		t.Errorf("Expected problemDetails.Detail to be '%s', got: '%s'", expectedDetail, problemDetails.Detail)
	}
}

// Exercise parsed NF config through the same startup configuration API and
// generated NRF client used in production. No deployment files are modified.
func TestSBIConfigTLSVerification(t *testing.T) {
	t.Cleanup(func() {
		_ = tlsOpenAPI.ConfigureHTTP2Clients(tlsOpenAPI.TransportReadIdleTimeout, tlsOpenAPI.TransportPingTimeout, false)
	})
	var requests tlsAtomic.Int32
	server := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests.Add(1)
		w.Header().Set("Content-Type", "application/json")
		_, _ = tlsIO.WriteString(w, `{}`)
	}))
	server.EnableHTTP2 = true
	server.Config.ErrorLog = tlsLog.New(tlsIO.Discard, "", 0)
	server.StartTLS()
	defer server.Close()
	for _, value := range []string{"", "false", "true"} {
		t.Run("verify="+value, func(t *testing.T) {
			data := "configuration:\n  sbi:\n    scheme: https\n"
			if value != "" {
				data += "    tls:\n      verify: " + value + "\n"
			}
			var cfg factory.Config
			if err := tlsYAML.Unmarshal([]byte(data), &cfg); err != nil {
				t.Fatal(err)
			}
			if cfg.GetTLSVerify() != (value == "true") {
				t.Fatal("verify parsing mismatch")
			}
			if err := tlsOpenAPI.ConfigureHTTP2Clients(cfg.GetReadIdleTimeout(), cfg.GetPingTimeout(), cfg.GetTLSVerify()); err != nil {
				t.Fatal(err)
			}
			apiCfg := tlsNFM.NewConfiguration()
			apiCfg.SetBasePath(server.URL)
			client := tlsNFM.NewAPIClient(apiCfg)
			req := &tlsNFM.GetNFInstanceRequest{}
			req.SetNfInstanceID("tls-test")
			before := requests.Load()
			_, err := client.NFInstanceIDDocumentApi.GetNFInstance(tlsContext.Background(), req)
			if value == "true" {
				var unknown tlsX509.UnknownAuthorityError
				if !tlsErrors.As(err, &unknown) {
					t.Fatalf("expected unknown authority, got %v", err)
				}
				if requests.Load() != before {
					t.Fatal("HTTP handler reached despite rejected certificate")
				}
			} else if err != nil {
				t.Fatal(err)
			} else if requests.Load() != before+1 {
				t.Fatal("request not received")
			}
		})
	}
}
