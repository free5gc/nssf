package processor

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"

	"github.com/free5gc/nssf/pkg/factory"
	"github.com/free5gc/openapi/models"
)

// Test that AMF-A cannot delete AMF-B's subscription,
// while AMF-B can delete its own subscription.
func TestSubscriptionOwnerAuthorization(t *testing.T) {
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

	p := NewProcessor(nil)

	// Case 1: AMF-A attempts to delete AMF-B's subscription.
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	p.NssaiAvailabilitySubscriptionUnsubscribe(
		c, "subscription-b", "amf-a",
	)

	if w.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d", w.Code)
	}

	var problemDetails models.ProblemDetails
	if err := json.Unmarshal(
		w.Body.Bytes(),
		&problemDetails,
	); err != nil {
		t.Fatalf("invalid ProblemDetails JSON: %v", err)
	}

	if problemDetails.Status != http.StatusForbidden {
		t.Errorf(
			"expected ProblemDetails status 403, got %d",
			problemDetails.Status,
		)
	}

	if problemDetails.Cause != "NOT_AUTHORIZED" {
		t.Errorf(
			"expected cause NOT_AUTHORIZED, got %q",
			problemDetails.Cause,
		)
	}

	if problemDetails.Detail == "" {
		t.Error("expected non-empty ProblemDetails detail")
	}

	if len(factory.NssfConfig.Subscriptions) != 1 {
		t.Fatal("AMF-B subscription was incorrectly deleted")
	}

	// Case 2: AMF-B deletes its own subscription.
	w2 := httptest.NewRecorder()
	c2, _ := gin.CreateTestContext(w2)

	p.NssaiAvailabilitySubscriptionUnsubscribe(
		c2, "subscription-b", "amf-b",
	)

	// Flush the HTTP status to the test recorder.
	c2.Writer.WriteHeaderNow()

	if w2.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d", w2.Code)
	}

	if len(factory.NssfConfig.Subscriptions) != 0 {
		t.Fatal("owner failed to delete its subscription")
	}
}

// Test that subscription IDs stay unique even after deletion.
func TestSubscriptionIDsRemainUnique(t *testing.T) {
	oldConfig := factory.NssfConfig

	factory.NssfConfig = &factory.Config{
		Configuration: &factory.Configuration{},
	}

	t.Cleanup(func() {
		factory.NssfConfig = oldConfig
	})

	seen := make(map[string]bool)

	for i := 0; i < 20; i++ {
		sub, err := createSubscription(
			models.Nssf_NSSAIAvail_NssfEventSubscriptionCreateData{},
			"amf-a",
		)
		if err != nil {
			t.Fatal(err)
		}

		if sub.SubscriptionId == "" {
			t.Fatal("empty subscription ID")
		}

		if seen[sub.SubscriptionId] {
			t.Fatalf(
				"duplicate subscription ID: %s",
				sub.SubscriptionId,
			)
		}

		seen[sub.SubscriptionId] = true

		if sub.OwnerNfId != "amf-a" {
			t.Fatal("subscription owner was not saved")
		}

		// Delete an existing entry to simulate ID gaps.
		if i == 3 {
			factory.NssfConfig.Subscriptions = append(
				factory.NssfConfig.Subscriptions[:1],
				factory.NssfConfig.Subscriptions[2:]...,
			)
		}
	}
}
