package consumer

import (
	"context"

	nssf_context "github.com/free5gc/nssf/internal/context"
	"github.com/free5gc/nssf/internal/logger"
	"github.com/free5gc/openapi"
	"github.com/free5gc/openapi/models"
	"github.com/free5gc/openapi/oauth"
)

func GetTokenCtx(scope, targetNF string) (context.Context, *models.ProblemDetails, error) {
	if nssf_context.GetSelf().OAuth2Required {
		logger.ConsumerLog.Debugln("GetToekenCtx")
		nssfSelf := nssf_context.GetSelf()
		tok, pd, err := oauth.SendAccTokenReq(nssfSelf.NfId, models.NfType_NSSF, scope, targetNF, nssfSelf.NrfUri)
		if err != nil {
			return nil, pd, err
		}
		return context.WithValue(context.Background(),
			openapi.ContextOAuth2, tok), pd, nil
	}
	return context.TODO(), nil, nil
}
