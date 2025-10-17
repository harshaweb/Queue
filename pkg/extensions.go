package pkg

import (
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
)

// Retry reprocesses a failed message
func (q *Queue) Retry(messageID string) error {
	q.mu.RLock()
	defer q.mu.RUnlock()

	// If we have a DLQ, try to retry from there
	if q.dlq != nil {
		return q.dlq.Retry(messageID, q)
	}

	return fmt.Errorf("no dead letter queue configured")
}

// SkipToDeadline moves messages past their deadline to DLQ
func (q *Queue) SkipToDeadline() error {
	q.mu.RLock()
	defer q.mu.RUnlock()

	if !q.config.EnableDLQ {
		return fmt.Errorf("dead letter queue not enabled")
	}

	now := time.Now()

	// Get pending messages
	pendingMessages, err := q.GetPendingMessages()
	if err != nil {
		return err
	}

	var skippedCount int
	for _, msg := range pendingMessages {
		if msg.Deadline != nil && now.After(*msg.Deadline) {
			// Move to DLQ
			if err := q.dlq.Send(msg, "deadline_exceeded_manual_skip"); err == nil {
				// Acknowledge the original message
				q.Ack(msg.StreamID)
				skippedCount++
			}
		}
	}

	if q.metrics != nil {
		q.metrics.IncrementErrors(fmt.Sprintf("deadline_skipped_%d", skippedCount))
	}

	return nil
}

// Scheduler provides advanced message scheduling capabilities
type Scheduler struct {
	queue     *Queue
	scheduled map[string]*ScheduledMessage
	mu        sync.RWMutex
	ticker    *time.Ticker
	stopCh    chan struct{}
	isRunning bool
}

// ScheduledMessage represents a message scheduled for future processing
type ScheduledMessage struct {
	Message    *Message      `json:"message"`
	ScheduleID string        `json:"schedule_id"`
	ExecuteAt  time.Time     `json:"execute_at"`
	Interval   time.Duration `json:"interval,omitempty"` // For recurring messages
	MaxRuns    int           `json:"max_runs,omitempty"` // Limit for recurring messages
	RunCount   int           `json:"run_count"`
}

// NewScheduler creates a new message scheduler
func (q *Queue) NewScheduler() *Scheduler {
	return &Scheduler{
		queue:     q,
		scheduled: make(map[string]*ScheduledMessage),
		stopCh:    make(chan struct{}),
	}
}

// ScheduleMessage schedules a message for future execution
func (s *Scheduler) ScheduleMessage(msg *Message, executeAt time.Time) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	scheduleID := uuid.New().String()
	scheduled := &ScheduledMessage{
		Message:    msg,
		ScheduleID: scheduleID,
		ExecuteAt:  executeAt,
	}

	s.scheduled[scheduleID] = scheduled
	return scheduleID, nil
}

// ScheduleRecurring schedules a recurring message
func (s *Scheduler) ScheduleRecurring(msg *Message, interval time.Duration, maxRuns int) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	scheduleID := uuid.New().String()
	scheduled := &ScheduledMessage{
		Message:    msg,
		ScheduleID: scheduleID,
		ExecuteAt:  time.Now().Add(interval),
		Interval:   interval,
		MaxRuns:    maxRuns,
	}

	s.scheduled[scheduleID] = scheduled
	return scheduleID, nil
}

// CancelSchedule cancels a scheduled message
func (s *Scheduler) CancelSchedule(scheduleID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.scheduled[scheduleID]; !exists {
		return fmt.Errorf("schedule %s not found", scheduleID)
	}

	delete(s.scheduled, scheduleID)
	return nil
}

// Start begins the scheduler
func (s *Scheduler) Start() error {
	if s.isRunning {
		return fmt.Errorf("scheduler is already running")
	}

	s.isRunning = true
	s.ticker = time.NewTicker(time.Second)

	go func() {
		defer func() {
			s.isRunning = false
		}()

		for {
			select {
			case <-s.stopCh:
				s.ticker.Stop()
				return
			case <-s.ticker.C:
				s.processScheduled()
			}
		}
	}()

	return nil
}

// Stop stops the scheduler
func (s *Scheduler) Stop() error {
	if !s.isRunning {
		return fmt.Errorf("scheduler is not running")
	}

	close(s.stopCh)
	return nil
}

// processScheduled processes ready scheduled messages
func (s *Scheduler) processScheduled() {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now()
	var toRemove []string

	for scheduleID, scheduled := range s.scheduled {
		if now.After(scheduled.ExecuteAt) || now.Equal(scheduled.ExecuteAt) {
			// Send the message
			_, err := s.queue.SendMessage(scheduled.Message)
			if err != nil {
				// Log error but continue
				continue
			}

			scheduled.RunCount++

			// Handle recurring messages
			if scheduled.Interval > 0 {
				if scheduled.MaxRuns == 0 || scheduled.RunCount < scheduled.MaxRuns {
					// Schedule next execution
					scheduled.ExecuteAt = scheduled.ExecuteAt.Add(scheduled.Interval)
				} else {
					// Max runs reached, remove from schedule
					toRemove = append(toRemove, scheduleID)
				}
			} else {
				// One-time message, remove from schedule
				toRemove = append(toRemove, scheduleID)
			}
		}
	}

	// Remove completed schedules
	for _, scheduleID := range toRemove {
		delete(s.scheduled, scheduleID)
	}
}

// GetScheduled returns all scheduled messages
func (s *Scheduler) GetScheduled() map[string]*ScheduledMessage {
	s.mu.RLock()
	defer s.mu.RUnlock()

	result := make(map[string]*ScheduledMessage)
	for k, v := range s.scheduled {
		result[k] = v
	}
	return result
}
