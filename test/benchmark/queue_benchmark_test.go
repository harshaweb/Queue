package benchmark

import (
	"context"
	"fmt"
	"testing"

	"github.com/harshaweb/Queue/pkg/client"
	"github.com/stretchr/testify/require"
)

// setupBenchmarkClient creates a client for benchmarking
func setupBenchmarkClient(b *testing.B) *client.Client {
	config := client.DefaultConfig()
	config.RedisConfig.Addresses = []string{"localhost:6379"}

	cl, err := client.NewClient(config)
	require.NoError(b, err)
	return cl
}

// BenchmarkEnqueue benchmarks enqueue operations
func BenchmarkEnqueue(b *testing.B) {
	testClient := setupBenchmarkClient(b)
	defer testClient.Close()

	ctx := context.Background()
	queueName := "benchmark-enqueue"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		payload := map[string]interface{}{
			"message": fmt.Sprintf("benchmark message %d", i),
			"id":      i,
		}
		_, err := testClient.Enqueue(ctx, queueName, payload)
		if err != nil {
			b.Fatal(err)
		}
	}
}
