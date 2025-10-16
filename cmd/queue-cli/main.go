package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/queue-system/redis-queue/pkg/client"
	"github.com/spf13/cobra"
)

var (
	redisURL     string
	outputFormat string
	verbose      bool
)

func main() {
	rootCmd := &cobra.Command{
		Use:   "queue-cli",
		Short: "Redis Queue System CLI",
		Long:  "A command line interface for the Redis Queue System",
	}

	rootCmd.PersistentFlags().StringVar(&redisURL, "redis-url", "redis://localhost:6379", "Redis connection URL")
	rootCmd.PersistentFlags().StringVar(&outputFormat, "output", "json", "Output format (json, table)")
	rootCmd.PersistentFlags().BoolVar(&verbose, "verbose", false, "Verbose output")

	// Add commands
	rootCmd.AddCommand(createEnqueueCmd())
	rootCmd.AddCommand(createConsumeCmd())
	rootCmd.AddCommand(createStatusCmd())
	rootCmd.AddCommand(createListCmd())
	rootCmd.AddCommand(createPurgeCmd())

	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func createClient() (*client.Client, error) {
	config := client.DefaultConfig()
	config.RedisConfig.Addresses = []string{"localhost:6379"}
	return client.NewClient(config)
}

func createEnqueueCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "enqueue [queue] [payload]",
		Short: "Enqueue a message to a queue",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			queueName := args[0]
			payload := args[1]

			client, err := createClient()
			if err != nil {
				return err
			}
			defer client.Close()

			var payloadData interface{}
			if err := json.Unmarshal([]byte(payload), &payloadData); err != nil {
				// If not JSON, treat as string
				payloadData = payload
			}

			// Use basic enqueue without options for now
			result, err := client.Enqueue(context.Background(), queueName, payloadData)
			if err != nil {
				return fmt.Errorf("failed to enqueue message: %v", err)
			}

			switch outputFormat {
			case "json":
				output, _ := json.MarshalIndent(result, "", "  ")
				fmt.Println(string(output))
			default:
				fmt.Printf("Message enqueued successfully! ID: %s\n", result.MessageID)
			}

			return nil
		},
	}

	return cmd
}

func createConsumeCmd() *cobra.Command {
	var count int

	cmd := &cobra.Command{
		Use:   "consume [queue]",
		Short: "Consume messages from a queue",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			queueName := args[0]

			client, err := createClient()
			if err != nil {
				return err
			}
			defer client.Close()

			fmt.Printf("Consuming messages from queue '%s'...\n", queueName)

			// Use dequeue with batch size
			messages, err := client.Dequeue(context.Background(), queueName, count)
			if err != nil {
				return fmt.Errorf("failed to consume messages: %v", err)
			}

			if len(messages) == 0 {
				fmt.Println("No messages available")
				return nil
			}

			for _, message := range messages {
				switch outputFormat {
				case "json":
					output, _ := json.MarshalIndent(message, "", "  ")
					fmt.Println(string(output))
				default:
					fmt.Printf("Message ID: %s\n", message.ID)
					fmt.Printf("Payload: %v\n", message.Payload)
					fmt.Printf("Created: %s\n", message.CreatedAt.Format(time.RFC3339))
				}

				// Acknowledge the message
				if err := client.Ack(context.Background(), queueName, message.ID); err != nil {
					fmt.Printf("Warning: failed to ack message %s: %v\n", message.ID, err)
				}
			}

			return nil
		},
	}

	cmd.Flags().IntVar(&count, "count", 1, "Number of messages to consume")

	return cmd
}

func createStatusCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "status [queue]",
		Short: "Show queue status",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			queueName := args[0]

			client, err := createClient()
			if err != nil {
				return err
			}
			defer client.Close()

			stats, err := client.GetQueueStats(context.Background(), queueName)
			if err != nil {
				return fmt.Errorf("failed to get queue stats: %v", err)
			}

			switch outputFormat {
			case "json":
				output, _ := json.MarshalIndent(stats, "", "  ")
				fmt.Println(string(output))
			default:
				fmt.Printf("Queue: %s\n", queueName)
				fmt.Printf("Length: %d\n", stats.Length)
				fmt.Printf("In Flight: %d\n", stats.InFlight)
				fmt.Printf("Processed: %d\n", stats.Processed)
				fmt.Printf("Failed: %d\n", stats.Failed)
				fmt.Printf("Dead Letter: %d\n", stats.DeadLetterSize)
			}

			return nil
		},
	}

	return cmd
}

func createListCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List all queues",
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := createClient()
			if err != nil {
				return err
			}
			defer client.Close()

			queues, err := client.ListQueues(context.Background())
			if err != nil {
				return fmt.Errorf("failed to list queues: %v", err)
			}

			switch outputFormat {
			case "json":
				output, _ := json.MarshalIndent(queues, "", "  ")
				fmt.Println(string(output))
			default:
				fmt.Println("Available queues:")
				for _, queue := range queues {
					fmt.Printf("  - %s\n", queue)
				}
			}

			return nil
		},
	}

	return cmd
}

func createPurgeCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "purge [queue]",
		Short: "Purge all messages from a queue (not implemented)",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return fmt.Errorf("purge functionality not yet implemented")
		},
	}

	return cmd
}
