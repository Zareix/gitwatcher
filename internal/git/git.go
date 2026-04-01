package git

import (
	"fmt"
	"gitwatcher/internal/config"
	"log"
	"strings"

	"github.com/go-git/go-git/v6"
	"github.com/go-git/go-git/v6/plumbing/transport"
	githttp "github.com/go-git/go-git/v6/plumbing/transport/http"
)

func CloseRepo(repo *git.Repository) {
	if repo == nil {
		return
	}
	if err := repo.Close(); err != nil {
		log.Println("close repository:", err)
	}
}

func BuildAuthMethod(cfg config.Config) (transport.AuthMethod, error) {
	switch strings.ToLower(cfg.AuthType) {
	case "", strings.ToLower(config.AuthTypeNone):
		return nil, nil
	case strings.ToLower(config.AuthTypeHTTP):
		if cfg.AuthUser == "" || cfg.AuthPassword == "" {
			return nil, fmt.Errorf("AUTH_TYPE=HTTP requires AUTH_USER and AUTH_PASSWORD")
		}
		return &githttp.BasicAuth{Username: cfg.AuthUser, Password: cfg.AuthPassword}, nil
	default:
		return nil, fmt.Errorf("unsupported AUTH_TYPE %q, expected %q or %q", cfg.AuthType, config.AuthTypeNone, config.AuthTypeHTTP)
	}
}
