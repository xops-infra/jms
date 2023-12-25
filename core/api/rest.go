package api

import (
	"github.com/gin-gonic/gin"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
	"github.com/xops-infra/ginx/middleware"
	hh "github.com/xops-infra/http-headers"

	_ "github.com/xops-infra/jms/docs"
)

type Response struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    any    `json:"data"`
}

func NewSuccessResponse(data any) Response {
	return Response{
		Code:    200,
		Message: "success",
		Data:    data,
	}
}

func NewErrorResponse(code int, message string) Response {
	return Response{
		Code:    code,
		Message: message,
		Data:    nil,
	}
}

func NewGin() *gin.Engine {
	r := gin.Default()
	middleware.AttachTo(r).
		WithCacheDisabled().
		WithCORS().
		WithRecover().
		WithRequestID(hh.XRequestID).
		WithSecurity().
		WithMetrics()
	// add swagger
	r.GET("/swagger/*any", func(c *gin.Context) {
		c.Next()
	}, ginSwagger.WrapHandler(swaggerFiles.Handler))

	r.GET("/ping", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"message": "pong",
			"code":    200,
		})
	})

	api := r.Group("/api/v1")
	p := api.Group("/policy")
	a := api.Group("/approval")
	{
		// group,user
		// 组管理不走 API 操作。直接走 cli操作。
	}
	{
		// policy
		p.GET("", listPolicy)
		// p.POST("", createPolicy)
		p.PUT("/:id", updatePolicy)
		p.DELETE("/:id", deletePolicy)
	}
	{
		a.POST("", createApproval)
	}
	return r
}
