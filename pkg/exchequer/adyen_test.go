package exchequer_test

import (
	"testing"

	"github.com/adyen/adyen-go-api-library/v11/src/webhook"
	"github.com/rotationalio/exchequer/pkg/exchequer"
	"github.com/stretchr/testify/require"
)

var (
	exampleHMACSecret   = "44782DEF547AAA06C910C43932B1EB0C71FC68D9D0C057550C48EC2ACF6BA056"
	exampleWebhookEvent = `{
   "live":"false",
   "notificationItems":[
      {
         "NotificationRequestItem":{
            "additionalData":{
               "hmacSignature":"coqCmt/IZ4E3CzPvMY8zTjQVL5hYJUiBRg8UU+iCWo0="
            },
            "amount":{
               "value":1130,
               "currency":"EUR"
            },
            "pspReference":"7914073381342284",
            "eventCode":"AUTHORISATION",
            "eventDate":"2019-05-06T17:15:34.121+02:00",
            "merchantAccountCode":"TestMerchant",
            "operations":[
               "CANCEL",
               "CAPTURE",
               "REFUND"
            ],
            "merchantReference":"TestPayment-1407325143704",
            "paymentMethod":"visa",
            "success":"true"
         }
      }
   ]
}`
)

func TestVerifyAdyenHMAC(t *testing.T) {
	event, err := webhook.HandleRequest(exampleWebhookEvent)
	require.NoError(t, err, "could not process example webhook event")

	t.Run("Valid", func(t *testing.T) {
		notification := event.GetNotificationItems()[0]
		err := exchequer.VerifyAdyenHMAC(notification, exampleHMACSecret)
		require.NoError(t, err, "could not verify valid hmac with secret")
	})

	t.Run("InvalidSecret", func(t *testing.T) {
		notification := event.GetNotificationItems()[0]
		err := exchequer.VerifyAdyenHMAC(notification, "229382c61727723d66ef1a819b2a136f")
		require.ErrorIs(t, err, exchequer.ErrInvalidHMACSignature, "expected error with invalid hmac secret")
	})

}
