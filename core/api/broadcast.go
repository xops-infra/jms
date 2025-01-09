package api

import (
	"github.com/gin-gonic/gin"
	"github.com/xops-infra/jms/app"
	. "github.com/xops-infra/jms/model"
)

// @Summary Broadcast
// @Description broadcast
// @Accept  json
// @Param   body body CreateBroadcastRequest true "body"
// @Success 200 {string} string "ok"
// @Router /api/v1/broadcast [post]
func broadcast(c *gin.Context) {
	var req CreateBroadcastRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, err)
		return
	}
	err := app.App.DBIo.AddBroadcast(req)
	if err != nil {
		c.JSON(500, err)
		return
	}
	c.String(200, "ok")
}
