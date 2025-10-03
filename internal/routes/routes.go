package routes

import (
	"database/sql"
	"salome-be/internal/handlers"
	"salome-be/internal/middleware"

	"github.com/gin-gonic/gin"
)

func SetupRoutes(r *gin.Engine, authHandler *handlers.AuthHandler, groupHandler *handlers.GroupHandler, subscriptionHandler *handlers.SubscriptionHandler, paymentHandler *handlers.PaymentHandler, appHandler *handlers.AppHandler, messageHandler *handlers.MessageHandler, transactionHandler *handlers.TransactionHandler, otpHandler *handlers.OTPHandler, accountCredentialsHandler *handlers.AccountCredentialsHandler, emailSubmissionHandler *handlers.EmailSubmissionHandler, adminHandler *handlers.AdminHandler, midtransHandler *handlers.MidtransHandler, notificationHandler *handlers.NotificationHandler, broadcastHandler *handlers.BroadcastHandler, chatHandler *handlers.ChatHandler, userBroadcastHandler *handlers.UserBroadcastHandler, db *sql.DB) {
	// API v1
	v1 := r.Group("/api/v1")

	// Auth routes
	auth := v1.Group("/auth")
	{
		auth.POST("/register", authHandler.Register)
		auth.POST("/login", authHandler.Login)
		auth.GET("/profile", middleware.OptimizedAuthRequired(), authHandler.GetProfile)
		auth.PUT("/profile", middleware.OptimizedAuthRequired(), authHandler.UpdateProfile)
		auth.PUT("/change-password", middleware.OptimizedAuthRequired(), authHandler.ChangePasswordWithOTP)
		auth.PUT("/reset-password", authHandler.ResetPasswordWithOTP) // No auth required for forgot password
		auth.GET("/stats", middleware.OptimizedAuthRequired(), authHandler.GetUserStats)
		auth.POST("/change-email", middleware.OptimizedAuthRequired(), authHandler.ChangeEmailWithOTP)
		auth.POST("/verify-email-change", middleware.OptimizedAuthRequired(), authHandler.VerifyEmailChange)
		auth.POST("/recalculate-total-spent", middleware.OptimizedAuthRequired(), authHandler.RecalculateTotalSpent)
		auth.POST("/recalculate-user-total-spent", middleware.OptimizedAuthRequired(), authHandler.RecalculateUserTotalSpent)
	}

	// OTP routes (no auth required, but with rate limiting)
	otp := v1.Group("/otp")
	{
		// otp.POST("/generate", middleware.OTPRateLimit(db), otpHandler.GenerateOTP)
		// otp.POST("/verify", middleware.OTPVerifyRateLimit(db), otpHandler.VerifyOTP)
		// otp.POST("/resend", middleware.OTPRateLimit(db), otpHandler.ResendOTP)
		otp.POST("/generate", otpHandler.GenerateOTP)
		otp.POST("/verify", otpHandler.VerifyOTP)
		otp.POST("/resend", otpHandler.ResendOTP)
	}

	// Group routes (require active status)
	groups := v1.Group("/groups")
	groups.Use(middleware.OptimizedAuthRequiredWithStatus(db))
	{
		groups.POST("", groupHandler.CreateGroup)
		groups.POST("/join", groupHandler.JoinGroup)
		groups.DELETE("/:id/leave", groupHandler.LeaveGroup)
		groups.GET("", groupHandler.GetUserGroups)
		groups.GET("/:id", groupHandler.GetGroupDetails)
		groups.GET("/:id/members", groupHandler.GetGroupMembers)
		groups.PUT("/:id", groupHandler.UpdateGroup)
		groups.PUT("/:id/status", groupHandler.UpdateGroupStatus)
		groups.PUT("/:id/transfer-ownership", groupHandler.TransferOwnership)
		groups.DELETE("/:id", groupHandler.DeleteGroup)
		groups.GET("/public", groupHandler.GetPublicGroups)               // Public groups for joining
		groups.GET("/creation-limit", groupHandler.GetGroupCreationLimit) // Get group creation limit info

		// State machine endpoints - using different path structure
		groups.GET("/:id/status", groupHandler.GetGroupStatus)
		groups.GET("/:id/users/:user_id/status", groupHandler.GetUserStatus)
	}

	// Public group routes (no auth required for browsing)
	publicGroups := v1.Group("/public-groups")
	{
		publicGroups.GET("", groupHandler.GetPublicGroups)
		publicGroups.GET("/invite/:code", groupHandler.GetGroupByInviteCode)
	}

	// Subscription routes (require active status) - removed unused routes

	// Payment routes (require active status)
	payments := v1.Group("/payments")
	payments.Use(middleware.OptimizedAuthRequiredWithStatus(db))
	{
		payments.POST("", paymentHandler.CreatePayment)
		payments.POST("/group-payment-link", paymentHandler.CreateGroupPaymentLink)
		payments.GET("", paymentHandler.GetUserPayments)
	}

	// App routes (no auth required for browsing)
	apps := v1.Group("/apps")
	{
		apps.GET("", appHandler.GetApps)
		apps.GET("/:id", appHandler.GetAppByID)
		apps.GET("/categories", appHandler.GetAppCategories)
		apps.GET("/popular", appHandler.GetPopularApps)
		apps.POST("/seed", appHandler.SeedApps) // Development only
	}

	// Message routes (require active status)
	messages := v1.Group("/messages")
	messages.Use(middleware.OptimizedAuthRequiredWithStatus(db))
	{
		messages.GET("/groups/:groupId", messageHandler.GetGroupMessages)
		messages.POST("/groups/:groupId", messageHandler.CreateGroupMessage)
	}

	// Transaction routes (require active status)
	transactions := v1.Group("/transactions")
	transactions.Use(middleware.OptimizedAuthRequiredWithStatus(db))
	{
		transactions.GET("", transactionHandler.GetUserTransactions)
		transactions.POST("", transactionHandler.CreateTransaction)
		transactions.POST("/top-up", transactionHandler.TopUpBalance)
	}

	// Account credentials routes (require active status)
	accountCredentials := v1.Group("/account-credentials")
	accountCredentials.Use(middleware.OptimizedAuthRequiredWithStatus(db))
	{
		accountCredentials.GET("", accountCredentialsHandler.GetUserAccountCredentials)
		accountCredentials.POST("", accountCredentialsHandler.CreateOrUpdateAccountCredentials)
		accountCredentials.GET("/group/:groupId", accountCredentialsHandler.GetAccountCredentialsByGroup)
	}

	// Email submission routes (require active status)
	emailSubmissions := v1.Group("/email-submissions")
	emailSubmissions.Use(middleware.OptimizedAuthRequiredWithStatus(db))
	{
		emailSubmissions.POST("", emailSubmissionHandler.CreateEmailSubmission)
		emailSubmissions.GET("", emailSubmissionHandler.GetUserEmailSubmissions)
		emailSubmissions.GET("/:id", emailSubmissionHandler.GetEmailSubmission)
	}

	// Admin routes (require admin role)
	admin := v1.Group("/admin")
	admin.Use(middleware.AuthRequiredWithStatus(db))
	admin.Use(middleware.AdminRequired(db))
	{
		// Email submissions admin routes
		admin.GET("/email-submissions", emailSubmissionHandler.GetEmailSubmissions)
		admin.PUT("/email-submissions/:id/status", emailSubmissionHandler.UpdateEmailSubmissionStatus)

		// Admin management routes
		admin.GET("/users", adminHandler.GetUsers)
		admin.PUT("/users/status", adminHandler.UpdateUserStatus)
		admin.GET("/groups", adminHandler.GetGroups)
		admin.PUT("/groups/status", adminHandler.UpdateGroupStatus)
		admin.POST("/groups", adminHandler.CreateGroup)
		admin.PUT("/groups", adminHandler.UpdateGroup)
		admin.DELETE("/groups", adminHandler.DeleteGroup)
		admin.GET("/groups/:id/members", adminHandler.GetGroupMembers)
		admin.PUT("/groups/change-owner", adminHandler.ChangeGroupOwner)
		admin.DELETE("/groups/members", adminHandler.RemoveGroupMember)
		admin.POST("/groups/members", adminHandler.AddGroupMember)
		admin.GET("/apps", adminHandler.GetApps)
		admin.PUT("/apps/status", adminHandler.UpdateAppStatus)
		admin.POST("/apps", adminHandler.CreateApp)
		admin.PUT("/apps", adminHandler.UpdateApp)
		admin.DELETE("/apps", adminHandler.DeleteApp)

		// Subscription management routes
		admin.GET("/subscriptions", subscriptionHandler.GetPaidGroupsWithCredentials)
	}

	// Midtrans routes
	midtrans := v1.Group("/midtrans")
	{
		midtrans.POST("/check-status", midtransHandler.CheckTransactionStatus)
		midtrans.GET("/payment-link/:order_id", midtransHandler.GetTransactionPaymentLink)
		midtrans.GET("/update-all-pending", midtransHandler.UpdateAllPendingTransactions)
	}

	// Notification routes
	notifications := v1.Group("/notifications")
	{
		notifications.GET("/:user_id", notificationHandler.GetUserNotifications)
		notifications.PUT("/mark-read", notificationHandler.MarkAsRead)
		notifications.PUT("/:user_id/mark-all-read", notificationHandler.MarkAllAsRead)
		notifications.POST("/create", notificationHandler.CreateNotification) // Admin only
	}

	// Broadcast routes
	broadcasts := v1.Group("/broadcasts")
	{
		// Public route to get broadcast for a group
		broadcasts.GET("/groups/:group_id", broadcastHandler.GetGroupBroadcast)

		// Admin routes (require admin role)
		adminBroadcasts := broadcasts.Group("")
		adminBroadcasts.Use(middleware.AuthRequiredWithStatus(db))
		adminBroadcasts.Use(middleware.AdminRequired(db))
		{
			adminBroadcasts.POST("", broadcastHandler.CreateBroadcast)
			adminBroadcasts.GET("", broadcastHandler.GetBroadcasts)
			adminBroadcasts.PUT("/:id", broadcastHandler.UpdateBroadcast)
			adminBroadcasts.DELETE("/:id", broadcastHandler.DeleteBroadcast)
		}
	}

	// Chat routes
	chats := v1.Group("/chats")
	{
		chats.POST("", chatHandler.CreateChat)                  // Can be anonymous or authenticated
		chats.GET("/:id/messages", chatHandler.GetChatMessages) // Can be anonymous or authenticated
		chats.POST("/:id/messages", chatHandler.SendMessage)    // Can be anonymous or authenticated
	}

	// Authenticated chat routes (for user's own chats)
	userChats := v1.Group("/user-chats")
	userChats.Use(middleware.OptimizedAuthRequired())
	{
		userChats.GET("", chatHandler.GetUserChats)
	}

	// Admin chat routes
	adminChats := v1.Group("/admin/chats")
	adminChats.Use(middleware.OptimizedAuthRequired())
	adminChats.Use(middleware.AdminRequired(db))
	{
		adminChats.GET("", chatHandler.GetAllChats)
		adminChats.GET("/:id/messages", chatHandler.GetChatMessages) // Admin can view any chat
		adminChats.POST("/:id/messages", chatHandler.SendMessageAsAdmin)
		adminChats.PUT("/:id/status", chatHandler.UpdateChatStatus)
		adminChats.PUT("/:id/read", chatHandler.MarkChatAsRead)     // Mark chat as read
		adminChats.PUT("/:id/unread", chatHandler.MarkChatAsUnread) // Mark chat as unread
		adminChats.POST("/broadcast", chatHandler.SendBroadcast)
	}

	// Admin user broadcast routes
	adminUserBroadcasts := v1.Group("/admin/user-broadcasts")
	adminUserBroadcasts.Use(middleware.OptimizedAuthRequired())
	adminUserBroadcasts.Use(middleware.AdminRequired(db))
	{
		adminUserBroadcasts.GET("", userBroadcastHandler.GetUserBroadcasts)
		adminUserBroadcasts.POST("", userBroadcastHandler.CreateUserBroadcast)
		adminUserBroadcasts.GET("/:id", userBroadcastHandler.GetUserBroadcast)
		adminUserBroadcasts.PUT("/:id", userBroadcastHandler.UpdateUserBroadcast)
		adminUserBroadcasts.DELETE("/:id", userBroadcastHandler.DeleteUserBroadcast)
		adminUserBroadcasts.POST("/:id/send", userBroadcastHandler.SendUserBroadcast)
	}

	// User broadcast routes (for dashboard)
	userBroadcasts := v1.Group("/user-broadcasts")
	userBroadcasts.Use(middleware.OptimizedAuthRequired())
	{
		userBroadcasts.GET("", userBroadcastHandler.GetUserBroadcastsForDashboard)
	}

	// Webhook routes (no auth required)
	webhooks := v1.Group("/webhooks")
	{
		webhooks.POST("/midtrans", paymentHandler.HandlePaymentNotification)
	}

	// Health check
	r.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok"})
	})
}
