package exchequer

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

func (s *Server) Index(c *gin.Context) {
	c.HTML(http.StatusOK, "index.html", gin.H{})
}
