/*
 * NSSF NSSAI Availability
 *
 * NSSF NSSAI Availability Service
 */

package producer

import (
	"fmt"
	"math"
	"net/http"
	"strconv"
	"time"

	. "free5gc/lib/openapi/models"
	"free5gc/src/nssf/factory"
	"free5gc/src/nssf/logger"
	"free5gc/src/nssf/util"
)

// Get available subscription ID from configuration
// In this implementation, string converted from 32-bit integer is used as subscription ID
func getUnusedSubscriptionId() (string, error) {
	var idx uint32 = 1
	for _, subscription := range factory.NssfConfig.Subscriptions {
		tempId, _ := strconv.Atoi(subscription.SubscriptionId)
		if uint32(tempId) == idx {
			if idx == math.MaxUint32 {
				return "", fmt.Errorf("No available subscription ID")
			}
			idx = idx + 1
		} else {
			break
		}
	}
	return strconv.Itoa(int(idx)), nil
}

// NSSAIAvailability subscription POST method
func subscriptionPost(createData NssfEventSubscriptionCreateData, createdData *NssfEventSubscriptionCreatedData, problemDetail *ProblemDetails) (status int) {
	var subscription factory.Subscription
	tempId, err := getUnusedSubscriptionId()
	if err != nil {
		logger.Nssaiavailability.Warnf(err.Error())

		*problemDetail = ProblemDetails{
			Title:  util.UNSUPPORTED_RESOURCE,
			Status: http.StatusNotFound,
			Detail: err.Error(),
		}

		status = http.StatusNotFound
		return
	}

	subscription.SubscriptionId = tempId
	subscription.SubscriptionData = new(NssfEventSubscriptionCreateData)
	*subscription.SubscriptionData = createData

	factory.NssfConfig.Subscriptions = append(factory.NssfConfig.Subscriptions, subscription)

	createdData.SubscriptionId = subscription.SubscriptionId
	if !subscription.SubscriptionData.Expiry.IsZero() {
		createdData.Expiry = new(time.Time)
		*createdData.Expiry = *subscription.SubscriptionData.Expiry
	}
	createdData.AuthorizedNssaiAvailabilityData = util.AuthorizeOfTaListFromConfig(subscription.SubscriptionData.TaiList)

	status = http.StatusCreated
	return
}

func subscriptionDelete(subscriptionId string, problemDetail *ProblemDetails) (status int) {
	for i, subscription := range factory.NssfConfig.Subscriptions {
		if subscription.SubscriptionId == subscriptionId {
			factory.NssfConfig.Subscriptions = append(factory.NssfConfig.Subscriptions[:i],
				factory.NssfConfig.Subscriptions[i+1:]...)

			status = http.StatusNoContent
			return
		}
	}

	// No specific subscription ID exists
	*problemDetail = ProblemDetails{
		Title:  util.UNSUPPORTED_RESOURCE,
		Status: http.StatusNotFound,
		Detail: fmt.Sprintf("Subscription ID '%s' is not available", subscriptionId),
	}

	status = http.StatusNotFound
	return
}
