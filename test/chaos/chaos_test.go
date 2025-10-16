package chaos

import (
	"context"
	"sync"
	"testing"

	"github.com/harshaweb/Queue/pkg/client"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// setupChaosClient creates a client for chaos testing
func setupChaosClient(t *testing.T) *client.Client {
	config := client.DefaultConfig()
	config.RedisConfig.Addresses = []string{"localhost:6379"}

	cl, err := client.NewClient(config)
	require.NoError(t, err)
	return cl
}

// TestConcurrentEnqueue tests concurrent enqueue operations
func TestConcurrentEnqueue(t *testing.T) {
	testClient := setupChaosClient(t)
	defer testClient.Close()

	ctx := context.Background()
	queueName := "chaos-concurrent-enqueue"
	numGoroutines := 10
	messagesPerGoroutine := 10

	var wg sync.WaitGroup
	wg.Add(numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		go func(workerID int) {
			defer wg.Done()
			for j := 0; j < messagesPerGoroutine; j++ {
				payload := map[string]interface{}{
					"worker": workerID,
					"msg":    j,
				}
				_, err := testClient.Enqueue(ctx, queueName, payload)
				assert.NoError(t, err)
			}
		}(i)
	}

	wg.Wait()

	// Verify messages were enqueued
	stats, err := testClient.GetQueueStats(ctx, queueName)
	require.NoError(t, err)
	assert.Greater(t, stats.Length, int64(0))
}
