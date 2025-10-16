package api

import (
	"context"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/queue-system/redis-queue/internal/auth"
	"github.com/queue-system/redis-queue/pkg/queue"
)

// RestAPI provides REST endpoints for the queue system
type RestAPI struct {
	queue  queue.QueueManager
	auth   auth.Authenticator
	config *RestConfig
}

// RestConfig contains REST API configuration
type RestConfig struct {
	EnableAuth     bool
	AllowedOrigins []string
	RateLimit      int
	Timeout        time.Duration
}

// NewRestAPI creates a new REST API instance
func NewRestAPI(queueManager queue.QueueManager, authenticator auth.Authenticator, config *RestConfig) *RestAPI {
	return &RestAPI{
		queue:  queueManager,
		auth:   authenticator,
		config: config,
	}
}

// SetupRoutes configures the Gin router with queue endpoints
func (api *RestAPI) SetupRoutes() *gin.Engine {
	r := gin.Default()

	// Middleware
	r.Use(api.corsMiddleware())
	r.Use(api.timeoutMiddleware())

	// Health check
	r.GET("/health", api.healthCheck)
	r.GET("/ready", api.readinessCheck)

	// API v1 routes
	v1 := r.Group("/api/v1")

	// Queue operations
	queues := v1.Group("/queues")
	if api.config.EnableAuth {
		queues.Use(api.authMiddleware())
	}

	// Message operations
	queues.POST("/:queue/messages", api.enqueueMessage)
	queues.POST("/:queue/messages/batch", api.batchEnqueueMessages)
	queues.POST("/:queue/consume", api.consumeMessage)
	queues.POST("/:queue/messages/:messageId/ack", api.ackMessage)
	queues.POST("/:queue/messages/:messageId/nack", api.nackMessage)

	// Advanced operations
	queues.POST("/:queue/messages/:messageId/move-to-last", api.moveToLast)
	queues.POST("/:queue/messages/:messageId/skip", api.skipMessage)
	queues.POST("/:queue/messages/:messageId/requeue", api.requeueMessage)
	queues.POST("/:queue/messages/:messageId/dead-letter", api.moveToDeadLetter)

	// Queue management
	queues.POST("", api.createQueue)
	queues.DELETE("/:queue", api.deleteQueue)
	queues.GET("/:queue/stats", api.getQueueStats)
	queues.GET("", api.listQueues)

	// Admin operations (require admin auth)
	admin := queues.Group("")
	if api.config.EnableAuth {
		admin.Use(api.adminAuthMiddleware())
	}

	admin.POST("/:queue/pause", api.pauseQueue)
	admin.POST("/:queue/resume", api.resumeQueue)
	admin.GET("/:queue/consumer-groups", api.getConsumerGroups)

	return r
}

// Message operations

func (api *RestAPI) enqueueMessage(c *gin.Context) {
	queueName := c.Param("queue")

	var req struct {
		Payload    map[string]interface{} `json:"payload" binding:"required"`
		Priority   *int                   `json:"priority,omitempty"`
		ScheduleAt *time.Time             `json:"schedule_at,omitempty"`
		Delay      *string                `json:"delay,omitempty"`
		TTL        *string                `json:"ttl,omitempty"`
		Headers    map[string]string      `json:"headers,omitempty"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Create message
	message := queue.NewMessage(req.Payload, req.Headers)

	// Create options
	opts := &queue.MessageOptions{
		Headers: req.Headers,
	}

	if req.Priority != nil {
		opts.Priority = *req.Priority
	}

	if req.ScheduleAt != nil {
		opts.ScheduleAt = req.ScheduleAt
	}

	if req.Delay != nil {
		duration, err := time.ParseDuration(*req.Delay)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid delay format"})
			return
		}
		opts.Delay = &duration
	}

	if req.TTL != nil {
		duration, err := time.ParseDuration(*req.TTL)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid TTL format"})
			return
		}
		opts.TTL = &duration
	}

	// Enqueue message
	err := api.queue.Enqueue(c.Request.Context(), queueName, message, opts)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"message_id": message.ID,
		"queue_name": queueName,
	})
}

func (api *RestAPI) batchEnqueueMessages(c *gin.Context) {
	queueName := c.Param("queue")

	var req struct {
		Messages []map[string]interface{} `json:"messages" binding:"required"`
		Priority *int                     `json:"priority,omitempty"`
		Headers  map[string]string        `json:"headers,omitempty"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Create messages
	var messages []*queue.Message
	var messageIDs []string

	for _, payload := range req.Messages {
		message := queue.NewMessage(payload, req.Headers)
		messages = append(messages, message)
		messageIDs = append(messageIDs, message.ID)
	}

	// Create options
	opts := &queue.MessageOptions{
		Headers: req.Headers,
	}

	if req.Priority != nil {
		opts.Priority = *req.Priority
	}

	// Batch enqueue
	err := api.queue.BatchEnqueue(c.Request.Context(), queueName, messages, opts)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"message_ids": messageIDs,
		"queue_name":  queueName,
		"count":       len(messageIDs),
	})
}

