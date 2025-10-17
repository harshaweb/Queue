package pkg

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"sync"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/google/uuid"
)

// MessageScheduler handles scheduled message delivery (Redis-backed)
type MessageScheduler struct {
	rdb           *redis.Client
	queueName     string
	schedulerName string
	ticker        *time.Ticker
	ctx           context.Context
	cancel        context.CancelFunc
	mu            sync.RWMutex
	isRunning     bool
}

// RedisScheduledMessage represents a message scheduled for future delivery in Redis
type RedisScheduledMessage struct {
	ID          string                 `json:"id"`
	QueueName   string                 `json:"queue_name"`
	Payload     json.RawMessage        `json:"payload"`
	Headers     map[string]string      `json:"headers"`
	ScheduledAt time.Time              `json:"scheduled_at"`
	DeliverAt   time.Time              `json:"deliver_at"`
	Priority    int                    `json:"priority"`
	Metadata    map[string]interface{} `json:"metadata"`
}

// RecurringSchedule defines recurring message patterns
type RecurringSchedule struct {
	ID          string            `json:"id"`
	QueueName   string            `json:"queue_name"`
	Payload     json.RawMessage   `json:"payload"`
	Headers     map[string]string `json:"headers"`
	CronPattern string            `json:"cron_pattern"`
	Interval    time.Duration     `json:"interval"`
	NextRun     time.Time         `json:"next_run"`
	Enabled     bool              `json:"enabled"`
	MaxRuns     int               `json:"max_runs"`
	RunCount    int               `json:"run_count"`
}

// NewMessageScheduler creates a new message scheduler
func NewMessageScheduler(rdb *redis.Client, queueName string) *MessageScheduler {
	ctx, cancel := context.WithCancel(context.Background())

	return &MessageScheduler{
		rdb:           rdb,
		queueName:     queueName,
		schedulerName: fmt.Sprintf("%s:scheduler", queueName),
		ticker:        time.NewTicker(time.Second), // Check every second
		ctx:           ctx,
		cancel:        cancel,
	}
}

// Start begins the scheduler
func (ms *MessageScheduler) Start() error {
	ms.mu.Lock()
	defer ms.mu.Unlock()

	if ms.isRunning {
		return fmt.Errorf("scheduler is already running")
	}

	ms.isRunning = true
	go ms.run()
	return nil
}

// Stop stops the scheduler
func (ms *MessageScheduler) Stop() error {
	ms.mu.Lock()
	defer ms.mu.Unlock()

	if !ms.isRunning {
		return fmt.Errorf("scheduler is not running")
	}

	ms.cancel()
	ms.ticker.Stop()
	ms.isRunning = false
	return nil
}

// ScheduleMessage schedules a message for future delivery
func (ms *MessageScheduler) ScheduleMessage(ctx context.Context, msg *RedisScheduledMessage) error {
	// Validate message
	if msg.DeliverAt.Before(time.Now()) {
		return fmt.Errorf("delivery time must be in the future")
	}

	if msg.ID == "" {
		msg.ID = fmt.Sprintf("sched_%d_%s", time.Now().UnixNano(), uuid.New().String()[:8])
	}

	msg.ScheduledAt = time.Now()

	// Serialize message
	data, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("failed to serialize scheduled message: %v", err)
	}

	// Store in Redis sorted set with delivery time as score
	score := float64(msg.DeliverAt.Unix())
	key := fmt.Sprintf("%s:scheduled", ms.queueName)

	return ms.rdb.ZAdd(ctx, key, &redis.Z{
		Score:  score,
		Member: data,
	}).Err()
}

// ScheduleRecurring schedules a recurring message
func (ms *MessageScheduler) ScheduleRecurring(ctx context.Context, schedule *RecurringSchedule) error {
	if schedule.ID == "" {
		schedule.ID = fmt.Sprintf("recurring_%s", uuid.New().String())
	}

	// Calculate next run time
	if schedule.NextRun.IsZero() {
		if schedule.Interval > 0 {
			schedule.NextRun = time.Now().Add(schedule.Interval)
		} else {
			// For cron patterns, calculate next run (simplified)
			schedule.NextRun = time.Now().Add(time.Hour) // Default to 1 hour
		}
	}

	schedule.Enabled = true

	// Store recurring schedule
	data, err := json.Marshal(schedule)
	if err != nil {
		return fmt.Errorf("failed to serialize recurring schedule: %v", err)
	}

	key := fmt.Sprintf("%s:recurring", ms.queueName)
	return ms.rdb.HSet(ctx, key, schedule.ID, data).Err()
}

