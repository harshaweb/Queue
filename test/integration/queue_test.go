package integration

import (
	"context"
	"testing"

	"github.com/queue-system/redis-queue/pkg/client"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// setupTestClient creates a test client
func setupTestClient(t *testing.T) *client.Client {
	config := client.DefaultConfig()
	config.RedisConfig.Addresses = []string{"localhost:6379"}

	cl, err := client.NewClient(config)
	require.NoError(t, err)
	return cl
}

// TestBasicQueueOperations tests basic enqueue/dequeue operations
func TestBasicQueueOperations(t *testing.T) {
	testClient := setupTestClient(t)
	defer testClient.Close()

	ctx := context.Background()
	queueName := "test-basic-queue"

	// Enqueue a message
	payload := map[string]interface{}{"message": "hello world"}
	result, err := testClient.Enqueue(ctx, queueName, payload)
	require.NoError(t, err)
	assert.NotEmpty(t, result.MessageID)

	// Dequeue the message
	messages, err := testClient.Dequeue(ctx, queueName, 1)
	require.NoError(t, err)
	require.Len(t, messages, 1)

	message := messages[0]
	assert.Equal(t, result.MessageID, message.ID)
	assert.Equal(t, payload, message.Payload)

	// Acknowledge the message
	err = testClient.Ack(ctx, queueName, message.ID)
	assert.NoError(t, err)
}