func (api *RestAPI) consumeMessage(c *gin.Context) {
	queueName := c.Param("queue")

	var req struct {
		VisibilityTimeout *string `json:"visibility_timeout,omitempty"`
		MaxRetries        *int    `json:"max_retries,omitempty"`
		BatchSize         *int    `json:"batch_size,omitempty"`
		LongPoll          *bool   `json:"long_poll,omitempty"`
		LongPollTimeout   *string `json:"long_poll_timeout,omitempty"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		// Allow empty body for consume
	}

	// Create consume options
	opts := &queue.ConsumeOptions{
		VisibilityTimeout: 30 * time.Second,
		MaxRetries:        3,
		BatchSize:         1,
		LongPoll:          false,
	}

	if req.VisibilityTimeout != nil {
		duration, err := time.ParseDuration(*req.VisibilityTimeout)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid visibility_timeout format"})
			return
		}
		opts.VisibilityTimeout = duration
	}

	if req.MaxRetries != nil {
		opts.MaxRetries = *req.MaxRetries
	}

	if req.BatchSize != nil {
		opts.BatchSize = *req.BatchSize
	}

	if req.LongPoll != nil {
		opts.LongPoll = *req.LongPoll
	}

	if req.LongPollTimeout != nil {
		duration, err := time.ParseDuration(*req.LongPollTimeout)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid long_poll_timeout format"})
			return
		}
		opts.LongPollTimeout = duration
	}

	// Consume message(s)
	if opts.BatchSize == 1 {
		message, err := api.queue.Dequeue(c.Request.Context(), queueName, opts)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		if message == nil {
			c.JSON(http.StatusNoContent, nil)
			return
		}

		c.JSON(http.StatusOK, message)
	} else {
		messages, err := api.queue.BatchDequeue(c.Request.Context(), queueName, opts.BatchSize, opts)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		if len(messages) == 0 {
			c.JSON(http.StatusNoContent, nil)
			return
		}

		c.JSON(http.StatusOK, gin.H{"messages": messages})
	}
}

func (api *RestAPI) ackMessage(c *gin.Context) {
	queueName := c.Param("queue")
	messageID := c.Param("messageId")

	err := api.queue.Ack(c.Request.Context(), queueName, messageID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true})
}

func (api *RestAPI) nackMessage(c *gin.Context) {
	queueName := c.Param("queue")
	messageID := c.Param("messageId")

	var req struct {
		Requeue *bool `json:"requeue,omitempty"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		// Allow empty body, default to requeue=false
	}

	requeue := false
	if req.Requeue != nil {
		requeue = *req.Requeue
	}

	err := api.queue.Nack(c.Request.Context(), queueName, messageID, requeue)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true})
}

// Advanced operations

func (api *RestAPI) moveToLast(c *gin.Context) {
	queueName := c.Param("queue")
	messageID := c.Param("messageId")

	err := api.queue.MoveToLast(c.Request.Context(), queueName, messageID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true})
}

func (api *RestAPI) skipMessage(c *gin.Context) {
	queueName := c.Param("queue")
	messageID := c.Param("messageId")

	var req struct {
		Duration *string `json:"duration,omitempty"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		// Allow empty body
	}

	var duration *time.Duration
	if req.Duration != nil {
		d, err := time.ParseDuration(*req.Duration)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid duration format"})
			return
		}
		duration = &d
	}

	err := api.queue.Skip(c.Request.Context(), queueName, messageID, duration)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true})
}

func (api *RestAPI) requeueMessage(c *gin.Context) {
	queueName := c.Param("queue")
	messageID := c.Param("messageId")

	var req struct {
		Delay *string `json:"delay,omitempty"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		// Allow empty body
	}

	var delay *time.Duration
	if req.Delay != nil {
		d, err := time.ParseDuration(*req.Delay)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid delay format"})
			return
		}
		delay = &d
	}

	err := api.queue.Requeue(c.Request.Context(), queueName, messageID, delay)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true})
}

