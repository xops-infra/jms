package api

import (
	"github.com/gin-gonic/gin"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
	"github.com/xops-infra/ginx/middleware"
	hh "github.com/xops-infra/http-headers"

	_ "github.com/xops-infra/jms/docs"
)

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
	api.POST("/login", login)

	api.POST("/broadcast", broadcast)

	u := api.Group("/user")
	u.GET("", listUser)
	u.POST("", addUser)
	u.PATCH("/:id", updateUserGroup)
	u.PUT("/:id", updateUser)

	p := api.Group("/policy")
	p.GET("", listPolicy)
	p.PUT("/:id", updatePolicy)
	p.DELETE("/:id", deletePolicy)

	a := api.Group("/approval")
	a.POST("", createApproval)
	a.PATCH("/:id", updateApproval)

	k := api.Group("/key")
	k.GET("", listKey)
	k.POST("", addKey)
	k.DELETE("/:uuid", deleteKey)

	profile := api.Group("/profile")
	profile.GET("", listProfile)
	profile.POST("", createProfile)
	profile.PUT(":uuid", updateProfile)
	profile.DELETE(":uuid", deleteProfile)

	proxy := api.Group("/proxy")
	proxy.GET("", listProxy)
	proxy.POST("", addProxy)
	proxy.PUT("/:uuid", updateProxy)
	proxy.DELETE("/:uuid", deleteProxy)

	shell := api.Group("/shell/task")
	shell.GET("", listShellTask)
	shell.POST("", addShellTask)
	shell.PUT("/:uuid", updateShellTask)
	shell.DELETE("/:uuid", deleteShellTask)

	shellRecord := api.Group("/shell/record")
	shellRecord.GET("", listShellRecord)

	audits := api.Group("/audit")
	audits.GET("/login", listLoginAudit)
	audits.GET("/scp", listScpAudit)

	return r
}
