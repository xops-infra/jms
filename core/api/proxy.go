package api

import (
	"github.com/gin-gonic/gin"
	"github.com/xops-infra/jms/app"
	"github.com/xops-infra/jms/model"
)

type proxyListResponse struct {
	Name         *string `json:"name"`
	Host         *string `json:"host"`
	Port         *int    `json:"port"`
	IPPrefix     *string `json:"ip_prefix"`
	LoginUser    *string `json:"login_user"`
	KeyID        *string `json:"key_id"`
	IdentityFile *string `json:"identity_file"`
}

// @Summary ListProxy
// @Description list proxy servers
// @Tags proxy
// @Accept json
// @Produce json
// @Success 200 {object} []Proxy
// @Router /api/v1/proxy [get]
func listProxy(c *gin.Context) {
	proxies, err := app.App.DBIo.ListProxy()
	if err != nil {
		c.JSON(500, err.Error())
		return
	}
	var items []proxyListResponse
	for _, proxy := range proxies {
		items = append(items, proxyListResponse{
			Name:         proxy.Name,
			Host:         proxy.Host,
			Port:         proxy.Port,
			IPPrefix:     proxy.IPPrefix,
			LoginUser:    proxy.LoginUser,
			KeyID:        proxy.KeyID,
			IdentityFile: proxy.IdentityFile,
		})
	}
	c.JSON(200, items)
}

// @Summary AddProxy
// @Description add proxy server
// @Tags proxy
// @Param body body CreateProxyRequest true "proxy server info"
// @Success 200 {object} Proxy
// @Router /api/v1/proxy [post]
func addProxy(c *gin.Context) {
	var req model.CreateProxyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, err.Error())
		return
	}
	id, err := app.App.DBIo.CreateProxy(req)
	if err != nil {
		c.JSON(500, err.Error())
		return
	}
	c.JSON(200, id)
}

// @Summary UpdateProxy
// @Summary UpdateProxy
// @Tags proxy
// @Param body body CreateProxyRequest true "proxy server info"
// @Param uuid path string true "proxy server uuid"
// @Success 200 {object} Proxy
// @Router /api/v1/proxy/:uuid [put]
func updateProxy(c *gin.Context) {

	var req model.CreateProxyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, err.Error())
		return
	}
	id, err := app.App.DBIo.UpdateProxy(c.Param("uuid"), req)
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

	err := app.App.DBIo.DeleteProxy(c.Param("uuid"))
	if err != nil {
		c.JSON(500, err.Error())
		return
	}
	c.String(200, "success")
}
