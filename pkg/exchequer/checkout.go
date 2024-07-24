package exchequer

import (
	"net/http"
	"net/url"

	"github.com/adyen/adyen-go-api-library/v11/src/checkout"
	"github.com/adyen/adyen-go-api-library/v11/src/common"
	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
	"github.com/rotationalio/exchequer/pkg/api/v1"
	"github.com/rotationalio/exchequer/pkg/ulids"
)

func (s *Server) Checkout(c *gin.Context) {
	origin, _ := url.Parse(s.conf.Origin)

	// Create the checkout request object
	key := ulids.New()
	returnURL := origin.JoinPath("/checkout/complete")
	returnURL.RawQuery = url.Values{"session": []string{key.String()}}.Encode()

	sessionRequest := checkout.CreateCheckoutSessionRequest{
		Reference: "INV-0001",
		Amount: checkout.Amount{
			Currency: "USD",
			Value:    1000,
		},
		MerchantAccount: s.conf.Adyen.MerchantAccount,
		CountryCode:     common.PtrString("US"),
		ReturnUrl:       returnURL.String(),
	}

	// Send the request to Adyen
	service := s.adyen.Checkout()
	req := service.PaymentsApi.SessionsInput().IdempotencyKey(key.String()).CreateCheckoutSessionRequest(sessionRequest)
	rep, _, err := service.PaymentsApi.Sessions(c.Request.Context(), req)

	if err != nil {
		c.Error(err)
		c.Negotiate(http.StatusOK, gin.Negotiate{
			Offered:  []string{binding.MIMEJSON, binding.MIMEHTML},
			Data:     api.Error("could not create checkout session with adyen"),
			HTMLName: "500.html",
		})
		return
	}

	out := gin.H{
		"ClientKey":   s.conf.Adyen.ClientKey,
		"SessionID":   rep.Id,
		"SessionData": *rep.SessionData,
	}

	c.Negotiate(http.StatusOK, gin.Negotiate{
		Offered:  []string{binding.MIMEJSON, binding.MIMEHTML},
		Data:     out,
		HTMLName: "checkout.html",
	})
}
