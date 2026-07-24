package route

import (
	"yonghemolimis/src/apps/api/chatws"
	miniapi "yonghemolimis/src/apps/api/mini"
	misapi "yonghemolimis/src/apps/api/mis"
	"yonghemolimis/src/apps/api/user"
	"yonghemolimis/src/middlewares"

	"github.com/gin-gonic/gin"
)

// SetupRoutes 注册所有 API 路由
func SetupRoutes(r *gin.Engine) {
	mini := r.Group("/api/mini")
	miniapi.RegisterRoutes(mini)
	chatws.RegisterRoutes(mini, chatws.RoleMini)

	api := r.Group("/api/v1")
	chatws.RegisterRoutes(api, chatws.RoleAdmin)

	// 公开接口
	api.POST("/oidc/authorize", user.OIDCAuthorize)
	api.POST("/oidc/callback", user.OIDCCallback)
	api.GET("/oidc/session", user.OIDCSession)
	api.POST("/logout", user.Logout)

	// 需要认证的接口
	auth := api.Group("", middlewares.AuthRequired())
	{
		auth.GET("/me", user.Me)
		auth.POST("/uploads/image", misapi.UploadImage)
		auth.POST("/uploads/file", misapi.UploadFile)

		auth.GET("/users", misapi.ListUsers)
		auth.GET("/users/export", misapi.ExportUsers)
		auth.POST("/users/:id/ban", misapi.BanUser)
		auth.POST("/users/:id/unban", misapi.UnbanUser)

		// 装修
		auth.GET("/decoration", misapi.GetDecoration)
		auth.GET("/decoration/banners", misapi.GetDecorationBanners)
		auth.PUT("/decoration/banners", misapi.SaveDecorationBanners)
		auth.GET("/decoration/customer-service", misapi.GetDecorationCustomerService)
		auth.PUT("/decoration/customer-service", misapi.SaveDecorationCustomerService)
		auth.GET("/decoration/company", misapi.GetDecorationCompany)
		auth.PUT("/decoration/company", misapi.SaveDecorationCompany)
		auth.GET("/decoration/services", misapi.ListCareServiceCategories)
		auth.POST("/decoration/services", misapi.SaveCareServiceCategory)
		auth.PUT("/decoration/services/:id", misapi.SaveCareServiceCategory)
		auth.DELETE("/decoration/services/:id", misapi.DeleteCareServiceCategory)

		// 阿姨列表和阿姨申请
		auth.GET("/caregivers", misapi.ListCaregivers)
		auth.GET("/caregivers/:id", misapi.GetCaregiver)
		auth.POST("/caregivers", misapi.SaveCaregiver)
		auth.PUT("/caregivers/:id", misapi.SaveCaregiver)
		auth.PUT("/caregivers/:id/status", misapi.UpdateCaregiverStatus)
		auth.DELETE("/caregivers/:id", misapi.DeleteCaregiver)
		auth.GET("/caregiver-applications", misapi.ListResumes)
		auth.PUT("/caregiver-applications/:id/status", misapi.UpdateResumeStatus)
		auth.PUT("/caregiver-applications/:id/assign", misapi.AssignResume)

		// 预约管理
		auth.GET("/appointments", misapi.ListDemands)
		auth.PUT("/appointments/:id/status", misapi.UpdateDemandStatus)
		auth.PUT("/appointments/:id/assign", misapi.AssignDemand)
		auth.GET("/status-history/:entityType/:id", misapi.BusinessStatusHistory)

		auth.GET("/faqs", misapi.ListFAQs)
		auth.POST("/faqs", misapi.CreateFAQ)
		auth.PUT("/faqs/:id", misapi.UpdateFAQ)
		auth.DELETE("/faqs/:id", misapi.DeleteFAQ)
		auth.POST("/faqs/:id/publish", misapi.PublishFAQ)
		auth.POST("/faqs/:id/unpublish", misapi.UnpublishFAQ)

		auth.GET("/chat/sessions", misapi.ListChatSessions)
		auth.GET("/chat/sessions/:id/messages", misapi.ListChatMessages)
		auth.POST("/chat/sessions/:id/messages", misapi.CreateChatMessage)
		auth.POST("/chat/sessions/:id/close", misapi.CloseChatSession)
		auth.POST("/chat/sessions/:id/read", misapi.ReadChatSession)
	}
}
