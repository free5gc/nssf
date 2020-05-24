package handler

import (
	"github.com/sirupsen/logrus"

	. "free5gc/lib/openapi/models"
	"free5gc/src/nssf/handler/message"
	"free5gc/src/nssf/logger"
	"free5gc/src/nssf/plugin"
	"free5gc/src/nssf/producer"
)

const (
	MaxChannel int = 100
)

var nssfChannel chan message.HandlerMessage
var HandlerLog *logrus.Entry

func init() {
	// init Pool
	HandlerLog = logger.HandlerLog
	nssfChannel = make(chan message.HandlerMessage, MaxChannel)
}

func SendMessage(msg message.HandlerMessage) {
	nssfChannel <- msg
}

func Handle() {
	for {
		msg, ok := <-nssfChannel
		if ok {
			switch msg.Event {
			case message.NSSelectionGet:
				query := msg.HttpRequest.Query
				producer.NSSelectionGet(msg.ResponseChan, query)
			case message.NSSAIAvailabilityPut:
				nfId := msg.HttpRequest.Params["nfId"]
				nssaiAvailabilityInfo := msg.HttpRequest.Body.(NssaiAvailabilityInfo)
				producer.NSSAIAvailabilityPut(msg.ResponseChan, nfId, nssaiAvailabilityInfo)
			case message.NSSAIAvailabilityPatch:
				nfId := msg.HttpRequest.Params["nfId"]
				patchDocument := msg.HttpRequest.Body.(plugin.PatchDocument)
				producer.NSSAIAvailabilityPatch(msg.ResponseChan, nfId, patchDocument)
			case message.NSSAIAvailabilityDelete:
				nfId := msg.HttpRequest.Params["nfId"]
				producer.NSSAIAvailabilityDelete(msg.ResponseChan, nfId)
			case message.NSSAIAvailabilityPost:
				nssfEventSubscriptionCreateData := msg.HttpRequest.Body.(NssfEventSubscriptionCreateData)
				producer.NSSAIAvailabilityPost(msg.ResponseChan, nssfEventSubscriptionCreateData)
			case message.NSSAIAvailabilityUnsubscribe:
				subscriptionId := msg.HttpRequest.Params["subscriptionId"]
				producer.NSSAIAvailabilityUnsubscribe(msg.ResponseChan, subscriptionId)
			default:
				HandlerLog.Warnf("Event[%d] has not implemented", int(msg.Event))
			}
		} else {
			HandlerLog.Errorln("Channel closed!")
			break
		}
	}
}
