package pkg

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/go-redis/redis/v8"
)

// DeadLetterQueue handles failed messages
type DeadLetterQueue struct {
	rdb  *redis.Client
	name string
}

// NewDeadLetterQueue creates a new dead letter queue
func NewDeadLetterQueue(rdb *redis.Client, name string) *DeadLetterQueue {
	return &DeadLetterQueue{
		rdb:  rdb,
		name: name,
	}
}

// Initialize sets up the DLQ
func (dlq *DeadLetterQueue) Initialize(ctx context.Context) error {
	// Ensure the DLQ stream exists
	err := dlq.rdb.XGroupCreateMkStream(ctx, dlq.name, "dlq-group", "$").Err()
	if err != nil && err.Error() != "BUSYGROUP Consumer Group name already exists" {
		return fmt.Errorf("failed to create DLQ group: %v", err)
	}
	return nil
}

// Send moves a message to the dead letter queue
func (dlq *DeadLetterQueue) Send(msg *Message, reason string) error {
	ctx := context.Background()

	dlqData := map[string]interface{}{
		"original_id":    msg.ID,
		"payload":        dlq.serializePayload(msg.Payload),
		"headers":        dlq.serializeHeaders(msg.Headers),
		"priority":       msg.Priority,
		"retry_count":    msg.RetryCount,
		"max_retries":    msg.MaxRetries,
		"created_at":     msg.CreatedAt.Unix(),
		"failed_at":      time.Now().Unix(),
		"failure_reason": reason,
		"attempts":       msg.Attempts,
	}

	if msg.Deadline != nil {
		dlqData["deadline"] = msg.Deadline.Unix()
	}

	result := dlq.rdb.XAdd(ctx, &redis.XAddArgs{
		Stream: dlq.name,
		Values: dlqData,
	})

	return result.Err()
}

// GetFailedMessages retrieves messages from the DLQ
func (dlq *DeadLetterQueue) GetFailedMessages(limit int64) ([]*Message, error) {
	ctx := context.Background()

	result := dlq.rdb.XRange(ctx, dlq.name, "-", "+").Val()

	var messages []*Message
	for i, streamMsg := range result {
		if limit > 0 && int64(i) >= limit {
			break
		}

		msg, err := dlq.parseMessage(streamMsg)
		if err != nil {
			continue
		}

		messages = append(messages, msg)
	}

	return messages, nil
}

// Retry reprocesses a message from the DLQ
func (dlq *DeadLetterQueue) Retry(messageID string, targetQueue *Queue) error {
	ctx := context.Background()

	// Find the message in DLQ
	result := dlq.rdb.XRange(ctx, dlq.name, messageID, messageID).Val()
	if len(result) == 0 {
		return fmt.Errorf("message not found in DLQ")
	}

	msg, err := dlq.parseMessage(result[0])
	if err != nil {
		return fmt.Errorf("failed to parse DLQ message: %v", err)
	}

	// Reset retry count
	msg.RetryCount = 0
	msg.Attempts = 0
	msg.ProcessedAt = nil

	// Send back to original queue
	_, err = targetQueue.SendMessage(msg)
	if err != nil {
		return fmt.Errorf("failed to retry message: %v", err)
	}

	// Remove from DLQ
	return dlq.rdb.XDel(ctx, dlq.name, messageID).Err()
}

// Clear removes all messages from the DLQ
func (dlq *DeadLetterQueue) Clear() error {
	ctx := context.Background()
	return dlq.rdb.Del(ctx, dlq.name).Err()
}

// GetSize returns the number of messages in the DLQ
func (dlq *DeadLetterQueue) GetSize() (int64, error) {
	ctx := context.Background()
	return dlq.rdb.XLen(ctx, dlq.name).Result()
}

// Helper methods
func (dlq *DeadLetterQueue) serializePayload(payload map[string]interface{}) string {
	if payload == nil {
		return "{}"
	}
	data, _ := json.Marshal(payload)
	return string(data)
}

func (dlq *DeadLetterQueue) serializeHeaders(headers map[string]string) string {
	if headers == nil {
		return "{}"
	}
	data, _ := json.Marshal(headers)
	return string(data)
}

func (dlq *DeadLetterQueue) parseMessage(streamMsg redis.XMessage) (*Message, error) {
	msg := &Message{
		StreamID: streamMsg.ID,
	}

	if id, ok := streamMsg.Values["original_id"].(string); ok {
		msg.ID = id
	}

	if payloadStr, ok := streamMsg.Values["payload"].(string); ok {
		var payload map[string]interface{}
		if err := json.Unmarshal([]byte(payloadStr), &payload); err == nil {
			msg.Payload = payload
		}
	}

	if headersStr, ok := streamMsg.Values["headers"].(string); ok {
		var headers map[string]string
		if err := json.Unmarshal([]byte(headersStr), &headers); err == nil {
			msg.Headers = headers
		}
	}

	if priority, ok := streamMsg.Values["priority"].(string); ok {
		if p, err := json.Number(priority).Int64(); err == nil {
			msg.Priority = int(p)
		}
	}

	if retryCount, ok := streamMsg.Values["retry_count"].(string); ok {
		if rc, err := json.Number(retryCount).Int64(); err == nil {
			msg.RetryCount = int(rc)
		}
	}

	if maxRetries, ok := streamMsg.Values["max_retries"].(string); ok {
		if mr, err := json.Number(maxRetries).Int64(); err == nil {
			msg.MaxRetries = int(mr)
		}
	}

	if attempts, ok := streamMsg.Values["attempts"].(string); ok {
		if a, err := json.Number(attempts).Int64(); err == nil {
			msg.Attempts = int(a)
		}
	}

	if createdAt, ok := streamMsg.Values["created_at"].(string); ok {
		if ts, err := json.Number(createdAt).Int64(); err == nil {
			msg.CreatedAt = time.Unix(ts, 0)
		}
	}

	if deadline, ok := streamMsg.Values["deadline"].(string); ok {
		if ts, err := json.Number(deadline).Int64(); err == nil {
			deadlineTime := time.Unix(ts, 0)
			msg.Deadline = &deadlineTime
		}
	}

	return msg, nil
}
