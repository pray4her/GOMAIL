package main

import (
	"context"
	"email-service/internal/config"
	"email-service/internal/handler"
	"email-service/internal/queue"
	"email-service/internal/repository"
	"email-service/internal/service"
	"email-service/pkg/database"
	"email-service/pkg/redis"
	"log"

	"github.com/gin-gonic/gin"
)

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

	// Initialize queue service
	queueService := queue.NewRedisQueueService(redisClient)

	// Initialize repositories
	accountRepo := repository.NewAccountRepository(db)
	senderRepo := repository.NewSenderRepository(db)
	templateRepo := repository.NewTemplateRepository(db)
	emailRecordRepo := repository.NewEmailSendRecordRepository(db)
	recipientRepo := repository.NewRecipientRepository(db)
	taskRepo := repository.NewEmailTaskRepository(db)

	// Initialize services
	accountService := service.NewAccountService(accountRepo)
	senderService := service.NewSenderService(senderRepo, accountRepo)
	templateService := service.NewTemplateService(templateRepo)
	recipientService := service.NewRecipientService(recipientRepo)

	// Initialize Email Service with sender repository and Aliyun endpoint
	emailService := service.NewEmailService(senderRepo, emailRecordRepo, queueService, cfg.Aliyun.Endpoint)

	// Initialize Email Task Service - now uses the generic queue service and template repo
	taskService := service.NewEmailTaskService(taskRepo, recipientRepo, templateRepo, queueService)

	// Initialize the new Task Dispatcher Service
	taskDispatcherService := service.NewTaskDispatcherService(taskRepo, emailRecordRepo, templateRepo, queueService)
	// Start the dispatcher in the background
	taskDispatcherService.Start(context.Background())

	// Initialize Email Worker Service
	emailWorker := service.NewEmailWorkerService(queueService, emailService)
	// Start the worker in the background with configured number of workers
	emailWorker.Start(context.Background(), cfg.Worker.EmailSenderCount)

	// Initialize handlers
	accountHandler := handler.NewAccountHandler(accountService)
	senderHandler := handler.NewSenderHandler(senderService)
	templateHandler := handler.NewTemplateHandler(templateService)
	emailHandler := handler.NewEmailHandler(emailService)
	recipientHandler := handler.NewRecipientHandler(recipientService)
	taskHandler := handler.NewEmailTaskHandler(taskService)

	// Setup router
	r := gin.Default()

	api := r.Group("/api/v1")
	{
		accounts := api.Group("/accounts")
		{
			accounts.POST("", accountHandler.CreateAccount)
			accounts.GET("", accountHandler.GetAccounts)
			accounts.GET("/:id", accountHandler.GetAccount)
			accounts.PUT("/:id", accountHandler.UpdateAccount)
			accounts.DELETE("/:id", accountHandler.DeleteAccount)
		}

		senders := api.Group("/senders")
		{
			senders.POST("", senderHandler.CreateSender)
			senders.POST("/:senderId/accounts/:accountId", senderHandler.AddSenderToAccount)
		}

		templates := api.Group("/templates")
		{
			templates.POST("", templateHandler.CreateTemplate)
			templates.GET("", templateHandler.GetTemplates)
			templates.GET("/:id", templateHandler.GetTemplate)
			templates.PUT("/:id", templateHandler.UpdateTemplate)
			templates.DELETE("/:id", templateHandler.DeleteTemplate)
		}

		recipients := api.Group("/recipients")
		{
			recipients.POST("", recipientHandler.CreateRecipient)
			recipients.GET("", recipientHandler.ListRecipients)
			recipients.GET("/:id", recipientHandler.GetRecipient)
			recipients.PUT("/:id", recipientHandler.UpdateRecipient)
			recipients.DELETE("/:id", recipientHandler.DeleteRecipient)
		}

		tasks := api.Group("/tasks")
		{
			tasks.POST("", taskHandler.CreateBatchTask)
		}

		emails := api.Group("/emails")
		{
			emails.POST("/send", emailHandler.SendSingleEmail)
		}
	}

	r.GET("/ping", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"message": "pong",
		})
	})

	// Start server
	log.Printf("Starting server on port %s", cfg.Server.Port)
	if err := r.Run(cfg.Server.Port); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
