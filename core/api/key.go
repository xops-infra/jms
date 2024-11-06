package api

import (
	"fmt"
	"net/http"

	"github.com/alibabacloud-go/tea/tea"
	"github.com/gin-gonic/gin"
	"github.com/xops-infra/jms/app"
	. "github.com/xops-infra/jms/model"
	"github.com/xops-infra/noop/log"
)

// @Summary 列出密钥
// @Description 列出密钥，数据隐藏
// @Tags Key
// @Accept  json
// @Produce  json
// @Success 200 {object} []Key
// @Router /api/v1/key [get]
func listKey(c *gin.Context) {
	keys, err := app.App.JmsDBService.ListKey()
	if err != nil {
		c.String(http.StatusInternalServerError, err.Error())
		return
	}
	c.JSON(http.StatusOK, keys)
}

// @Summary 添加密钥
// @Description 添加密钥
// @Tags Key
// @Accept  json
// @Produce  json
// @Param Authorization header string false "token"
// @Param key body AddKeyRequest true "key"
// @Success 200 {string} id
// @Router /api/v1/key [post]
func addKey(c *gin.Context) {
	var req AddKeyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, err.Error())
		return
	}
	log.Debugf("add key: %s", tea.Prettify(req))
	id, err := app.App.JmsDBService.AddKey(req)
	if err != nil {
		c.JSON(500, err.Error())
		return
	}
	c.String(200, id)
}

// @Summary 删除密钥
// @Tags Key
// @Accept  json
// @Produce  json
// @Param Authorization header string false "token"
// @Param uuid path string true "key uuid"
// @Success 200 {string} success
// @Router /api/v1/key/:uuid [delete]
func deleteKey(c *gin.Context) {
	id := c.Param("uuid")
	if id == "" {
		c.JSON(400, fmt.Errorf("uuid is empty"))
		return
	}
	if err := app.App.JmsDBService.DeleteKey(id); err != nil {
		c.JSON(500, err.Error())
		return
	}
	c.String(200, "success")
}
