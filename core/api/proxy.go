package api

import (
	"github.com/gin-gonic/gin"
	"github.com/xops-infra/jms/app"
	"github.com/xops-infra/jms/core/db"
)

// @Summary ListProxy
// @Description list proxy servers
// @Tags proxy
// @Accept json
// @Produce json
// @Success 200 {object} []db.Proxy
// @Router /api/v1/proxy [get]
func listProxy(c *gin.Context) {
	proxies, err := app.App.DBService.ListProxy()
	if err != nil {
		c.JSON(500, err.Error())
		return
	}
	c.JSON(200, proxies)
}

// @Summary AddProxy
// @Description add proxy server
// @Tags proxy
// @Param body body db.CreateProxyRequest true "proxy server info"
// @Success 200 {object} db.Proxy
// @Router /api/v1/proxy [post]
func addProxy(c *gin.Context) {
	var req db.CreateProxyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, err.Error())
		return
	}
	id, err := app.App.DBService.CreateProxy(req)
	if err != nil {
		c.JSON(500, err.Error())
		return
	}
	c.JSON(200, id)
}

// @Summary UpdateProxy
// @Summary UpdateProxy
// @Tags proxy
// @Param body body db.CreateProxyRequest true "proxy server info"
// @Param uuid path string true "proxy server uuid"
// @Success 200 {object} db.Proxy
// @Router /api/v1/proxy/:uuid [put]
func updateProxy(c *gin.Context) {

	var req db.CreateProxyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, err.Error())
		return
	}
	id, err := app.App.DBService.UpdateProxy(c.Param("uuid"), req)
	if err != nil {
		c.JSON(500, err.Error())
		return
	}
	c.JSON(200, id)
}

// @Summary DeleteProxy
// @Tags proxy
// @Accept json
// @Produce json
// @Param uuid path string true "proxy server uuid"
// @Success 200 {string} success
// @Router /api/v1/proxy/:uuid [delete]
func deleteProxy(c *gin.Context) {

	err := app.App.DBService.DeleteProxy(c.Param("uuid"))
	if err != nil {
		c.JSON(500, err.Error())
		return
	}
	c.String(200, "success")
}
