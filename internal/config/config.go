package config

import (
	"log"
	"os"
	"strconv"

	"github.com/google/uuid"
	"github.com/joho/godotenv"
)

const (
	AuthTypeNone = "None"
	AuthTypeHTTP = "HTTP"
)

type Config struct {
	PullerJobCron  string
	PullerJobUUID  uuid.UUID
	RepositoryPath string
	Port           int
	AuthType       string
	AuthUser       string
	AuthPassword   string
}

func LoadConfig() Config {
	if err := godotenv.Load(); err != nil {
		log.Println("Could not load .env file, proceeding with environment variables")
	}

	repositoryPath, exists := os.LookupEnv("REPOSITORY_PATH")
	if !exists {
		repositoryPath = "./output"
	}

	portEnv := os.Getenv("PORT")
	if portEnv == "" {
		portEnv = "8080"
	}
	port, err := strconv.Atoi(portEnv)
	if err != nil {
		log.Fatalf("Invalid PORT value: %v", err)
	}

	pullerJobCronSchedule, exists := os.LookupEnv("PULLER_JOB_CRON")
	if !exists {
		pullerJobCronSchedule = "* */5 * * * *"
	}

	authType, exists := os.LookupEnv("AUTH_TYPE")
	if !exists || authType == "" {
		authType = AuthTypeNone
	}

	pullerJobUUID, err := uuid.NewUUID()
	if err != nil {
		log.Fatal(err)
	}
	pullerJobUUIDEnv, exists := os.LookupEnv("PULLER_JOB_UUID")
	if exists {
		pullerJobUUID = uuid.MustParse(pullerJobUUIDEnv)
	}

	return Config{
		PullerJobCron:  pullerJobCronSchedule,
		PullerJobUUID:  pullerJobUUID,
		RepositoryPath: repositoryPath,
		Port:           port,
		AuthType:       authType,
		AuthUser:       os.Getenv("AUTH_USER"),
		AuthPassword:   os.Getenv("AUTH_PASSWORD"),
	}
}
