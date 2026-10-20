/*
 * NSSF NSSAI Availability
 *
 * NSSF NSSAI Availability Service
 */

package processor

import (
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/free5gc/nssf/internal/logger"
	"github.com/free5gc/nssf/internal/util"
	"github.com/free5gc/nssf/pkg/factory"
	"github.com/free5gc/openapi/models"
	"github.com/free5gc/util/metrics/sbi"
)

// Create a new subscription with a UUID and add it to configuration.

func createSubscription(
	createData models.Nssf_NSSAIAvail_NssfEventSubscriptionCreateData,
	ownerNfID string,
) (factory.Subscription, error) {
	var subscription factory.Subscription

	factory.NssfConfig.Lock()
	defer factory.NssfConfig.Unlock()

	// Generate a unique subscription ID.
	subscription.SubscriptionId = uuid.NewString()
	subscription.OwnerNfId = ownerNfID

	subscription.SubscriptionData = new(models.Nssf_NSSAIAvail_NssfEventSubscriptionCreateData)
	*subscription.SubscriptionData = createData

	factory.NssfConfig.Subscriptions = append(
		factory.NssfConfig.Subscriptions,
		subscription,
	)

	return subscription, nil
}

// NSSAIAvailability subscription POST method
func (p *Processor) NssaiAvailabilitySubscriptionCreate(
	c *gin.Context,
	createData models.Nssf_NSSAIAvail_NssfEventSubscriptionCreateData,
	ownerNfID string,
) {
	var (
		response       = &models.Nssf_NSSAIAvail_NssfEventSubscriptionCreatedData{}
		problemDetails *models.ProblemDetails
	)

	subscription, err := createSubscription(createData, ownerNfID)
	if err != nil {
		logger.NssaiavailLog.Warn(err)

		problemDetails = &models.ProblemDetails{
			Title:  util.UNSUPPORTED_RESOURCE,
			Status: http.StatusNotFound,
			Detail: err.Error(),
		}
		c.Set(sbi.IN_PB_DETAILS_CTX_STR, problemDetails.Title)
		util.GinProblemJson(c, problemDetails)
		return
	}

	response.SubscriptionId = subscription.SubscriptionId
	if subscription.SubscriptionData.Expiry != nil && !subscription.SubscriptionData.Expiry.IsZero() {
		response.Expiry = new(time.Time)
		*response.Expiry = *subscription.SubscriptionData.Expiry
	}
	response.AuthorizedNssaiAvailabilityData = util.AuthorizeOfTaListFromConfig(subscription.SubscriptionData.TaiList)

	c.JSON(http.StatusOK, response)
}

func (p *Processor) NssaiAvailabilitySubscriptionUnsubscribe(
	c *gin.Context,
	subscriptionId string,
	callerNfID string,
) {
	factory.NssfConfig.Lock()
	defer factory.NssfConfig.Unlock()

	for i, subscription := range factory.NssfConfig.Subscriptions {
		if subscription.SubscriptionId == subscriptionId {
			// Only the subscription owner may delete it.
			if subscription.OwnerNfId != callerNfID {
				c.JSON(http.StatusForbidden, gin.H{
					"error": "NF is not the subscription owner",
				})
				return
			}
			factory.NssfConfig.Subscriptions = append(
				factory.NssfConfig.Subscriptions[:i],
				factory.NssfConfig.Subscriptions[i+1:]...,
			)

			c.Status(http.StatusNoContent)
			return
		}
	}

	problemDetails := &models.ProblemDetails{
		Title:  util.UNSUPPORTED_RESOURCE,
		Status: http.StatusNotFound,
		Detail: fmt.Sprintf(
			"Subscription ID '%s' is not available",
			subscriptionId,
		),
	}

	c.Set(sbi.IN_PB_DETAILS_CTX_STR, problemDetails.Title)
	util.GinProblemJson(c, problemDetails)
}
