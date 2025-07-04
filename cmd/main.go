// @title           Go Email Service API
// @version         1.0
// @description     This is a powerful email service API for batch sending and management.
// @termsOfService  http://swagger.io/terms/

// @contact.name   email-service/internal/handler
// @contact.url    http://www.swagger.io/support
// @contact.email  support@swagger.io

// @license.name  Apache 2.0
// @license.url   http://www.apache.org/licenses/LICENSE-2.0.html

// @host      localhost:8080
// @BasePath  /

// @securityDefinitions.apikey ApiKeyAuth
// @in header
// @name Authorization
// @description Type "Bearer" followed by a space and a JWT token.
package main

import (
	"context"
	"email-service/internal/config"
	"email-service/internal/handler"
	"email-service/internal/middleware"
	"email-service/internal/queue"
	"email-service/internal/repository"
	"email-service/internal/service"
	"email-service/pkg/database"
	"email-service/pkg/elasticsearch"
	"email-service/pkg/redis"
	"log"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
)

const (
	// elasticsearchRecipientsIndex is the name of the Elasticsearch index for recipients.
	elasticsearchRecipientsIndex = "recipients"
)

// elasticsearchRecipientsMapping defines the mapping for the 'recipients' index.
// Using 'keyword' for fields that are used for filtering, sorting, or aggregations.
// Using 'text' for fields that are meant for full-text search.
// 'dynamic: true' for metadata allows for flexible, unstructured data.
var elasticsearchRecipientsMapping = `
{
  "mappings": {
    "properties": {
      "id":         { "type": "integer" },
      "account_id": { "type": "integer" },
      "email":      { "type": "keyword" },
      "first_name": { "type": "text" },
      "last_name":  { "type": "text" },
      "status":     { "type": "keyword" },
      "created_at": { "type": "date" },
      "updated_at": { "type": "date" },
      "metadata":   { "type": "object", "dynamic": true }
    }
  }
}`

