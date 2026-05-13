package client

import (
	"context"
	"crypto/tls"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	authentik "goauthentik.io/api/v3"
)

// Client wraps the generated authentik API client and carries auth context.
// It also retains the base URL and HTTP client so endpoints with quirky
// serialization can be called directly without going through the generated
// JSON parsing.
type Client struct {
	api     *authentik.APIClient
	http    *http.Client
	baseURL string // e.g. "https://authentik.example.com/api/v3"
	token   string
}

func New(baseURL, token string, insecureSkipVerify bool) (*Client, error) {
	u, err := url.Parse(baseURL)
	if err != nil {
		return nil, fmt.Errorf("parse url: %w", err)
	}
	if u.Scheme == "" || u.Host == "" {
		return nil, fmt.Errorf("AUTHENTIK_URL must be absolute (got %q)", baseURL)
	}

	cfg := authentik.NewConfiguration()
	cfg.Host = u.Host
	cfg.Scheme = u.Scheme
	cfg.UserAgent = "authentik-exporter"
	cfg.Servers = authentik.ServerConfigurations{{URL: "/api/v3"}}

	httpClient := &http.Client{}
	if insecureSkipVerify {
		httpClient.Transport = &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		}
	}
	cfg.HTTPClient = httpClient

	apiBase := strings.TrimRight(baseURL, "/") + "/api/v3"
	return &Client{
		api:     authentik.NewAPIClient(cfg),
		http:    httpClient,
		baseURL: apiBase,
		token:   token,
	}, nil
}

// ctx returns a context with the bearer token attached for the authentik client.
func (c *Client) ctx(parent context.Context) context.Context {
	return context.WithValue(parent, authentik.ContextAccessToken, c.token)
}
