package exchequer

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"

	"github.com/rotationalio/exchequer/pkg/api/v1"
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
