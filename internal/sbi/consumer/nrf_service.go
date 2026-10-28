/*
 * NSSF Consumer
 *
 * Network Function Management
 */

package consumer

import (
	"context"
	"errors"
	"fmt"
	"time"

	nssf_context "github.com/free5gc/nssf/internal/context"
	"github.com/free5gc/nssf/internal/logger"
	"github.com/free5gc/openapi"
	"github.com/free5gc/openapi/models"
	"github.com/free5gc/openapi/nrf/NFMgmt"
	"github.com/free5gc/util/nfheartbeat"
)

const registerRetryInterval = 2 * time.Second

type NrfService struct {
	nrfNfMgmtClient *NFMgmt.APIClient
	// NOTE: No mutex needed. One connection at a time.

	heartbeat *nfheartbeat.Runner

	// Interval in seconds from the last registration response. Written before the heartbeat starts, then only on
	// the heartbeat goroutine.
	heartbeatTimer int32
}

func (ns *NrfService) buildNFProfile(context *nssf_context.NSSFContext) (
	profile models.Nrf_NFMgmt_NFProfile, err error,
) {
	profile.NfInstanceId = context.NfId
	profile.NfType = models.Nrf_NFMgmt_NFType_NSSF
	profile.NfStatus = models.Nrf_NFMgmt_NFStatus_REGISTERED
	profile.PlmnList = context.SupportedPlmnList
	profile.Ipv4Addresses = []string{context.RegisterIPv4}
	var services []models.Nrf_NFMgmt_NFService
	for _, nfService := range context.NfService {
		services = append(services, nfService)
	}
	if len(services) > 0 {
		profile.NfServices = services
	}
	return
}

// SendRegisterNFInstance PUTs the profile on the NF's own instance ID (3GPP TS 29.510 clause 5.2.2.2.2) until it
// succeeds or ctx is done. Only the startup call may pass applyOAuth2: it writes OAuth2Required unsynchronized.
func (ns *NrfService) SendRegisterNFInstance(ctx context.Context, applyOAuth2 bool) error {
	nssfCtx := nssf_context.GetSelf()
	nfInstanceId := nssfCtx.NfId

	profile, err := ns.buildNFProfile(nssfCtx)
	if err != nil {
		return fmt.Errorf("failed to build nrf profile: %s", err.Error())
	}

	var res *NFMgmt.RegisterNFInstanceResponse
	req := &NFMgmt.RegisterNFInstanceRequest{
		NfInstanceID: &nfInstanceId,
		RequestBody:  &profile,
	}
	for ctx.Err() == nil {
		res, err = ns.nrfNfMgmtClient.NFInstanceIDDocumentApi.RegisterNFInstance(ctx, req)
		if err == nil && res != nil {
			var nf models.Nrf_NFMgmt_NFProfile
			if res.Nrf_NFMgmt_NFProfile != nil {
				nf = *res.Nrf_NFMgmt_NFProfile
			}
			ns.processRegisterResponse(nssfCtx, nf, applyOAuth2)
			return nil
		}
		logger.ConsumerLog.Errorf("NSSF register to NRF Error[%v]", err)
		select {
		case <-ctx.Done():
		case <-time.After(registerRetryInterval):
		}
	}
	return fmt.Errorf("NFRegister aborted: %w (last error: %v)", ctx.Err(), err)
}

// processRegisterResponse adopts what the NRF answered to the NFRegister PUT: the
// heartbeat interval and the oauth2 custom info.
func (ns *NrfService) processRegisterResponse(
	nssfCtx *nssf_context.NSSFContext,
	nf models.Nrf_NFMgmt_NFProfile,
	applyOAuth2 bool,
) {
	ns.heartbeatTimer = nf.HeartBeatTimer

	oauth2 := false
	if customInfo, isMap := nf.CustomInfo.(map[string]interface{}); isMap {
		if v, ok := customInfo["oauth2"].(bool); ok {
			oauth2 = v
			logger.MainLog.Infoln("OAuth2 setting receive from NRF:", oauth2)
		}
	}
	if applyOAuth2 {
		nssfCtx.OAuth2Required = oauth2
		if oauth2 && nssfCtx.NrfCertPem == "" {
			logger.CfgLog.Error("OAuth2 enable but no nrfCertPem provided in config.")
		}
	} else if oauth2 != nssfCtx.OAuth2Required {
		logger.ConsumerLog.Warnf("NRF OAuth2 setting changed to %v, restart NSSF to apply it", oauth2)
	}
}

// SendUpdateNFInstance sends an NFUpdate PATCH to the NRF, honoring ctx. The
// raw err comes back alongside any ProblemDetails so callers can read its
// GenericOpenAPIError status.
func (ns *NrfService) SendUpdateNFInstance(ctx context.Context, patchItems []models.PatchItem) (
	nf models.Nrf_NFMgmt_NFProfile, problemDetails *models.ProblemDetails, err error,
) {
	nssfCtx := nssf_context.GetSelf()
	tokCtx, pd, err := nssfCtx.GetTokenCtx(
		models.Nrf_NFMgmt_ServiceName_NNRF_NFM, models.Nrf_NFMgmt_NFType_NRF)
	if err != nil {
		return nf, pd, err
	}
	// GetTokenCtx takes no parent, so the token request stays uncancelable;
	// transplanting the token lets at least the PATCH honor ctx.
	if tok := tokCtx.Value(openapi.ContextOAuth2); tok != nil {
		ctx = context.WithValue(ctx, openapi.ContextOAuth2, tok)
	}

	req := &NFMgmt.UpdateNFInstanceRequest{
		NfInstanceID: &nssfCtx.NfId,
		RequestBody:  patchItems,
	}

	res, err := ns.nrfNfMgmtClient.NFInstanceIDDocumentApi.UpdateNFInstance(ctx, req)
	if err != nil {
		var apiErr openapi.GenericOpenAPIError
		if errors.As(err, &apiErr) {
			if updateErr, okModel := apiErr.Model().(NFMgmt.UpdateNFInstanceError); okModel {
				return nf, updateErr.ProblemDetails, err
			}
		}
		return nf, nil, err
	}
	if res == nil {
		return nf, nil, errors.New("empty NFUpdate response")
	}
	if res.Nrf_NFMgmt_NFProfile != nil {
		nf = *res.Nrf_NFMgmt_NFProfile
	}
	return nf, nil, nil
}

func (ns *NrfService) SendDeregisterNFInstance(nfInstanceId string) (*models.ProblemDetails, error) {
	logger.ConsumerLog.Infof("Send Deregister NFInstance [%s]", nfInstanceId)

	var err error

	ctx, pd, err := nssf_context.GetSelf().GetTokenCtx(
		models.Nrf_NFMgmt_ServiceName_NNRF_NFM, models.Nrf_NFMgmt_NFType_NRF)
	if err != nil {
		return pd, err
	}

	client := ns.nrfNfMgmtClient

	req := &NFMgmt.DeregisterNFInstanceRequest{
		NfInstanceID: &nfInstanceId,
	}

	_, err = client.NFInstanceIDDocumentApi.DeregisterNFInstance(ctx, req)
	if err != nil {
		if apiErr, ok := err.(openapi.GenericOpenAPIError); ok {
			// API error
			if deregError, ok2 := apiErr.Model().(NFMgmt.DeregisterNFInstanceError); ok2 {
				return deregError.ProblemDetails, err
			}
			return nil, err
		}

		// Golang error
		return nil, err
	}

	return nil, nil
}
