package processor_test

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"

	"github.com/gin-gonic/gin"
	"go.uber.org/mock/gomock"

	"github.com/free5gc/nssf/internal/plugin"
	"github.com/free5gc/nssf/internal/sbi/processor"
	"github.com/free5gc/nssf/internal/util"
	"github.com/free5gc/nssf/pkg/app"
	"github.com/free5gc/nssf/pkg/factory"
	"github.com/free5gc/openapi/models"
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

func TestNfInstancePatchUpdatesCorrectAmf(t *testing.T) {
	oldConfig := factory.NssfConfig
	t.Cleanup(func() {
		factory.NssfConfig = oldConfig
	})

	plmn := &models.PlmnId{Mcc: "208", Mnc: "93"}
	tai := &models.Tai{
		PlmnId: plmn,
		Tac:    "000001",
	}

	factory.NssfConfig = &factory.Config{
		Configuration: &factory.Configuration{
			SupportedNssaiInPlmnList: []factory.SupportedNssaiInPlmn{
				{PlmnId: plmn},
			},
			AmfList: []factory.AmfConfig{
				{
					NfId: "amf-other",
					SupportedNssaiAvailabilityData: []models.Nssf_NSSAIAvail_SupportedNssaiAvailabilityData{
						{
							Tai: tai,
							SupportedSnssaiList: []models.ExtSnssai{
								{Sst: 3},
							},
						},
					},
				},
				{
					NfId: "amf-target",
					SupportedNssaiAvailabilityData: []models.Nssf_NSSAIAvail_SupportedNssaiAvailabilityData{
						{
							Tai: tai,
							SupportedSnssaiList: []models.ExtSnssai{
								{Sst: 1},
							},
						},
					},
				},
			},
		},
	}

	p := processor.NewProcessor(
		app.NewMockNssfApp(gomock.NewController(t)),
	)

	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)

	patch := plugin.PatchDocument{
		{
			Op:    models.PatchOperation_REPLACE,
			Path:  "/0/supportedSnssaiList/0/sst",
			Value: 2,
		},
	}

	p.NssaiAvailabilityNfInstancePatch(c, patch, "amf-target")

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected HTTP 200, got %d: %s",
			recorder.Code, recorder.Body.String())
	}

	amfList := factory.NssfConfig.Configuration.AmfList

	if amfList[0].SupportedNssaiAvailabilityData[0].
		SupportedSnssaiList[0].Sst != 3 {
		t.Fatal("PATCH modified the wrong AMF")
	}

	if amfList[1].SupportedNssaiAvailabilityData[0].
		SupportedSnssaiList[0].Sst != 2 {
		t.Fatal("PATCH did not update the target AMF")
	}

	var response models.Nssf_NSSAIAvail_AuthorizedNssaiAvailabilityInfo
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatalf("invalid response JSON: %v", err)
	}

	if len(response.AuthorizedNssaiAvailabilityData) != 1 {
		t.Fatalf("expected one authorized NSSAI entry")
	}

	if response.AuthorizedNssaiAvailabilityData[0].
		SupportedSnssaiList[0].Sst != 2 {
		t.Fatal("response does not match updated NSSAI data")
	}
}

