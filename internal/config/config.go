package config

import (
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

	IntegrationArcaneUrl       string
	IntegrationArcaneToken     string
	IntegrationArcaneEnvId     string
	IntegrationArcaneSkipNames []string

	CommitName    string
	CommitEmail   string
	CommitMessage string

	DivergencePolicy string
}

func LoadConfig() Config {
	if err := godotenv.Load(); err != nil {
	}

	logJson := os.Getenv("LOG_JSON")
	if strings.ToLower(logJson) == "true" {
		slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, nil)))
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
		slog.Error("Invalid PORT value, must be an integer", "error", err)
		os.Exit(1)
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

	divergencePolicy := strings.ToLower(strings.TrimSpace(os.Getenv("DIVERGENCE_POLICY")))
	if divergencePolicy == "" {
		divergencePolicy = DivergencePolicyManual
	}
	if divergencePolicy != DivergencePolicyManual && divergencePolicy != DivergencePolicyRebase {
		slog.Warn("Invalid DIVERGENCE_POLICY value, falling back to manual", "value", divergencePolicy)
		divergencePolicy = DivergencePolicyManual
	}

	skipNames := parseCommaSeparatedEnv("INTEGRATION_ARCANE_SKIP_NAMES")

	jobUUID, err := uuid.NewUUID()
	if err != nil {
		slog.Error("Failed to generate job UUID", "error", err)
		os.Exit(1)
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

		AuthType:     authType,
		AuthUser:     os.Getenv("AUTH_USER"),
		AuthPassword: os.Getenv("AUTH_PASSWORD"),

		IntegrationArcaneUrl:       os.Getenv("INTEGRATION_ARCANE_URL"),
		IntegrationArcaneToken:     os.Getenv("INTEGRATION_ARCANE_TOKEN"),
		IntegrationArcaneEnvId:     os.Getenv("INTEGRATION_ARCANE_ENV_ID"),
		IntegrationArcaneSkipNames: skipNames,

		CommitName:    commitName,
		CommitEmail:   commitEmail,
		CommitMessage: commitMessage,

		DivergencePolicy: divergencePolicy,
	}
}

func parseCommaSeparatedEnv(key string) []string {
	raw := os.Getenv(key)
	if strings.TrimSpace(raw) == "" {
		return nil
	}

	parts := strings.Split(raw, ",")
	values := make([]string, 0, len(parts))
	seen := make(map[string]struct{}, len(parts))
	for _, part := range parts {
		name := strings.ToLower(strings.TrimSpace(part))
		if name == "" {
			continue
		}
		if _, exists := seen[name]; exists {
			continue
		}
		seen[name] = struct{}{}
		values = append(values, name)
	}

	if len(values) == 0 {
		return nil
	}

	return values
}
