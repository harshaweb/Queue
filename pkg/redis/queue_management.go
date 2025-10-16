package redis

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/harshaweb/Queue/pkg/queue"
)

// CreateQueue creates a new queue with the given configuration
func (sq *StreamQueue) CreateQueue(ctx context.Context, config *queue.QueueConfig) error {
	// Store queue configuration
	configKey := sq.client.QueueConfigKey(config.Name)
	configData, err := json.Marshal(config)
	if err != nil {
		return fmt.Errorf("failed to serialize queue config: %w", err)
	}

	// Check if queue already exists
	exists, err := sq.client.Get().Exists(ctx, configKey).Result()
	if err != nil {
		return fmt.Errorf("failed to check queue existence: %w", err)
	}
	if exists > 0 {
		return queue.ErrQueueExists
	}

	// Create the queue configuration
	pipe := sq.client.Get().Pipeline()
	pipe.Set(ctx, configKey, string(configData), 0)
	pipe.SAdd(ctx, sq.client.QueuesSetKey(), config.Name)

	// Initialize queue statistics
	statsKey := sq.client.QueueStatsKey(config.Name)
	pipe.HMSet(ctx, statsKey, map[string]interface{}{
		"name":             config.Name,
		"length":           0,
		"in_flight":        0,
		"processed":        0,
		"failed":           0,
		"consumer_count":   0,
		"dead_letter_size": 0,
		"created_at":       time.Now().Unix(),
		"last_activity":    time.Now().Unix(),
	})

	_, err = pipe.Exec(ctx)
	if err != nil {
		return fmt.Errorf("failed to create queue: %w", err)
	}

	// Ensure consumer group exists
	return sq.ensureConsumerGroup(ctx, config.Name)
}

// DeleteQueue removes a queue and all its messages
func (sq *StreamQueue) DeleteQueue(ctx context.Context, queueName string) error {
	// Get all keys related to this queue
	streamKey := sq.client.StreamKey(queueName)
	configKey := sq.client.QueueConfigKey(queueName)
	statsKey := sq.client.QueueStatsKey(queueName)
	scheduledKey := sq.client.ScheduledKey(queueName)
	dlqKey := sq.client.DeadLetterKey(queueName)

	pipe := sq.client.Get().Pipeline()
	pipe.Del(ctx, streamKey, configKey, statsKey, scheduledKey, dlqKey)
	pipe.SRem(ctx, sq.client.QueuesSetKey(), queueName)

	_, err := pipe.Exec(ctx)
	return err
}

// GetQueueStats returns statistics for a queue
func (sq *StreamQueue) GetQueueStats(ctx context.Context, queueName string) (*queue.QueueStats, error) {
	statsKey := sq.client.QueueStatsKey(queueName)
	streamKey := sq.client.StreamKey(queueName)
	consumerGroup := sq.client.ConsumerGroupKey(queueName)

	// Get basic stats from hash
	stats, err := sq.client.Get().HGetAll(ctx, statsKey).Result()
	if err != nil {
		return nil, fmt.Errorf("failed to get queue stats: %w", err)
	}

	if len(stats) == 0 {
		return nil, queue.ErrQueueNotFound
	}

	// Get real-time stream length
	streamInfo, err := sq.client.Get().XInfoStream(ctx, streamKey).Result()
	if err != nil && err != redis.Nil {
		return nil, fmt.Errorf("failed to get stream info: %w", err)
	}

	// Get pending messages count
	pending, err := sq.client.Get().XPending(ctx, streamKey, consumerGroup).Result()
	if err != nil && err != redis.Nil {
		return nil, fmt.Errorf("failed to get pending info: %w", err)
	}

	// Parse stats
	queueStats := &queue.QueueStats{
		Name: queueName,
	}

	if streamInfo != nil {
		queueStats.Length = streamInfo.Length
	}

	if pending != nil {
		queueStats.InFlight = pending.Count
	}

	// Parse stored stats
	if val, ok := stats["processed"]; ok {
		if processed, err := strconv.ParseInt(val, 10, 64); err == nil {
			queueStats.Processed = processed
		}
	}

	if val, ok := stats["failed"]; ok {
		if failed, err := strconv.ParseInt(val, 10, 64); err == nil {
			queueStats.Failed = failed
		}
	}

	if val, ok := stats["dead_letter_size"]; ok {
		if dlSize, err := strconv.ParseInt(val, 10, 64); err == nil {
			queueStats.DeadLetterSize = dlSize
		}
	}

	if val, ok := stats["last_activity"]; ok {
		if lastActivity, err := strconv.ParseInt(val, 10, 64); err == nil {
			queueStats.LastActivity = time.Unix(lastActivity, 0)
		}
	}

	// Get consumer count
	consumers, err := sq.GetConsumerGroups(ctx, queueName)
	if err == nil && len(consumers) > 0 {
		queueStats.ConsumerCount = consumers[0].Consumers
	}

	return queueStats, nil
}

