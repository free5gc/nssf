package context

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"

	"github.com/free5gc/openapi/models"
	"github.com/free5gc/openapi/oauth"
)

// VerifiedCallerNFID validates an access token and returns the authenticated NF Instance ID (sub).
func (c *NSSFContext) VerifiedCallerNFID(
	authorization string,
	serviceName models.Nrf_NFMgmt_ServiceName,
) (string, error) {
	if !c.OAuth2Required {
		return "", errors.New(
			"cannot verify NF identity when OAuth2 is disabled",
		)
	}

	fields := strings.Fields(authorization)
	if len(fields) != 2 ||
		!strings.EqualFold(fields[0], "Bearer") {
		return "", errors.New("invalid Bearer authorization header")
	}

	// Preserve free5GC's existing OAuth2 validation,
	// including its service scope check.
	if err := c.AuthorizationCheck(
		authorization, serviceName,
	); err != nil {
		return "", err
	}

	publicKey, err := oauth.ParsePublicKeyFromPEM(
		c.NrfCertPem,
	)
	if err != nil {
		return "", fmt.Errorf(
			"load NRF public key: %w", err,
		)
	}

	claims := &models.Nrf_AccTok_AccessTokenClaims{}

	token, err := jwt.ParseWithClaims(
		fields[1],
		claims,
		func(token *jwt.Token) (interface{}, error) {
			return publicKey, nil
		},
		jwt.WithValidMethods([]string{"RS512"}),
	)
	if err != nil {
		return "", fmt.Errorf(
			"verify access token: %w", err,
		)
	}

	if !token.Valid {
		return "", errors.New("invalid access token")
	}

	// Reject tokens without a valid expiration timestamp.
	if claims.Exp <= 0 {
		return "", errors.New("missing or invalid access token expiration")
	}

	// Exp is a Unix timestamp in seconds.
	// The token must not be accepted at or after expiration.
	if time.Now().Unix() >= int64(claims.Exp) {
		return "", errors.New("access token has expired")
	}

	if strings.TrimSpace(claims.Sub) == "" {
		return "", errors.New(
			"missing NF Instance ID in token subject",
		)
	}

	return claims.Sub, nil
}
