package client

import (
	"context"
	"fmt"
	"net/http"
	"strconv"

	"github.com/grafana/grafana-com-public-clients/go/gcom"
)

// TokenClient defines operations for managing service account tokens
// in Grafana Cloud.
type TokenClient interface {
	// CreateToken creates a new token for a service account in the specified stack.
	CreateToken(stackId int, serviceAccountID int, tokenName string) (*Token, error)
}

// Token represents a Grafana service account token with its
// associated ID, key, and name.
type Token struct {
	Id   int64  `json:"id,omitempty"`
	Key  string `json:"key,omitempty"`
	Name string `json:"name,omitempty"`
}

func (c *CloudClient) CreateToken(stackId int, serviceAccountID int, tokenName string) (*Token, error) {
	var secondsToLive int32 = 500
	resp, httpResp, err := c.gComClient.InstancesAPI.PostInstanceServiceAccountTokens(context.Background(),
		strconv.Itoa(stackId), strconv.Itoa(serviceAccountID)).
		XRequestId(strconv.Itoa(serviceAccountID)).PostInstanceServiceAccountTokensRequest(
		gcom.PostInstanceServiceAccountTokensRequest{
			Name:          tokenName,
			SecondsToLive: &secondsToLive,
		}).Execute()

	if err != nil {
		return nil, fmt.Errorf("failed to create service account token: %w", err)
	}

	if httpResp.StatusCode < http.StatusOK || httpResp.StatusCode >= http.StatusMultipleChoices {
		return nil, fmt.Errorf("unexpected return code %d", httpResp.StatusCode)
	}

	return &Token{
		Id:   resp.GetId(),
		Key:  resp.GetKey(),
		Name: resp.GetName(),
	}, nil
}
