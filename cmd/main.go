package main

import (
	"context"
	"crowdfunding-backend/config"
	"crowdfunding-backend/internal/api/admin"
	"crowdfunding-backend/internal/api/community"
	"crowdfunding-backend/internal/api/payment"
	"crowdfunding-backend/internal/api/project"
	"crowdfunding-backend/internal/api/user"
	"crowdfunding-backend/internal/middleware"
	"crowdfunding-backend/internal/repository/mysql"
	"crowdfunding-backend/internal/service"
	"crowdfunding-backend/internal/util"
	"database/sql"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"crowdfunding-backend/internal/storage"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
	"github.com/go-playground/validator/v10"
	_ "github.com/go-sql-driver/mysql"
	"go.uber.org/zap"
)

func main() {
	// 在 main 函数开始处添加
	defer func() {
		if r := recover(); r != nil {
			util.Logger.Error("程序发生严重错误", zap.Any("error", r))
		}
	}()

	// 初始化配置
	config.Init()

	// 初始化日志
	util.InitLogger(config.AppConfig.LogLevel)
	defer util.Logger.Sync()

	util.Logger.Info("应用程序启动")

	// 设置数据库连接字符串
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?charset=utf8mb4&parseTime=True&loc=Local",
		config.AppConfig.DBUser,
		config.AppConfig.DBPassword,
		config.AppConfig.DBHost,
		config.AppConfig.DBPort,
		config.AppConfig.DBName)

	// 连接数据库
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		util.Logger.Fatal("连接数据库失败", zap.Error(err))
	}
	defer db.Close()

	// 测试数据库连接
	err = db.Ping()
	if err != nil {
		util.Logger.Fatal("数据库连接测试失败", zap.Error(err))
	}
	util.Logger.Info("数据库连接成功")

	// 在初始化数据库连接后添加
	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(25)
	db.SetConnMaxLifetime(5 * time.Minute)

	util.Logger.Info("数据库连接池配置完成")

	// 注册自定义验证器
	if v, ok := binding.Validator.Engine().(*validator.Validate); ok {
		v.RegisterValidation("future_date", util.ValidateFutureDate)
	}

	// 确保上传文件夹存在
	ensureUploadsFolder()

	// 替换 GCS 客户端初始化
	localStorage, err := storage.NewLocalStorage(config.AppConfig.LocalStoragePath)
	if err != nil {
		util.Logger.Fatal("初始化本地存储败", zap.Error(err))
	}

	// 初化存储库、服务和处理器
	userRepo := mysql.NewUserRepository(db)
	userService := service.NewUserService(userRepo)
	authHandler := user.NewAuthHandler(userService)
	profileHandler := user.NewProfileHandler(userService, localStorage)
	projectRepo := mysql.NewProjectRepository(db)
	projectService := service.NewProjectService(projectRepo)
	projectHandler := project.NewProjectHandler(projectService, localStorage)

	// 添加 paymentRepo 初始化
	paymentRepo := mysql.NewPaymentRepository(db)

	// 修改 AdminService 初始化
	adminService := service.NewAdminService(
		userRepo,
		projectRepo,
		paymentRepo,
		db,
	)
	adminHandler := admin.NewAdminHandler(adminService)

	paymentService := service.NewPaymentService(
		paymentRepo,
		userRepo,
		projectRepo,
		db,
	)
	paymentHandler := payment.NewPaymentHandler(paymentService, projectService)

	// 初始化 CommunityService 和 CommunityHandler
	communityRepo := mysql.NewCommunityRepository(db)
	communityService := service.NewCommunityService(communityRepo)
	communityHandler := community.NewCommunityHandler(communityService, localStorage)

	// 初始化 EmailService
	emailService := service.NewEmailService(userRepo)

	// 测试发送邮件
	err = emailService.SendVerificationEmail("your-test-email@example.com", "TestUser")
	if err != nil {
		util.Logger.Error("测试发送邮件失败", zap.Error(err))
	} else {
		util.Logger.Info("测试发送邮件成功")
	}

	// 启动定时任务检查过期项目
	go func() {
		ticker := time.NewTicker(1 * time.Minute) // 每分钟检查一次
		for range ticker.C {
			util.Logger.Info("开始检查过期项目")
			if err := adminService.CheckExpiredProjects(); err != nil {
				util.Logger.Error("检查过期项目失败", zap.Error(err))
			}
		}
	}()

	// 初始化 RefundService
	refundService := service.NewRefundService(paymentRepo, db)
	refundHandler := payment.NewRefundHandler(refundService)

	// 初始化错误监控
	errorMonitor := middleware.NewErrorMonitor()

	// 设置 Gin 路由
	r := gin.Default()

	// 添加中间件
	r.Use(middleware.RecoveryMiddleware())
	r.Use(middleware.ErrorMonitorMiddleware(errorMonitor))

	// 配置 CORS
	corsConfig := cors.DefaultConfig()
	corsConfig.AllowOrigins = []string{config.AppConfig.FrontendURL}
	corsConfig.AllowCredentials = true
	corsConfig.AllowMethods = []string{"GET", "POST", "PUT", "PATCH", "DELETE", "HEAD", "OPTIONS"}
	corsConfig.AllowHeaders = []string{
		"Origin",
		"Content-Length",
		"Content-Type",
		"Authorization",
	}
	corsConfig.ExposeHeaders = []string{
		"Content-Length",
		"Content-Type",
		"Access-Control-Allow-Origin",
	}

	// 先应用 CORS 中间件
	r.Use(cors.New(corsConfig))

	// 加一个自定义的中间件来处理静态文件的CORS
	r.Use(func(c *gin.Context) {
		if strings.HasPrefix(c.Request.URL.Path, "/uploads/") {
			c.Header("Access-Control-Allow-Origin", config.AppConfig.FrontendURL)
			c.Header("Access-Control-Allow-Methods", "GET, OPTIONS")
			c.Header("Access-Control-Allow-Headers", "Origin, Content-Type")

			// 如果是 OPTIONS 请求，直接返回200
			if c.Request.Method == "OPTIONS" {
				c.AbortWithStatus(200)
				return
			}
		}
		c.Next()
	})

	// 配置静态文件服务（只配置一次）
	r.Static("/uploads", config.AppConfig.LocalStoragePath)

	// 定义 API 路由
	api := r.Group("/api")
	{
		// 用户相关路由
		api.POST("/register", authHandler.Register)
		api.POST("/login", authHandler.Login)
		api.POST("/request-password-reset", authHandler.RequestPasswordReset)
		api.POST("/reset-password", authHandler.ResetPassword)
		api.GET("/verify-email", authHandler.VerifyEmail)

		// 需要认证的路由
		authorized := api.Group("/")
		authorized.Use(middleware.AuthMiddleware(userService))
		{
			authorized.GET("/profile", profileHandler.GetProfile)
			authorized.PUT("/profile", profileHandler.UpdateProfile)
			authorized.POST("/logout", authHandler.Logout)
			authorized.POST("/refresh-token", authHandler.RefreshToken)
			authorized.POST("/profile/avatar", profileHandler.UploadAvatar)
			authorized.DELETE("/account", profileHandler.DeleteAccount)
		}

		// 项目相关路由
		api.POST("/projects", middleware.AuthMiddleware(userService), projectHandler.CreateProject)
		api.GET("/projects/:id", projectHandler.GetProject)
		api.PUT("/projects/:id", middleware.AuthMiddleware(userService), projectHandler.UpdateProject)
		api.GET("/projects", projectHandler.ListProjects)
		api.POST("/projects/:id/pledge", middleware.AuthMiddleware(userService), projectHandler.PledgeToProject)

		// 新增的项目相关路由
		api.POST("/projects/search", projectHandler.SearchProjects)
		api.GET("/project-categories", projectHandler.GetCategories)
		api.GET("/project-tags", projectHandler.GetTags)
		api.POST("/projects/:id/tags", middleware.AuthMiddleware(userService), projectHandler.AddTagToProject)
		api.POST("/projects/:id/updates", middleware.AuthMiddleware(userService), projectHandler.CreateProjectUpdate)
		api.GET("/projects/:id/updates", projectHandler.GetProjectUpdates)
		api.POST("/projects/:id/comments", middleware.AuthMiddleware(userService), projectHandler.CreateProjectComment)
		api.GET("/projects/:id/comments", projectHandler.GetProjectComments)

		// 支付相关路由
		api.POST("/payments/projects/:project_id", middleware.AuthMiddleware(userService), paymentHandler.CreatePayment)
		api.GET("/orders/:id", middleware.AuthMiddleware(userService), paymentHandler.GetOrder)
		api.GET("/orders", middleware.AuthMiddleware(userService), paymentHandler.ListOrders)
		api.POST("/orders/:id/refund/failed", middleware.AuthMiddleware(userService), paymentHandler.RequestRefundForFailedProject)
		api.GET("/orders/:id/refund", middleware.AuthMiddleware(userService), refundHandler.GetRefundStatus)

		// 管理员路由组
		adminRoutes := api.Group("/admin")
		adminRoutes.Use(middleware.AuthMiddleware(userService), middleware.AdminMiddleware(userService))
		{
			// 项目管理
			projectAdmin := adminRoutes.Group("/projects")
			{
				projectAdmin.GET("", adminHandler.GetProjects)                      // 获取项目列表
				projectAdmin.POST("/:id/review", adminHandler.ReviewProject)        // 审核项目
				projectAdmin.PATCH("/:id/status", adminHandler.UpdateProjectStatus) // 更新项目状态
				projectAdmin.DELETE("/:id", adminHandler.DeleteProject)             // 删除项目
				projectAdmin.GET("/:id/pledgers", adminHandler.GetProjectPledgers)  // 获取支持者
				projectAdmin.POST("/categories", projectHandler.CreateCategory)     // 创建分类
				projectAdmin.POST("/tags", projectHandler.CreateTag)                // 创建标签
			}

			// 用户管理
			userAdmin := adminRoutes.Group("/users")
			{
				userAdmin.GET("", adminHandler.GetUsers)                // 获取用户列表
				userAdmin.PUT("/:id/role", adminHandler.UpdateUserRole) // 更新用户角色
			}

			// 订单和退款管理
			orderAdmin := adminRoutes.Group("/orders")
			{
				orderAdmin.GET("/refunds", adminHandler.GetRefundRequests)         // 获取退款列表
				orderAdmin.GET("/refunds/:id/approve", adminHandler.ProcessRefund) // 处理退款
			}

			// 发货管理
			shipmentAdmin := adminRoutes.Group("/shipments")
			{
				shipmentAdmin.POST("", adminHandler.CreateShipment)          // 创建发货记录
				shipmentAdmin.PUT("/:id", adminHandler.UpdateShipmentStatus) // 更新发货状态
			}

			// 系统管理
			adminRoutes.GET("/stats", adminHandler.GetSystemStats) // 系统统计
		}

		// 社区相关路由
		api.POST("/posts", middleware.AuthMiddleware(userService), communityHandler.CreatePost)
		api.GET("/posts/:id", communityHandler.GetPost)
		api.PUT("/posts/:id", middleware.AuthMiddleware(userService), communityHandler.UpdatePost)
		api.DELETE("/posts/:id", middleware.AuthMiddleware(userService), communityHandler.DeletePost)
		api.GET("/posts", communityHandler.ListAllPosts)

		api.POST("/posts/:id/comments", middleware.AuthMiddleware(userService), communityHandler.CreateComment)
		api.GET("/posts/:id/comments", communityHandler.ListComments)
		api.DELETE("/comments/:id", middleware.AuthMiddleware(userService), communityHandler.DeleteComment)

		api.POST("/posts/:id/likes", middleware.AuthMiddleware(userService), communityHandler.LikePost)
		api.DELETE("/posts/:id/likes", middleware.AuthMiddleware(userService), communityHandler.UnlikePost)

		api.POST("/users/:id/follow", middleware.AuthMiddleware(userService), communityHandler.FollowUser)
		api.DELETE("/users/:id/follow", middleware.AuthMiddleware(userService), communityHandler.UnfollowUser)
		api.GET("/users/:id/followers", communityHandler.GetFollowers)
		api.GET("/users/:id/following", communityHandler.GetFollowing)

		// 添加这个新路由
		api.GET("/users/following", middleware.AuthMiddleware(userService), communityHandler.GetCurrentUserFollowing)

		// 在社区相关路由部分添加
		api.GET("/users/:id/posts", communityHandler.GetUserPosts)

		// 在用户路由组中添加
		userHandler := user.NewUserHandler(userService)
		api.POST("/addresses", middleware.AuthMiddleware(userService), userHandler.CreateAddress)
		api.PUT("/addresses/:id", middleware.AuthMiddleware(userService), userHandler.UpdateAddress)
		api.DELETE("/addresses/:id", middleware.AuthMiddleware(userService), userHandler.DeleteAddress)
		api.GET("/addresses", middleware.AuthMiddleware(userService), userHandler.ListAddresses)
		api.PUT("/addresses/:id/default", middleware.AuthMiddleware(userService), userHandler.SetDefaultAddress)

		// 在 API 路由组中添加
		api.POST("/admin/login", authHandler.AdminLogin)

		// 在社区相关路由部分添加
		api.POST("/comments/:id/reply", middleware.AuthMiddleware(userService), communityHandler.CreateCommentReply)
		api.GET("/comments/:id/replies", communityHandler.GetCommentReplies)
		api.GET("/users/following/posts", middleware.AuthMiddleware(userService), communityHandler.GetFollowingPosts)
		api.GET("/users/followers/posts", middleware.AuthMiddleware(userService), communityHandler.GetFollowersPosts)

		// 在社区相关路由部分添加
		api.GET("/users/:id/follow/status", middleware.AuthMiddleware(userService), communityHandler.GetFollowStatus)
	}

	// 创建一个带有超时的 http.Server
	srv := &http.Server{
		Addr:    ":8080",
		Handler: r,
	}

	// 在一个新的 goroutine 中启动服务器
	go func() {
		util.Logger.Info("服务器正在启动，监听端口 :8080")
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			util.Logger.Fatal("启动服务器失败", zap.Error(err))
		}
	}()

	// 等待中断信号以优雅地关闭服务器（设置 5 秒的超时时间）
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	util.Logger.Info("正在关闭服务器...")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		util.Logger.Fatal("服务器强制关闭", zap.Error(err))
	}

	util.Logger.Info("服务器已优雅关闭")

	// 在 main 函数末尾添加路由打印
	if config.AppConfig.Debug {
		routes := r.Routes()
		util.Logger.Info("已注册的路由列表：")
		for _, route := range routes {
			util.Logger.Info("路由",
				zap.String("method", route.Method),
				zap.String("path", route.Path),
				zap.String("handler", route.Handler))
		}
	}
}

// 确保上传文件夹存在
func ensureUploadsFolder() {
	uploadsPath := config.AppConfig.LocalStoragePath
	if err := os.MkdirAll(uploadsPath, 0755); err != nil {
		util.Logger.Fatal("创建上传文件夹失败", zap.Error(err), zap.String("path", uploadsPath))
	}
	util.Logger.Info("上传文件夹已创建或已存在", zap.String("path", uploadsPath))
}