func TestNfInstancePutPatchDeleteConcurrent(t *testing.T) {
	oldConfig := factory.NssfConfig
	t.Cleanup(func() {
		factory.NssfConfig = oldConfig
	})

	plmn := &models.PlmnId{Mcc: "208", Mnc: "93"}
	tai := &models.Tai{
		PlmnId: plmn,
		Tac:    "000001",
	}

	data := []models.Nssf_NSSAIAvail_SupportedNssaiAvailabilityData{
		{
			Tai: tai,
			SupportedSnssaiList: []models.ExtSnssai{
				{Sst: 1},
			},
		},
	}

	amfList := make([]factory.AmfConfig, 4)
	for i := range amfList {
		amfList[i] = factory.AmfConfig{
			NfId:                           fmt.Sprintf("amf-%d", i),
			SupportedNssaiAvailabilityData: data,
		}
	}

	factory.NssfConfig = &factory.Config{
		Configuration: &factory.Configuration{
			SupportedNssaiInPlmnList: []factory.SupportedNssaiInPlmn{
				{PlmnId: plmn},
			},
			TaList: []factory.TaConfig{
				{Tai: tai},
			},
			AmfList: amfList,
		},
	}

	p := processor.NewProcessor(
		app.NewMockNssfApp(gomock.NewController(t)),
	)

	info := models.Nssf_NSSAIAvail_NssaiAvailabilityInfo{
		SupportedNssaiAvailabilityData: data,
	}

	router := gin.New()
	router.Use(gin.Recovery())

	router.PUT("/test/:nfId", func(c *gin.Context) {
		p.NssaiAvailabilityNfInstanceUpdate(c, info, c.Param("nfId"))
	})

	patch := plugin.PatchDocument{
		{
			Op:    models.PatchOperation_REPLACE,
			Path:  "/0/supportedSnssaiList/0/sst",
			Value: 2,
		},
	}

	router.PATCH("/test/:nfId", func(c *gin.Context) {
		p.NssaiAvailabilityNfInstancePatch(c, patch, c.Param("nfId"))
	})

	router.DELETE("/test/:nfId", func(c *gin.Context) {
		p.NssaiAvailabilityNfInstanceDelete(c, c.Param("nfId"))
	})

	const workers = 12
	const iterations = 100

	start := make(chan struct{})
	var wg sync.WaitGroup

	for worker := 0; worker < workers; worker++ {
		wg.Add(1)

		go func(worker int) {
			defer wg.Done()
			<-start

			for i := 0; i < iterations; i++ {
				nfId := fmt.Sprintf("amf-%d", (i+worker)%4)

				method := http.MethodPut

				switch worker % 3 {
				case 1:
					method = http.MethodPatch
				case 2:
					method = http.MethodDelete
				}

				recorder := httptest.NewRecorder()
				req := httptest.NewRequestWithContext(
					context.Background(), method, "/test/"+nfId, nil,
				)

				router.ServeHTTP(recorder, req)

				status := recorder.Code
				switch method {
				case http.MethodPut:
					if status != http.StatusOK {
						t.Errorf("PUT %s: unexpected status %d", nfId, status)
						continue
					}

					var response models.Nssf_NSSAIAvail_AuthorizedNssaiAvailabilityInfo
					if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
						t.Errorf("PUT %s: invalid response: %v", nfId, err)
						continue
					}

					if len(response.AuthorizedNssaiAvailabilityData) != 1 {
						t.Errorf("PUT %s: expected one authorized NSSAI entry, got %d",
							nfId, len(response.AuthorizedNssaiAvailabilityData))
						continue
					}

					snssaiList := response.AuthorizedNssaiAvailabilityData[0].SupportedSnssaiList
					if len(snssaiList) != 1 || snssaiList[0].Sst != 1 {
						t.Errorf("PUT %s: response does not match submitted NSSAI", nfId)
					}

				case http.MethodPatch:
					if status != http.StatusOK &&
						status != http.StatusNotFound {
						t.Errorf("PATCH %s: unexpected status %d", nfId, status)
					}

				case http.MethodDelete:
					if status != http.StatusNoContent &&
						status != http.StatusNotFound {
						t.Errorf("DELETE %s: unexpected status %d", nfId, status)
					}
				}
			}
		}(worker)
	}

	close(start)
	wg.Wait()

	// Check final state after all workers have finished.
	factory.NssfConfig.RLock()
	defer factory.NssfConfig.RUnlock()

	seen := make(map[string]bool)
	for _, amf := range factory.NssfConfig.Configuration.AmfList {
		if seen[amf.NfId] {
			t.Errorf("duplicate AMF ID: %s", amf.NfId)
		}
		seen[amf.NfId] = true
	}
}

func TestNfInstancePatchNotFoundProblemDetails(t *testing.T) {
	oldConfig := factory.NssfConfig
	t.Cleanup(func() {
		factory.NssfConfig = oldConfig
	})

	factory.NssfConfig = &factory.Config{
		Configuration: &factory.Configuration{},
	}

	p := processor.NewProcessor(
		app.NewMockNssfApp(gomock.NewController(t)),
	)

	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)

	p.NssaiAvailabilityNfInstancePatch(c, nil, "missing-amf")

	result := recorder.Result()
	defer func() {
		if err := result.Body.Close(); err != nil {
			t.Errorf("failed to close response body: %v", err)
		}
	}()

	if result.StatusCode != http.StatusNotFound {
		t.Errorf("expected HTTP 404, got %d", result.StatusCode)
	}

	var details models.ProblemDetails
	if err := json.NewDecoder(result.Body).Decode(&details); err != nil {
		t.Fatalf("invalid ProblemDetails response: %v", err)
	}

	if details.Status != http.StatusNotFound {
		t.Errorf("expected ProblemDetails status 404, got %d", details.Status)
	}

	if details.Title != util.UNSUPPORTED_RESOURCE {
		t.Errorf("unexpected ProblemDetails title: %s", details.Title)
	}
}
