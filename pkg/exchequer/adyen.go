package exchequer

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"net/http"
	"strings"

	"github.com/adyen/adyen-go-api-library/v11/src/webhook"
	"github.com/gin-gonic/gin"
	"github.com/rotationalio/exchequer/pkg/api/v1"
	"github.com/rs/zerolog/log"
)

func (s *Server) AdyenWebhookAuth() gin.HandlerFunc {
	if s.conf.Adyen.Webhook.UseBasicAuth {
		return gin.BasicAuth(gin.Accounts{
			s.conf.Adyen.Webhook.Username: s.conf.Adyen.Webhook.Password,
		})
	}

	// If no authorization is required return no-op for auth
	return func(c *gin.Context) {
		c.Next()
	}
}

func (s *Server) AdyenPaymentsWebhook(c *gin.Context) {
	var (
		err   error
		event *webhook.Webhook
	)

	// Bind JSON data from webhook
	event = &webhook.Webhook{}
	if err = c.BindJSON(event); err != nil {
		c.Error(err)
		c.JSON(http.StatusBadRequest, api.Error("could not parse payments webhook request"))
	}

	notifications := event.GetNotificationItems()
	for i, notification := range notifications {
		// Verify HMAC Signature if required
		if s.conf.Adyen.Webhook.VerifyHMAC {
			if err = VerifyAdyenHMAC(notification, s.conf.Adyen.Webhook.HMACSecret); err != nil {
				c.Error(err)
				c.JSON(http.StatusUnauthorized, api.Error("HMAC signature cannot be verified"))
				return
			}
		}

		log.Info().
			Str("live", event.Live).
			Int("num_notification_items", len(notifications)).
			Int("notification_index", i).
			Str("event_code", notification.EventCode).
			Time("event_date", *notification.EventDate).
			Int64("amount", notification.Amount.Value).
			Str("curency", notification.Amount.Currency).
			Str("merchant_account_code", notification.MerchantAccountCode).
			Str("merchant_reference", notification.MerchantReference).
			Strs("operations", notification.Operations).
			Str("original_reference", notification.OriginalReference).
			Str("payment_method", notification.PaymentMethod).
			Str("psp_reference", notification.PspReference).
			Str("reason", notification.Reason).
			Str("success", notification.Success).
			Msg("adyen payment webhook received")
	}

	c.Status(http.StatusAccepted)
}

func VerifyAdyenHMAC(payload *webhook.NotificationRequestItem, secret string) (err error) {
	// Step 1: Extract the HMAC signature to verify from the additonal data.
	if payload.AdditionalData == nil {
		return ErrMissingHMACSignature
	}
	additionalData := *payload.AdditionalData

	var checkSignature string
	if item, ok := additionalData["hmacSignature"]; !ok {
		return ErrMissingHMACSignature
	} else {
		if checkSignature, ok = item.(string); !ok {
			return ErrInvalidHMACSignature
		}
	}

	if checkSignature == "" {
		return ErrMissingHMACSignature
	}

	// Step 2: Construct the payload for verification
	reference := strings.Join([]string{
		payload.PspReference,
		payload.OriginalReference,
		payload.MerchantAccountCode,
		payload.MerchantReference,
		fmt.Sprintf("%d", payload.Amount.Value),
		payload.Amount.Currency,
		payload.EventCode,
		payload.Success,
	}, ":")

	var secretBytes []byte
	if secretBytes, err = hex.DecodeString(secret); err != nil {
		return ErrInvalidHMACSecret
	}

	// Step 3: Compute the HMAC from the signature and the secret
	hmac := hmac.New(sha256.New, secretBytes)
	hmac.Write([]byte(reference))
	signature := base64.StdEncoding.EncodeToString(hmac.Sum(nil))

	// Step 4: Verify the signature with the check signature
	if signature != checkSignature {
		return ErrInvalidHMACSignature
	}
	return nil
}
