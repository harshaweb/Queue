package pkg

import (
	"context"

	"github.com/go-redis/redis/v8"
)

// Additional Queue methods that complete the core functionality

// Nack negatively acknowledges a message (for manual retry)
func (q *Queue) Nack(streamID string) error {
	q.mu.RLock()
	defer q.mu.RUnlock()

	ctx := context.Background()
	// Move message back to pending
	return q.rdb.XClaim(ctx, &redis.XClaimArgs{
		Stream:   q.name,
		Group:    q.config.ConsumerGroup,
		Consumer: q.config.ConsumerName,
		MinIdle:  0,
		Messages: []string{streamID},
	}).Err()
}

// Peek returns messages without consuming them
func (q *Queue) Peek(count int64) ([]*Message, error) {
	q.mu.RLock()
	defer q.mu.RUnlock()

	ctx := context.Background()
	result := q.rdb.XRange(ctx, q.name, "-", "+").Val()

	var messages []*Message
	for i, streamMsg := range result {
		if count > 0 && int64(i) >= count {
			break
		}

		msg, err := q.parseMessage(streamMsg)
		if err != nil {
			continue
		}

		messages = append(messages, msg)
	}

	return messages, nil
}

// GetPendingMessages returns messages that are being processed but not acknowledged
func (q *Queue) GetPendingMessages() ([]*Message, error) {
	q.mu.RLock()
	defer q.mu.RUnlock()

	ctx := context.Background()
	result := q.rdb.XPending(ctx, q.name, q.config.ConsumerGroup).Val()

	if result.Count == 0 {
		return nil, nil
	}

	// Get detailed pending info
	pendingResult := q.rdb.XPendingExt(ctx, &redis.XPendingExtArgs{
		Stream: q.name,
		Group:  q.config.ConsumerGroup,
		Start:  "-",
		End:    "+",
		Count:  result.Count,
	}).Val()

	var messages []*Message
	for _, pending := range pendingResult {
		// Get the actual message
		msgResult := q.rdb.XRange(ctx, q.name, pending.ID, pending.ID).Val()
		if len(msgResult) > 0 {
			msg, err := q.parseMessage(msgResult[0])
			if err == nil {
				messages = append(messages, msg)
			}
		}
	}

	return messages, nil
}

// Purge removes all messages from the queue
func (q *Queue) Purge() error {
	q.mu.Lock()
	defer q.mu.Unlock()

	ctx := context.Background()
	return q.rdb.Del(ctx, q.name).Err()
}

// GetSize returns the number of messages in the queue
func (q *Queue) GetSize() (int64, error) {
	q.mu.RLock()
	defer q.mu.RUnlock()

	ctx := context.Background()
	return q.rdb.XLen(ctx, q.name).Result()
}
