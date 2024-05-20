/*
 * NSSF Consumer
 *
 * Network Function Management
 */

package consumer

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	nssf_context "github.com/free5gc/nssf/internal/context"
	"github.com/free5gc/nssf/internal/logger"
	"github.com/free5gc/openapi"
	"github.com/free5gc/openapi/Nnrf_NFManagement"
	"github.com/free5gc/openapi/models"
)

type NrfService struct {
	nrfNfMgmtClient *Nnrf_NFManagement.APIClient
}

func (ns *NrfService) buildNFProfile(context *nssf_context.NSSFContext) (profile models.NfProfile, err error) {
	profile.NfInstanceId = context.NfId
	profile.NfType = models.NfType_NSSF
	profile.NfStatus = models.NfStatus_REGISTERED
	profile.PlmnList = &context.SupportedPlmnList
	profile.Ipv4Addresses = []string{context.RegisterIPv4}
	var services []models.NfService
	for _, nfService := range context.NfService {
		services = append(services, nfService)
	}
	if len(services) > 0 {
		profile.NfServices = &services
	}
	return
}

func (ns *NrfService) SendRegisterNFInstance(ctx context.Context, nssfCtx *nssf_context.NSSFContext) (
	resourceNrfUri string, retrieveNfInstanceId string, err error,
) {
	nfInstanceId := nssfCtx.NfId
	profile, err := ns.buildNFProfile(nssfCtx)
	if err != nil {
		return "", "", fmt.Errorf("failed to build nrf profile: %s", err.Error())
	}
	apiClient := ns.nrfNfMgmtClient

	var res *http.Response
	var nf models.NfProfile
	finish := false
	for !finish {
		select {
		case <-ctx.Done():
			return "", "", fmt.Errorf("context done")

		default:
			nf, res, err = apiClient.NFInstanceIDDocumentApi.RegisterNFInstance(ctx, nfInstanceId, profile)
			if err != nil || res == nil {
				// TODO : add log
				logger.ConsumerLog.Errorf("NSSF register to NRF Error[%s]", err.Error())
				const retryInterval = 2 * time.Second
				time.Sleep(retryInterval)
				continue
			}
			defer func() {
				if resCloseErr := res.Body.Close(); resCloseErr != nil {
					logger.ConsumerLog.Errorf("NFInstanceIDDocumentApi response body cannot close: %+v", resCloseErr)
				}
			}()
			status := res.StatusCode
			if status == http.StatusOK {
				// NFUpdate
				finish = true
			} else if status == http.StatusCreated {
				// NFRegister
				resourceUri := res.Header.Get("Location")
				resourceNrfUri, _, _ = strings.Cut(resourceUri, "/nnrf-nfm/")
				retrieveNfInstanceId = resourceUri[strings.LastIndex(resourceUri, "/")+1:]

				oauth2 := false
				if nf.CustomInfo != nil {
					v, ok := nf.CustomInfo["oauth2"].(bool)
					if ok {
						oauth2 = v
						logger.MainLog.Infoln("OAuth2 setting receive from NRF:", oauth2)
					}
				}
				nssf_context.GetSelf().OAuth2Required = oauth2
				if oauth2 && nssf_context.GetSelf().NrfCertPem == "" {
					logger.CfgLog.Error("OAuth2 enable but no nrfCertPem provided in config.")
				}
				finish = true
			} else {
				fmt.Println("NRF return wrong status code", status)
			}
		}
	}
	return resourceNrfUri, retrieveNfInstanceId, err
}

func (ns *NrfService) SendDeregisterNFInstance(nfInstanceId string) (*models.ProblemDetails, error) {
	logger.ConsumerLog.Infof("Send Deregister NFInstance")

	var err error

	ctx, pd, err := nssf_context.GetSelf().GetTokenCtx(models.ServiceName_NNRF_NFM, models.NfType_NRF)
	if err != nil {
		return pd, err
	}

	client := ns.nrfNfMgmtClient

	var res *http.Response

	res, err = client.NFInstanceIDDocumentApi.DeregisterNFInstance(ctx, nfInstanceId)
	if err == nil {
		return nil, err
	} else if res != nil {
		defer func() {
			if resCloseErr := res.Body.Close(); resCloseErr != nil {
				logger.ConsumerLog.Errorf("NFInstanceIDDocumentApi response body cannot close: %+v", resCloseErr)
			}
		}()
		if res.Status != err.Error() {
			return nil, err
		}
		problem := err.(openapi.GenericOpenAPIError).Model().(models.ProblemDetails)
		return &problem, err
	} else {
		return nil, openapi.ReportError("server no response")
	}
}
