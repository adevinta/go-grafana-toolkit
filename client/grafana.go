// Package client provides a Go client for interacting with Grafana Cloud and Grafana HTTP APIs.
// It supports operations for managing service accounts, tokens, organizations, and dashboards.
package client

import (
	"fmt"
	"net/http"
	"net/url"
	"os"
	"time"

	log "github.com/adevinta/go-log-toolkit"
	"github.com/go-openapi/strfmt"
	"github.com/grafana/grafana-com-public-clients/go/gcom"
	"github.com/grafana/grafana-openapi-client-go/client"
)

// GrafanaCloudClient is the main interface for interacting with Grafana Cloud API.
// It provides access to service accounts, tokens, and organization management.
// It also allows creating new stack clients for specific Grafana instances.
type GrafanaCloudClient interface {
	ServiceAccountClient
	TokenClient
	OrganisationClient
	NewStackClient(stack *Stack) (GrafanaStackClient, error)
	NewStackClientWithHttpClient(stack *Stack, httpClient *http.Client) (GrafanaStackClient, error)
}

// CloudClient implements GrafanaCloudClient interface and handles
// communication with the Grafana Cloud API.
type CloudClient struct {
	gComClient *gcom.APIClient
}

// GrafanaStackClient represents a client for a specific Grafana stack instance.
// It provides operations for managing dashboards and cleanup operations.
type GrafanaStackClient interface {
	DashboardClient
	Cleanup() error
	GrafanaStackClient() *client.GrafanaHTTPAPI
}

// StackClient implements GrafanaStackClient interface and handles
// operations for a specific Grafana stack instance.
type StackClient struct {
	httpApi  *client.GrafanaHTTPAPI
	cloudApi GrafanaCloudClient
	sa       *ServiceAccount
	stack    *Stack
}

var timeNow = time.Now

// NewCloudClient creates a new GrafanaCloudClient using the default HTTP client.
// It requires GRAFANA_CLOUD_TOKEN environment variable to be set.
func NewCloudClient() (GrafanaCloudClient, error) {
	return newCloudClient(nil)
}

// NewCloudClientWithHttpClient creates a new GrafanaCloudClient using the provided HTTP client.
// It requires GRAFANA_CLOUD_TOKEN environment variable to be set.
func NewCloudClientWithHttpClient(httpClient *http.Client) (GrafanaCloudClient, error) {
	return newCloudClient(httpClient)
}

func newCloudClient(httpClient *http.Client) (GrafanaCloudClient, error) {
	gcToken, ok := os.LookupEnv("GRAFANA_CLOUD_TOKEN")
	if !ok {
		return nil, fmt.Errorf("GRAFANA_CLOUD_TOKEN not set")
	}
	config := gcom.NewConfiguration()
	config.AddDefaultHeader("Authorization", "Bearer "+gcToken)
	config.Host = "grafana.com"
	config.Scheme = "https"
	config.HTTPClient = httpClient

	return &CloudClient{
		gComClient: gcom.NewAPIClient(config),
	}, nil
}

func (cc *CloudClient) NewStackClientWithHttpClient(stack *Stack, httpClient *http.Client) (GrafanaStackClient, error) {
	return cc.newStackClient(stack, httpClient)
}

func (cc *CloudClient) NewStackClient(stack *Stack) (GrafanaStackClient, error) {
	return cc.newStackClient(stack, nil)
}

func (cc *CloudClient) newStackClient(stack *Stack, httpClient *http.Client) (GrafanaStackClient, error) {
	roleName := "Editor"
	saName := fmt.Sprintf("cpr-dashboard-editor-%s", time.Now().Format("20060102_1504"))
	log.DefaultLogger.WithField("stack", stack.Slug).WithField("saName", saName).Println("creating SA")

	cprSA, err := cc.CreateServiceAccount(stack.StackID, saName, roleName)
	if err != nil {
		return nil, fmt.Errorf("failed to create Stack Client for %s : %w", stack.Slug, err)
	}

	tokenName := "temp-token-" + saName
	log.DefaultLogger.WithField("stack", stack.Slug).WithField("tokenName", tokenName).Println("creating SA token")

	token, err := cc.CreateToken(stack.StackID, cprSA.Id, tokenName)
	if err != nil {
		return nil, fmt.Errorf("failed to create Stack Client for %s : %w", stack.Slug, err)
	}

	u, err := url.Parse(stack.StackURL)
	if err != nil {
		return nil, fmt.Errorf("failed to create Stack Client for %s : %w", stack.Slug, err)
	}

	cfg := &client.TransportConfig{
		Host:     u.Host,
		BasePath: "/api",
		Schemes:  []string{"https"},
		// APIKey is an optional API key or service account token.
		APIKey: token.Key,

		// TLSConfig provides an optional configuration for a TLS client
		// TLSConfig:  &tls.Config{},
		// NumRetries contains the optional number of attempted retries
		NumRetries: 3,
		// RetryTimeout sets an optional time to wait before retrying a request
		RetryTimeout: 0,
		// RetryStatusCodes contains the optional list of status codes to retry
		// Use "x" as a wildcard for a single digit (default: [429, 5xx])
		RetryStatusCodes: []string{"42x", "5xx"},
	}

	if httpClient != nil {
		cfg.Client = httpClient
	}

	return &StackClient{
		httpApi:  client.NewHTTPClientWithConfig(strfmt.Default, cfg),
		cloudApi: cc,
		stack:    stack,
		sa:       cprSA,
	}, nil
}

func (c *StackClient) GrafanaStackClient() *client.GrafanaHTTPAPI {
	return c.httpApi
}

func (c *StackClient) Cleanup() error {
	err := c.cloudApi.DeleteServiceAccount(c.stack.StackID, c.sa.Id)
	if err != nil {
		return fmt.Errorf("failed to delete SA %d in stack %s: %w", c.sa.Id, c.stack.Slug, err)
	}
	return nil
}