func (api *RestAPI) moveToDeadLetter(c *gin.Context) {
	queueName := c.Param("queue")
	messageID := c.Param("messageId")

	err := api.queue.MoveToDeadLetter(c.Request.Context(), queueName, messageID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true})
}

// Queue management

func (api *RestAPI) createQueue(c *gin.Context) {
	var req queue.QueueConfig

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	err := api.queue.CreateQueue(c.Request.Context(), &req)
	if err != nil {
		if err == queue.ErrQueueExists {
			c.JSON(http.StatusConflict, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"success": true})
}

func (api *RestAPI) deleteQueue(c *gin.Context) {
	queueName := c.Param("queue")

	err := api.queue.DeleteQueue(c.Request.Context(), queueName)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true})
}

func (api *RestAPI) getQueueStats(c *gin.Context) {
	queueName := c.Param("queue")

	stats, err := api.queue.GetQueueStats(c.Request.Context(), queueName)
	if err != nil {
		if err == queue.ErrQueueNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, stats)
}

func (api *RestAPI) listQueues(c *gin.Context) {
	queues, err := api.queue.ListQueues(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"queues": queues})
}

// Admin operations

func (api *RestAPI) pauseQueue(c *gin.Context) {
	queueName := c.Param("queue")

	err := api.queue.PauseQueue(c.Request.Context(), queueName)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true})
}

func (api *RestAPI) resumeQueue(c *gin.Context) {
	queueName := c.Param("queue")

	err := api.queue.ResumeQueue(c.Request.Context(), queueName)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true})
}

func (api *RestAPI) getConsumerGroups(c *gin.Context) {
	queueName := c.Param("queue")

	groups, err := api.queue.GetConsumerGroups(c.Request.Context(), queueName)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"consumer_groups": groups})
}

// Health checks

func (api *RestAPI) healthCheck(c *gin.Context) {
	err := api.queue.HealthCheck(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"healthy": false,
			"error":   err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"healthy": true,
		"status":  "ok",
	})
}

func (api *RestAPI) readinessCheck(c *gin.Context) {
	// Additional readiness checks can be added here
	api.healthCheck(c)
}

// Middleware

func (api *RestAPI) corsMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		origin := c.Request.Header.Get("Origin")
		if len(api.config.AllowedOrigins) == 0 || contains(api.config.AllowedOrigins, origin) {
			c.Header("Access-Control-Allow-Origin", origin)
		}
		c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Content-Type, Authorization")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}

		c.Next()
	}
}

func (api *RestAPI) timeoutMiddleware() gin.HandlerFunc {
	return gin.HandlerFunc(func(c *gin.Context) {
		ctx, cancel := context.WithTimeout(c.Request.Context(), api.config.Timeout)
		defer cancel()
		c.Request = c.Request.WithContext(ctx)
		c.Next()
	})
}

func (api *RestAPI) authMiddleware() gin.HandlerFunc {
	return gin.HandlerFunc(func(c *gin.Context) {
		if !api.config.EnableAuth {
			c.Next()
			return
		}

		token := c.GetHeader("Authorization")
		if token == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "missing authorization header"})
			c.Abort()
			return
		}

		user, err := api.auth.ValidateToken(c.Request.Context(), token)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid token"})
			c.Abort()
			return
		}

		c.Set("user", user)
		c.Next()
	})
}

func (api *RestAPI) adminAuthMiddleware() gin.HandlerFunc {
	return gin.HandlerFunc(func(c *gin.Context) {
		if !api.config.EnableAuth {
			c.Next()
			return
		}

		user, exists := c.Get("user")
		if !exists {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "user not authenticated"})
			c.Abort()
			return
		}

		if !api.auth.HasAdminRole(user) {
			c.JSON(http.StatusForbidden, gin.H{"error": "admin role required"})
			c.Abort()
			return
		}

		c.Next()
	})
}

// Helper functions

func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}
