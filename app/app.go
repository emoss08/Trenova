package main

import (
	"log"

	"github.com/confluentinc/confluent-kafka-go/v2/kafka"
	"github.com/emoss08/trenova/database"
	_ "github.com/emoss08/trenova/ent/runtime"
	"github.com/emoss08/trenova/server"
	tools "github.com/emoss08/trenova/util"
	trenova_kafka "github.com/emoss08/trenova/util/kafka"
	"github.com/emoss08/trenova/util/logger"
	"github.com/emoss08/trenova/util/minio"
	"github.com/emoss08/trenova/util/redis"
	"github.com/emoss08/trenova/util/session"

	_ "github.com/lib/pq"
)

func main() {
	customLogger := logger.NewLogger()
	logger.SetLogger(customLogger)

	// Initialize the database
	client := database.NewEntClient(
		tools.GetEnv("SERVER_DB_DSN", "host=localhost port=5432 user=postgres password=postgres dbname=trenova_go_db sslmode=disable"),
	)

	defer client.Close()

	// Set the client to variable defined in the database package
	// This will enable the client instance to be accessed anywhere through the accessor
	// named GetClient
	database.SetClient(client)

	// Setup Session Store
	sessionStore := session.New(client, []byte(
		tools.GetEnv("SEVER_SESSION_KEY", "trenova-session-key"),
	))

	// If the session store is not set, panic
	if sessionStore == nil {
		panic("Failed to create session store. Exiting...")
	}

	session.SetStore(sessionStore)

	// Initialize the redis client
	redisClient := redis.NewRedisClient(
		tools.GetEnv("SERVER_REDIS_HOST", "localhost:6379"),
	)

	// Set the redis client to variable defined in the redis package
	// This will enable the client instance to be accessed anywhere through the accessor
	// named GetClient
	redis.SetClient(redisClient)

	// Initialize minio client
	minioClient, err := minio.NewMinioClient(
		tools.GetEnv("SERVER_MINIO_ENDPOINT", "localhost:9000"),
		tools.GetEnv("SERVER_MINIO_ACCESS_KEY_ID", "minio"),
		tools.GetEnv("SERVER_MINIO_SECRET_ACCESS_KEY", "minio"),
		false,
	)
	if err != nil {
		log.Panicf("Failed to connect to minio: %v", err)
	}

	// Set the minio client to variable defined in the minio package
	// This will enable the client instance to be accessed anywhere through the accessor
	// named GetClient
	minio.SetClient(minioClient)

	// Initialize the Kafka Client Configuration
	kafkaConfig := &kafka.ConfigMap{
		"bootstrap.servers": tools.GetEnv("SERVER_KAFKA_BROKER", "localhost:9092"),
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
	if err != nil {
		log.Panicf("Failed to create media bucket: %v", err)
	}

	// Setup server
	server.SetupAndRun()
}