func main() {
	// Load configuration
	cfg, err := config.LoadConfig(".")
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	// Initialize database
	db, err := database.InitDB(&cfg.Database)
	if err != nil {
		log.Fatalf("could not connect to the database: %v", err)
	}

	// Initialize Redis client
	redisClient, err := redis.NewClient(&cfg.Redis)
	if err != nil {
		log.Fatalf("could not connect to redis: %v", err)
	}

	// Initialize Elasticsearch client
	esClient, err := elasticsearch.NewClient(&cfg.Elasticsearch)
	if err != nil {
		log.Fatalf("could not connect to elasticsearch: %v", err)
	}

	// Ensure the Elasticsearch index for recipients exists with the correct mapping.
	// This is a crucial step for production readiness.
	if err := elasticsearch.EnsureIndexExists(esClient, elasticsearchRecipientsIndex, elasticsearchRecipientsMapping); err != nil {
		log.Fatalf("Failed to ensure Elasticsearch index exists: %v", err)
	}

	// Initialize queue service
	queueService := queue.NewRedisQueueService(redisClient)

	// Initialize repositories
	accountRepo := repository.NewAccountRepository(db)
	senderRepo := repository.NewSenderRepository(db)
	templateRepo := repository.NewTemplateRepository(db)
	emailRecordRepo := repository.NewEmailSendRecordRepository(db)
	recipientRepo := repository.NewRecipientRepository(db)
	taskRepo := repository.NewEmailTaskRepository(db)
	userRepo := repository.NewGORMUserRepository(db)
	userPermissionRepo := repository.NewGORMUserPermissionRepository(db)
	sendStatisticsRepo := repository.NewSendStatisticsRepository(db)
	groupRepo := repository.NewRecipientGroupRepository(db, esClient)

	// Initialize services
	accountService := service.NewAccountService(accountRepo)
	senderService := service.NewSenderService(senderRepo, accountRepo)
	templateService := service.NewTemplateService(templateRepo)
	recipientService := service.NewRecipientService(recipientRepo, queueService)
	userService := service.NewUserService(userRepo)
	authService := service.NewAuthService(userService, cfg.JWT)
	groupService := service.NewRecipientGroupService(groupRepo, recipientRepo)
	syncService := service.NewSyncService(queueService, recipientRepo, esClient)

	// New: Initialize the LoadBalancerService
	loadBalancerService := service.NewLoadBalancerService(senderRepo, sendStatisticsRepo, userPermissionRepo)

	emailService := service.NewEmailService(senderRepo, emailRecordRepo, taskRepo, queueService, cfg.Aliyun.Endpoint)

	// Updated: EmailTaskService now has fewer dependencies
	taskService := service.NewEmailTaskService(taskRepo, groupRepo, templateRepo, queueService)

	pollingInterval, err := time.ParseDuration(cfg.Scheduler.PollingInterval)
	if err != nil {
		log.Fatalf("Invalid scheduler polling interval: %v", err)
	}
	// Updated: TaskDispatcherService now includes the LoadBalancer
	taskDispatcherService := service.NewTaskDispatcherService(taskRepo, emailRecordRepo, senderRepo, templateRepo, groupService, loadBalancerService, queueService, pollingInterval)
	taskDispatcherService.Start(context.Background())

	trackingService := service.NewTrackingService(taskRepo, emailRecordRepo, senderRepo, sendStatisticsRepo, cfg.Aliyun.Endpoint)
	go trackingService.Start(context.Background(), time.Minute*1)

	// Start the new ES Sync Service
	syncService.Start(context.Background())

	emailWorker := service.NewEmailWorkerService(queueService, emailService)
	emailWorker.Start(context.Background(), cfg.Worker.EmailSenderCount)

	statisticsService := service.NewStatisticsService(emailRecordRepo, taskRepo, sendStatisticsRepo)

	// Initialize handlers
	accountHandler := handler.NewAccountHandler(accountService)
	senderHandler := handler.NewSenderHandler(senderService)
	templateHandler := handler.NewTemplateHandler(templateService)
	emailHandler := handler.NewEmailHandler(senderRepo, emailRecordRepo, queueService)
	recipientHandler := handler.NewRecipientHandler(recipientService)
	taskHandler := handler.NewEmailTaskHandler(taskService)
	authHandler := handler.NewAuthHandler(authService)
	userHandler := handler.NewUserHandler(userService)
	statisticsHandler := handler.NewStatisticsHandler(statisticsService)
	groupHandler := handler.NewRecipientGroupHandler(groupService)

	// Setup router
	r := gin.Default()

	// Use a more permissive CORS middleware
	corsConfig := cors.DefaultConfig()
	corsConfig.AllowAllOrigins = true
	corsConfig.AllowHeaders = []string{"Origin", "Content-Length", "Content-Type", "Authorization"}
	r.Use(cors.New(corsConfig))

	// Swagger setup to avoid routing conflicts.
	// 1. Serve the Swagger UI assets from the /swagger/ path.
	// 2. Point the UI to /swagger.json to fetch the API specification. This path is outside the /swagger/*any group.
	r.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler, ginSwagger.URL("/swagger.json")))

	// 3. Serve the static swagger.json file from the /swagger.json path.
	// This avoids the runtime rendering of the spec which can be problematic.
	r.GET("/swagger.json", func(c *gin.Context) {
		c.File("./docs/swagger.json")
	})

	// API v1 routes
	apiV1 := r.Group("/api/v1")
	{
		// --- Public Routes ---
		apiV1.POST("/login", authHandler.Login)
		apiV1.GET("/ping", func(c *gin.Context) {
			c.JSON(200, gin.H{"message": "pong"})
		})

		// --- Protected Routes ---
		authRequired := apiV1.Group("/")
		authRequired.Use(middleware.AuthMiddleware(cfg.JWT))
		{
			// User management
			authRequired.POST("/users", userHandler.CreateUser)

			// Account management
			accounts := authRequired.Group("/accounts")
			{
				accounts.POST("", accountHandler.CreateAccount)
				accounts.GET("", accountHandler.GetAccounts)
				accounts.GET("/:id", accountHandler.GetAccount)
				accounts.PUT("/:id", accountHandler.UpdateAccount)
				accounts.DELETE("/:id", accountHandler.DeleteAccount)
			}

			// Sender management
			senders := authRequired.Group("/senders")
			{
				senders.POST("", senderHandler.CreateSender)
				senders.POST("/:senderId/accounts/:accountId", senderHandler.AddSenderToAccount)
			}

			// Template management
			templates := authRequired.Group("/templates")
			{
				templates.POST("", templateHandler.CreateTemplate)
				templates.GET("", templateHandler.GetTemplates)
				templates.GET("/:id", templateHandler.GetTemplate)
				templates.PUT("/:id", templateHandler.UpdateTemplate)
				templates.DELETE("/:id", templateHandler.DeleteTemplate)
				templates.POST("/:id/preview", templateHandler.PreviewTemplate)
			}

			// Recipient management
			recipients := authRequired.Group("/recipients")
			{
				recipients.POST("", recipientHandler.CreateRecipient)
				recipients.GET("", recipientHandler.ListRecipients)
				recipients.GET("/:id", recipientHandler.GetRecipient)
				recipients.PUT("/:id", recipientHandler.UpdateRecipient)
				recipients.DELETE("/:id", recipientHandler.DeleteRecipient)
			}

			// Recipient Group management
			groups := authRequired.Group("/recipient-groups")
			{
				groups.POST("", groupHandler.CreateRecipientGroup)
				groups.GET("", groupHandler.ListRecipientGroups)
				groups.GET("/:id", groupHandler.GetRecipientGroup)
				groups.PUT("/:id", groupHandler.UpdateRecipientGroup)
				groups.DELETE("/:id", groupHandler.DeleteRecipientGroup)
				groups.POST("/:id/members", groupHandler.AddRecipientGroupMembers)
				groups.DELETE("/:id/members", groupHandler.RemoveRecipientGroupMembers)
			}

			// Task management
			tasks := authRequired.Group("/tasks")
			{
				tasks.POST("", taskHandler.CreateEmailTask)
				tasks.GET("/:id", statisticsHandler.GetTaskSummary)
				tasks.GET("/:id/records", statisticsHandler.GetTaskRecords)
			}

			// Email management
			emails := authRequired.Group("/emails")
			{
				emails.POST("/send", emailHandler.SendSingleEmail)
			}

			// Statistics management
			authRequired.GET("/statistics", statisticsHandler.GetStatistics)
		}
	}

	// Start server
	log.Printf("Starting server on port %s", cfg.Server.Port)
	if err := r.Run(cfg.Server.Port); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