// run is the main scheduler loop
func (ms *MessageScheduler) run() {
	for {
		select {
		case <-ms.ctx.Done():
			return
		case <-ms.ticker.C:
			ms.processScheduledMessages()
			ms.processRecurringMessages()
		}
	}
}

// processScheduledMessages processes due scheduled messages
func (ms *MessageScheduler) processScheduledMessages() {
	ctx := context.Background()
	now := time.Now()
	key := fmt.Sprintf("%s:scheduled", ms.queueName)

	// Get messages due for delivery
	result := ms.rdb.ZRangeByScore(ctx, key, &redis.ZRangeBy{
		Min: "0",
		Max: strconv.FormatInt(now.Unix(), 10),
	})

	messages, err := result.Result()
	if err != nil {
		return
	}

	for _, msgData := range messages {
		var msg RedisScheduledMessage
		if err := json.Unmarshal([]byte(msgData), &msg); err != nil {
			continue
		}

		// Deliver message to queue
		err = ms.deliverScheduledMessage(ctx, &msg)
		if err != nil {
			continue
		}

		// Remove from scheduled set
		ms.rdb.ZRem(ctx, key, msgData)
	}
}

// processRecurringMessages processes recurring message schedules
func (ms *MessageScheduler) processRecurringMessages() {
	ctx := context.Background()
	key := fmt.Sprintf("%s:recurring", ms.queueName)

	// Get all recurring schedules
	result := ms.rdb.HGetAll(ctx, key)
	schedules, err := result.Result()
	if err != nil {
		return
	}

	now := time.Now()

	for scheduleID, scheduleData := range schedules {
		var schedule RecurringSchedule
		if err := json.Unmarshal([]byte(scheduleData), &schedule); err != nil {
			continue
		}

		// Check if it's time to run
		if !schedule.Enabled || now.Before(schedule.NextRun) {
			continue
		}

		// Check if max runs reached
		if schedule.MaxRuns > 0 && schedule.RunCount >= schedule.MaxRuns {
			// Disable schedule
			schedule.Enabled = false
			updatedData, _ := json.Marshal(schedule)
			ms.rdb.HSet(ctx, key, scheduleID, updatedData)
			continue
		}

		// Create scheduled message
		msg := &RedisScheduledMessage{
			ID:        fmt.Sprintf("recurring_%s", uuid.New().String()),
			QueueName: schedule.QueueName,
			Payload:   schedule.Payload,
			Headers:   schedule.Headers,
			DeliverAt: now,
			Metadata: map[string]interface{}{
				"recurring_id": schedule.ID,
				"run_count":    schedule.RunCount + 1,
			},
		}

		// Deliver message
		err = ms.deliverScheduledMessage(ctx, msg)
		if err != nil {
			continue
		}

		// Update schedule
		schedule.RunCount++
		schedule.NextRun = now.Add(schedule.Interval)

		updatedData, _ := json.Marshal(schedule)
		ms.rdb.HSet(ctx, key, scheduleID, updatedData)
	}
}

// deliverScheduledMessage delivers a scheduled message to the queue
func (ms *MessageScheduler) deliverScheduledMessage(ctx context.Context, msg *RedisScheduledMessage) error {
	// Create message for queue
	queueMsg := map[string]interface{}{
		"id":         msg.ID,
		"payload":    msg.Payload,
		"headers":    msg.Headers,
		"timestamp":  time.Now().Unix(),
		"scheduled":  true,
		"deliver_at": msg.DeliverAt.Unix(),
		"metadata":   msg.Metadata,
	}

	if msg.Priority > 0 {
		queueMsg["priority"] = msg.Priority
	}

	data, err := json.Marshal(queueMsg)
	if err != nil {
		return err
	}

	// Add to main queue stream
	return ms.rdb.XAdd(ctx, &redis.XAddArgs{
		Stream: msg.QueueName,
		Values: map[string]interface{}{
			"data": string(data),
		},
	}).Err()
}

