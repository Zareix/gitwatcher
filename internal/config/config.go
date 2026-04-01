package config

import (
	"log"
	"os"
	"strconv"
	"strings"

	"github.com/google/uuid"
	"github.com/joho/godotenv"
)

const (
	AuthTypeNone = "None"
	AuthTypeHTTP = "HTTP"
)

type Config struct {
	WatcherJobCron string
	WatcherJobUUID uuid.UUID
	RepositoryPath string
	Port           int
	AuthType       string
	AuthUser       string
	AuthPassword   string
	CommitName     string
	CommitEmail    string
	CommitMessage  string
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

	cronSchedule, exists := os.LookupEnv("CRON")
	if !exists {
		cronSchedule = "0 */1 * * * *"
	}

	authType, exists := os.LookupEnv("AUTH_TYPE")
	if !exists || authType == "" {
		authType = AuthTypeNone
	}

	commitName, exists := os.LookupEnv("COMMIT_NAME")
	if !exists || commitName == "" {
		commitName = "gitwatcher"
	}

	commitEmail, exists := os.LookupEnv("COMMIT_EMAIL")
	if !exists || commitEmail == "" {
		commitEmail = "gitwatcher@local"
	}

	defaultCommitMessage := "chore: sync changes from gitwatcher"
	commitMessage := strings.TrimSpace(os.Getenv("COMMIT_MESSAGE"))
	if commitMessage == "" {
		commitMessage = defaultCommitMessage
	}

	jobUUID, err := uuid.NewUUID()
	if err != nil {
		log.Fatal(err)
	}
	jobUUIDEnv, exists := os.LookupEnv("JOB_UUID")
	if exists {
		jobUUID = uuid.MustParse(jobUUIDEnv)
	}

	return Config{
		WatcherJobCron: cronSchedule,
		WatcherJobUUID: jobUUID,
		RepositoryPath: repositoryPath,
		Port:           port,
		AuthType:       authType,
		AuthUser:       os.Getenv("AUTH_USER"),
		AuthPassword:   os.Getenv("AUTH_PASSWORD"),
		CommitName:     commitName,
		CommitEmail:    commitEmail,
		CommitMessage:  commitMessage,
	}
}
