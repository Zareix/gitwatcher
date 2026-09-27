package config

import (
	"fmt"
	"log/slog"
	"os"
	"strconv"
	"strings"

	"github.com/google/uuid"
	"github.com/joho/godotenv"
)

const (
	AuthTypeNone = "None"
	AuthTypeHTTP = "HTTP"

	DivergencePolicyManual = "manual"
	DivergencePolicyRebase = "rebase"
)

type Config struct {
	WatcherJobCron string
	WatcherJobUUID uuid.UUID
	RepositoryPath string
	Port           int

	AuthType     string
	AuthUser     string
	AuthPassword string

	IntegrationWebhookUrl   string
	IntegrationWebhookToken string

	CommitName    string
	CommitEmail   string
	CommitMessage string

	DivergencePolicy string
}

func LoadConfig() (Config, error) {
	_ = godotenv.Load()

	if strings.ToLower(os.Getenv("LOG_JSON")) == "true" {
		slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, nil)))
	}

	repositoryPath := os.Getenv("REPOSITORY_PATH")
	if repositoryPath == "" {
		repositoryPath = "./output"
	}

	port := 8080
	if portEnv := os.Getenv("PORT"); portEnv != "" {
		parsed, err := strconv.Atoi(portEnv)
		if err != nil {
			return Config{}, fmt.Errorf("invalid PORT value %q, must be an integer: %w", portEnv, err)
		}
		port = parsed
	}

	cronSchedule := os.Getenv("CRON")
	if cronSchedule == "" {
		cronSchedule = "0 */1 * * * *"
	}

	authType := os.Getenv("AUTH_TYPE")
	if authType == "" {
		authType = AuthTypeNone
	}

	commitName := os.Getenv("COMMIT_NAME")
	if commitName == "" {
		commitName = "gitwatcher"
	}

	commitEmail := os.Getenv("COMMIT_EMAIL")
	if commitEmail == "" {
		commitEmail = "gitwatcher@local"
	}

	commitMessage := strings.TrimSpace(os.Getenv("COMMIT_MESSAGE"))
	if commitMessage == "" {
		commitMessage = "chore: sync changes from gitwatcher"
	}

	divergencePolicy := strings.ToLower(strings.TrimSpace(os.Getenv("DIVERGENCE_POLICY")))
	switch divergencePolicy {
	case "":
		divergencePolicy = DivergencePolicyManual
	case DivergencePolicyManual, DivergencePolicyRebase:
	default:
		return Config{}, fmt.Errorf("invalid DIVERGENCE_POLICY %q, expected %q or %q", divergencePolicy, DivergencePolicyManual, DivergencePolicyRebase)
	}

	jobUUID, err := uuid.NewRandom()
	if err != nil {
		return Config{}, fmt.Errorf("generate job UUID: %w", err)
	}
	if jobUUIDEnv := os.Getenv("JOB_UUID"); jobUUIDEnv != "" {
		parsed, err := uuid.Parse(jobUUIDEnv)
		if err != nil {
			return Config{}, fmt.Errorf("invalid JOB_UUID value %q: %w", jobUUIDEnv, err)
		}
		jobUUID = parsed
	}

	return Config{
		WatcherJobCron: cronSchedule,
		WatcherJobUUID: jobUUID,
		RepositoryPath: repositoryPath,
		Port:           port,

		AuthType:     authType,
		AuthUser:     os.Getenv("AUTH_USER"),
		AuthPassword: os.Getenv("AUTH_PASSWORD"),

		IntegrationWebhookUrl:   os.Getenv("INTEGRATION_WEBHOOK_URL"),
		IntegrationWebhookToken: os.Getenv("INTEGRATION_WEBHOOK_TOKEN"),

		CommitName:    commitName,
		CommitEmail:   commitEmail,
		CommitMessage: commitMessage,

		DivergencePolicy: divergencePolicy,
	}, nil
}