// GetScheduledMessages returns all scheduled messages
func (ms *MessageScheduler) GetScheduledMessages(ctx context.Context) ([]*RedisScheduledMessage, error) {
	key := fmt.Sprintf("%s:scheduled", ms.queueName)

	result := ms.rdb.ZRangeWithScores(ctx, key, 0, -1)
	items, err := result.Result()
	if err != nil {
		return nil, err
	}

	var messages []*RedisScheduledMessage
	for _, item := range items {
		var msg RedisScheduledMessage
		if err := json.Unmarshal([]byte(item.Member.(string)), &msg); err != nil {
			continue
		}
		messages = append(messages, &msg)
	}

	return messages, nil
}

// GetRecurringSchedules returns all recurring schedules
func (ms *MessageScheduler) GetRecurringSchedules(ctx context.Context) ([]*RecurringSchedule, error) {
	key := fmt.Sprintf("%s:recurring", ms.queueName)

	result := ms.rdb.HGetAll(ctx, key)
	schedules, err := result.Result()
	if err != nil {
		return nil, err
	}

	var recurringSchedules []*RecurringSchedule
	for _, scheduleData := range schedules {
		var schedule RecurringSchedule
		if err := json.Unmarshal([]byte(scheduleData), &schedule); err != nil {
			continue
		}
		recurringSchedules = append(recurringSchedules, &schedule)
	}

	return recurringSchedules, nil
}

// CancelScheduledMessage cancels a scheduled message
func (ms *MessageScheduler) CancelScheduledMessage(ctx context.Context, messageID string) error {
	key := fmt.Sprintf("%s:scheduled", ms.queueName)

	// Get all scheduled messages to find the one to cancel
	result := ms.rdb.ZRange(ctx, key, 0, -1)
	messages, err := result.Result()
	if err != nil {
		return err
	}

	for _, msgData := range messages {
		var msg RedisScheduledMessage
		if err := json.Unmarshal([]byte(msgData), &msg); err != nil {
			continue
		}

		if msg.ID == messageID {
			return ms.rdb.ZRem(ctx, key, msgData).Err()
		}
	}

	return fmt.Errorf("scheduled message with ID %s not found", messageID)
}

// DisableRecurringSchedule disables a recurring schedule
func (ms *MessageScheduler) DisableRecurringSchedule(ctx context.Context, scheduleID string) error {
	key := fmt.Sprintf("%s:recurring", ms.queueName)

	// Get the schedule
	result := ms.rdb.HGet(ctx, key, scheduleID)
	scheduleData, err := result.Result()
	if err != nil {
		return err
	}

	var schedule RecurringSchedule
	if err := json.Unmarshal([]byte(scheduleData), &schedule); err != nil {
		return err
	}

	// Disable it
	schedule.Enabled = false

	// Save back
	updatedData, err := json.Marshal(schedule)
	if err != nil {
		return err
	}

	return ms.rdb.HSet(ctx, key, scheduleID, updatedData).Err()
}

// SchedulerStats provides scheduler statistics
type SchedulerStats struct {
	IsRunning          bool `json:"is_running"`
	ScheduledMessages  int  `json:"scheduled_messages"`
	RecurringSchedules int  `json:"recurring_schedules"`
}

// Stats returns scheduler statistics
func (ms *MessageScheduler) Stats(ctx context.Context) SchedulerStats {
	ms.mu.RLock()
	defer ms.mu.RUnlock()

	stats := SchedulerStats{
		IsRunning: ms.isRunning,
	}

	// Count scheduled messages
	key := fmt.Sprintf("%s:scheduled", ms.queueName)
	if count := ms.rdb.ZCard(ctx, key); count.Err() == nil {
		stats.ScheduledMessages = int(count.Val())
	}

	// Count recurring schedules
	key = fmt.Sprintf("%s:recurring", ms.queueName)
	if count := ms.rdb.HLen(ctx, key); count.Err() == nil {
		stats.RecurringSchedules = int(count.Val())
	}

	return stats
}
