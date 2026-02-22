package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"govision/worker/internal/services/rabbitmq"
	"govision/worker/internal/services/roboflow"
	"govision/worker/internal/worker"

	"github.com/joho/godotenv"
)

func main() {
	log.SetFlags(log.LstdFlags | log.Lmicroseconds | log.Lshortfile)

	_ = godotenv.Load()
	rabbitConnString := os.Getenv("RABBITMQ_URL")
	rabbitQueueString := os.Getenv("RABBITMQ_QUEUE")
	roboflowAPIKey := os.Getenv("ROBOFLOW_API_KEY")
	roboflowModel := os.Getenv("ROBOFLOW_MODEL")

	if rabbitConnString == "" || rabbitQueueString == "" {
		log.Printf("[ERROR] - Environment variables not found.")
		panic(errors.New("environment variables not found"))
	}

	if roboflowAPIKey == "" || roboflowModel == "" {
		log.Printf("[ERROR] - Roboflow environment variables not found.")
		panic(errors.New("ROBOFLOW_API_KEY and ROBOFLOW_MODEL must be set"))
	}

	// RabbitMQ connection
	rabbitMQConnection, err := rabbitmq.NewRabbittMQConnection(rabbitConnString)
	if err != nil {
		log.Printf("[ERROR] - RabbitMQ connection error: %v", err)
		panic(err)
	}
	defer rabbitMQConnection.Close()

	ch, err := rabbitMQConnection.Channel()
	if err != nil {
		log.Printf("[ERROR] - RabbitMQ channel error: %v", err)
		panic(err)
	}
	defer ch.Close()

	// Consumer
	consumer := rabbitmq.NewRabbitMQConsumer(ch, rabbitQueueString)

	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	msgs, err := consumer.Consume(ctx)
	if err != nil {
		log.Printf("[ERROR] - Failed to start consuming: %v", err)
		panic(err)
	}

	// Roboflow client
	rfClient := roboflow.NewClient(roboflowAPIKey, roboflowModel)

	// Worker
	w := worker.New(rfClient)

	fmt.Println("Successfully connected to RabbitMQ instance")
	fmt.Println("[*] - Waiting for messages")

	w.ProcessMessages(ctx, msgs)
}
