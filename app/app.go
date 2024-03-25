package main

import (
	"log"
	"os"

	"github.com/confluentinc/confluent-kafka-go/v2/kafka"
	"github.com/emoss08/trenova/database"
	_ "github.com/emoss08/trenova/ent/runtime"
	"github.com/emoss08/trenova/server"
	trenova_kafka "github.com/emoss08/trenova/tools/kafka"
	"github.com/emoss08/trenova/tools/minio"
	"github.com/emoss08/trenova/tools/redis"
	"github.com/emoss08/trenova/tools/session"

	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
)

func main() {
	err := godotenv.Load()
	if err != nil {
		panic("Error loading .env file")
	}

	// Initialize the database
	client, err := database.NewEntClient(os.Getenv("DB_DSN"))
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}

	defer client.Close()

	// Set the client to variable defined in the database package
	// This will enable the client instance to be accessed anywhere through the accessor
	// named GetClient
	database.SetClient(client)

	// Setup Session Store
	sessionStore := session.New(client, []byte(os.Getenv("SESSION_KEY")))

	// If the session store is not set, panic
	if sessionStore == nil {
		panic("Failed to create session store. Exiting...")
	}

	session.SetStore(sessionStore)

	// Initialize the redis client
	redisClient := redis.NewRedisClient(os.Getenv("REDIS_ADDR"))

	// Set the redis client to variable defined in the redis package
	// This will enable the client instance to be accessed anywhere through the accessor
	// named GetClient
	redis.SetClient(redisClient)

	// Initialize minio client
	minioClient, err := minio.NewMinioClient("localhost:9000", "s11TWrbGf10TuA46rRTN", "C6Y039j3sseNwb8rEAbBfMwZxhRxgfD6ACADHEIC", false)
	if err != nil {
		log.Panicf("Failed to connect to minio: %v", err)
	}

	// Set the minio client to variable defined in the minio package
	// This will enable the client instance to be accessed anywhere through the accessor
	// named GetClient
	minio.SetClient(minioClient)

	// Initialize the Kafka Client Configuration
	kafkaConfig := &kafka.ConfigMap{
		"bootstrap.servers": os.Getenv("KAFKA_BROKER"),
	}

	// Initialize the Kafka Admin Client
	kafkaAdminClient, err := trenova_kafka.NewKafkaClient(kafkaConfig)
	if err != nil {
		log.Panicf("Failed to connect to kafka: %v", err)
	}
	defer trenova_kafka.CloseKafkaClient()

	// Set the kafka admin client to variable defined in the kafka package
	// This will enable the client instance to be accessed anywhere through the accessor
	// named GetKafkaClient
	trenova_kafka.SetKafkaClient(kafkaAdminClient)

	// Create media bucket
	err = minio.CreateMediaBucket("trenova-media", "us-east-1")

	// Setup server
	server.SetupAndRun()
}
