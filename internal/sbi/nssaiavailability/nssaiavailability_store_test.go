package nssaiavailability

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/free5gc/nssf/internal/util"
	"github.com/free5gc/nssf/pkg/factory"
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
	problemDetails := NfInstanceDelete(nfIdToDelete)
	if problemDetails != nil {
		t.Errorf("Expected problemDetails to be nil, got: %v", problemDetails)
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
	problemDetails = NfInstanceDelete(nfIdToDelete)
	if problemDetails == nil {
		t.Errorf("Expected problemDetails to be non-nil")
	} else {
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
}
