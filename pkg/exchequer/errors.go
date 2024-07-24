package exchequer

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"

	"github.com/rotationalio/exchequer/pkg/api/v1"
)

var (
	ErrMissingHMACSignature = errors.New("HMAC id or signature is missing")
	ErrInvalidHMACSignature = errors.New("invalid HMAC signature")
	ErrInvalidHMACSecret    = errors.New("HMAC secret must be a hex encoded string")
)

func (s *Server) NotFound(c *gin.Context) {
	c.Negotiate(http.StatusServiceUnavailable, gin.Negotiate{
		Offered:  []string{binding.MIMEJSON, binding.MIMEHTML},
		Data:     api.NotFound,
		HTMLName: "404.html",
	})
}

func (s *Server) NotAllowed(c *gin.Context) {
	c.Negotiate(http.StatusServiceUnavailable, gin.Negotiate{
		Offered:  []string{binding.MIMEJSON, binding.MIMEHTML},
		Data:     api.NotAllowed,
		HTMLName: "405.html",
	})
}