// ListQueues returns all queue names
func (sq *StreamQueue) ListQueues(ctx context.Context) ([]string, error) {
	queues, err := sq.client.Get().SMembers(ctx, sq.client.QueuesSetKey()).Result()
	if err != nil {
		return nil, fmt.Errorf("failed to list queues: %w", err)
	}
	return queues, nil
}

// PauseQueue pauses message consumption for a queue
func (sq *StreamQueue) PauseQueue(ctx context.Context, queueName string) error {
	configKey := sq.client.QueueConfigKey(queueName)

	// Get current config
	configData, err := sq.client.Get().Get(ctx, configKey).Result()
	if err != nil {
		if err == redis.Nil {
			return queue.ErrQueueNotFound
		}
		return fmt.Errorf("failed to get queue config: %w", err)
	}

	var config queue.QueueConfig
	if err := json.Unmarshal([]byte(configData), &config); err != nil {
		return fmt.Errorf("failed to parse queue config: %w", err)
	}

	// Mark as paused by setting a special key
	pauseKey := sq.client.GetShardKey("paused:" + queueName)
	return sq.client.Get().Set(ctx, pauseKey, "true", 0).Err()
}

// ResumeQueue resumes message consumption for a queue
func (sq *StreamQueue) ResumeQueue(ctx context.Context, queueName string) error {
	pauseKey := sq.client.GetShardKey("paused:" + queueName)
	return sq.client.Get().Del(ctx, pauseKey).Err()
}

// GetConsumerGroups returns consumer group information
func (sq *StreamQueue) GetConsumerGroups(ctx context.Context, queueName string) ([]queue.ConsumerGroup, error) {
	streamKey := sq.client.StreamKey(queueName)

	groups, err := sq.client.Get().XInfoGroups(ctx, streamKey).Result()
	if err != nil {
		if err == redis.Nil {
			return []queue.ConsumerGroup{}, nil
		}
		return nil, fmt.Errorf("failed to get consumer groups: %w", err)
	}

	var result []queue.ConsumerGroup
	for _, group := range groups {
		cg := queue.ConsumerGroup{
			Name:      group.Name,
			Consumers: int(group.Consumers),
			Pending:   group.Pending,
			LastID:    group.LastDeliveredID,
		}
		result = append(result, cg)
	}

	return result, nil
}

// HealthCheck verifies the queue system is healthy
func (sq *StreamQueue) HealthCheck(ctx context.Context) error {
	return sq.client.Ping(ctx)
}

// CleanupExpiredMessages removes expired messages and handles cleanup
func (sq *StreamQueue) CleanupExpiredMessages(ctx context.Context) error {
	queues, err := sq.ListQueues(ctx)
	if err != nil {
		return err
	}

	for _, queueName := range queues {
		if err := sq.cleanupQueueMessages(ctx, queueName); err != nil {
			// Log error but continue with other queues
			continue
		}
	}

	return nil
}

func (sq *StreamQueue) cleanupQueueMessages(ctx context.Context, queueName string) error {
	streamKey := sq.client.StreamKey(queueName)
	consumerGroup := sq.client.ConsumerGroupKey(queueName)

	// Clean up old acknowledged messages
	cutoff := time.Now().Add(-24 * time.Hour).UnixMilli()
	cutoffID := fmt.Sprintf("%d-0", cutoff)

	_, err := sq.client.Get().XTrimMinID(ctx, streamKey, cutoffID).Result()
	if err != nil && err != redis.Nil {
		return fmt.Errorf("failed to trim stream: %w", err)
	}

	// Clean up stale pending messages (older than visibility timeout)
	staleTime := time.Now().Add(-sq.config.DefaultVisibilityTimeout * 2)
	staleID := fmt.Sprintf("%d-0", staleTime.UnixMilli())

	// Get pending messages older than stale time
	pending, err := sq.client.Get().XPendingExt(ctx, &redis.XPendingExtArgs{
		Stream: streamKey,
		Group:  consumerGroup,
		Start:  "-",
		End:    staleID,
		Count:  100,
	}).Result()

	if err != nil && err != redis.Nil {
		return fmt.Errorf("failed to get pending messages: %w", err)
	}

	// Auto-claim stale messages to reset their idle time
	if len(pending) > 0 {
		msgIDs := make([]string, len(pending))
		for i, p := range pending {
			msgIDs[i] = p.ID
		}

		_, err = sq.client.Get().XClaimJustID(ctx, &redis.XClaimArgs{
			Stream:   streamKey,
			Group:    consumerGroup,
			Consumer: "cleanup",
			MinIdle:  sq.config.DefaultVisibilityTimeout,
			Messages: msgIDs,
		}).Result()

		if err != nil && err != redis.Nil {
			return fmt.Errorf("failed to claim stale messages: %w", err)
		}
	}

	return nil
}

// Close closes the queue and cleans up resources
func (sq *StreamQueue) Close() error {
	return sq.client.Close()
}
