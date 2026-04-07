package integrations

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	neturl "net/url"
	"strings"
	"sync/atomic"

	"golang.org/x/sync/errgroup"
)

type ArcanePaginatedResponse[T any] struct {
	Schema     string                   `json:"$schema,omitempty"`
	Success    bool                     `json:"success"`
	Data       []T                      `json:"data"`
	Pagination ArcaneResponsePagination `json:"pagination"`
}

type ArcaneProject struct {
	ID     string `json:"id"`
	Name   string `json:"name,omitempty"`
	Status string `json:"status,omitempty"`
}

type ArcaneResponsePagination struct {
	CurrentPage     int64 `json:"currentPage"`
	GrandTotalItems int64 `json:"grandTotalItems,omitempty"`
	ItemsPerPage    int64 `json:"itemsPerPage"`
	TotalItems      int64 `json:"totalItems"`
	TotalPages      int64 `json:"totalPages"`
}

func RunArcaneIntegration(url string, token string, envId string, skipProjectNames []string) error {
	slog.Info("Starting redeploy of projects in environment", "environment", envId)

	ctx := context.Background()

	group, groupCtx := errgroup.WithContext(ctx)
	group.SetLimit(8)

	var totalProjects int64
	group.Go(func() error {
		projects, err := fetchArcaneProjects(groupCtx, url, token, envId)
		if err != nil {
			return fmt.Errorf("fetch Arcane projects for environment %q: %w", envId, err)
		}

		redeployGroup, redeployCtx := errgroup.WithContext(groupCtx)
		redeployGroup.SetLimit(16)
		for _, project := range projects {
			project := project
			if shouldSkipProject(project.Name, project.Status, skipProjectNames) {
				continue
			}
			redeployGroup.Go(func() error {
				if err := redeployArcaneProject(redeployCtx, url, token, envId, project.ID); err != nil {
					return fmt.Errorf("redeploy Arcane project %q for environment %q: %w", project.ID, envId, err)
				}
				return nil
			})
		}

		if err := redeployGroup.Wait(); err != nil {
			return err
		}

		atomic.AddInt64(&totalProjects, int64(len(projects)))
		return nil
	})

	if err := group.Wait(); err != nil {
		return err
	}

	slog.Info("Completed redeploy of projects in environment", "environment", envId, "projects", totalProjects)

	return nil
}

func fetchArcaneProjects(ctx context.Context, baseURL string, token string, environmentID string) ([]ArcaneProject, error) {
	endpoint := fmt.Sprintf("/api/environments/%s/projects", neturl.PathEscape(environmentID))

	projects, err := fetchAllArcanePages(ctx, baseURL, token, endpoint, "projects", func(project ArcaneProject) error {
		if project.ID == "" {
			return errors.New("invalid Arcane project payload: missing required field id")
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	return projects, nil
}

func fetchAllArcanePages[T any](ctx context.Context, baseURL string, token string, endpoint string, entityName string, validate func(item T) error) ([]T, error) {
	items := make([]T, 0)

	for page := 1; ; page++ {
		response, err := fetchArcanePage[T](ctx, baseURL, token, endpoint, page, entityName)
		if err != nil {
			return nil, err
		}

		for _, item := range response.Data {
			if err := validate(item); err != nil {
				return nil, err
			}
		}

		items = append(items, response.Data...)

		if response.Pagination.TotalPages <= 0 || int64(page) >= response.Pagination.TotalPages {
			break
		}
	}

	return items, nil
}

func fetchArcanePage[T any](ctx context.Context, baseURL string, token string, endpoint string, page int, entityName string) (ArcanePaginatedResponse[T], error) {
	endpointURL := fmt.Sprintf("%s%s?page=%d", strings.TrimRight(baseURL, "/"), endpoint, page)

	req, err := http.NewRequestWithContext(ctx, "GET", endpointURL, nil)
	if err != nil {
		return ArcanePaginatedResponse[T]{}, fmt.Errorf("create Arcane %s request: %w", entityName, err)
	}

	req.Header.Add("X-Api-Key", token)

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return ArcanePaginatedResponse[T]{}, fmt.Errorf("execute Arcane %s request: %w", entityName, err)
	}
	defer func(body io.ReadCloser) {
		_ = body.Close()
	}(res.Body)

	if res.StatusCode < http.StatusOK || res.StatusCode >= http.StatusMultipleChoices {
		body, _ := io.ReadAll(io.LimitReader(res.Body, 1024))
		return ArcanePaginatedResponse[T]{}, fmt.Errorf("Arcane %s API returned %d: %s", entityName, res.StatusCode, strings.TrimSpace(string(body)))
	}

	var response ArcanePaginatedResponse[T]
	decoder := json.NewDecoder(res.Body)
	if err := decoder.Decode(&response); err != nil {
		return ArcanePaginatedResponse[T]{}, fmt.Errorf("decode Arcane %s response: %w", entityName, err)
	}

	if !response.Success {
		return ArcanePaginatedResponse[T]{}, fmt.Errorf("Arcane %s response indicates failure", entityName)
	}

	return response, nil
}

func redeployArcaneProject(ctx context.Context, baseURL string, token string, environmentID string, projectID string) error {
	endpointURL := fmt.Sprintf("%s/api/environments/%s/projects/%s/redeploy", strings.TrimRight(baseURL, "/"), neturl.PathEscape(environmentID), neturl.PathEscape(projectID))

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpointURL, nil)
	if err != nil {
		return fmt.Errorf("create Arcane redeploy request: %w", err)
	}

	req.Header.Add("X-Api-Key", token)

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("execute Arcane redeploy request: %w", err)
	}
	defer func(body io.ReadCloser) {
		_ = body.Close()
	}(res.Body)

	if res.StatusCode < http.StatusOK || res.StatusCode >= http.StatusMultipleChoices {
		body, _ := io.ReadAll(io.LimitReader(res.Body, 1024))
		return fmt.Errorf("Arcane redeploy API returned %d: %s", res.StatusCode, strings.TrimSpace(string(body)))
	}

	return nil
}

func shouldSkipProject(name string, status string, skipNames []string) bool {
	if status != "running" {
		return true
	}
	if len(skipNames) == 0 {
		return false
	}
	for _, skipName := range skipNames {
		if name == skipName {
			return true
		}
	}
	return false
}
