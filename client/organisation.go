package client

import (
	"context"
	"fmt"
	"net/http"
)

// OrganisationClient defines operations for managing Grafana Cloud organizations
// and retrieving stack information.
type OrganisationClient interface {
	GetStack(slug string) (*Stack, error)
	ListStacks() (Stacks, error)
}

// Stack contains all the relevant details of a GrafanaCloud stack including
// instance IDs, URLs, and identification information.
type Stack struct {
	LogsInstanceID    int    `json:"hlInstanceId"`
	MetricsInstanceID int    `json:"hmInstancePromId"`
	PromURL           string `json:"hmInstancePromUrl"`
	LogsURL           string `json:"hlInstanceUrl"`
	StackID           int    `json:"id"`
	Slug              string `json:"slug" yaml:"slug"`
	StackURL          string `json:"url" yaml:"url"`
}

// Stacks represents a collection of Stack objects
type Stacks []Stack

// GetStack returns a stack definition for the corresponding GrafanaCloud stack
// identified by its slug. Returns an error if the stack cannot be found or
// if the API request fails.
func (c *CloudClient) GetStack(slug string) (*Stack, error) {
	resp, httpResp, err := c.gComClient.InstancesAPI.GetInstances(context.Background()).Slug(slug).Execute()
	if err != nil {
		return nil, fmt.Errorf("failed to get stack %s: %w", slug, err)
	}

	if httpResp.StatusCode < http.StatusOK || httpResp.StatusCode >= http.StatusMultipleChoices {
		return nil, fmt.Errorf("unexpected return code %d", httpResp.StatusCode)
	}

	if len(resp.GetItems()) == 0 {
		err := fmt.Errorf("stack not found: %s", slug)
		return nil, err
	}

	stack := resp.GetItems()[0]
	s := &Stack{
		LogsInstanceID:    int(stack.HlInstanceId),
		MetricsInstanceID: int(stack.HmInstancePromId),
		PromURL:           stack.HmInstancePromUrl,
		LogsURL:           stack.HlInstanceUrl,
		StackID:           int(stack.Id),
		Slug:              stack.Slug,
		StackURL:          stack.Url,
	}

	return s, nil
}

// ListStacks retrieves all available stacks from GrafanaCloud.
// Returns a collection of Stack objects or an error if the API request fails.
func (c *CloudClient) ListStacks() (Stacks, error) {
	resp, httpResp, err := c.gComClient.InstancesAPI.GetInstances(context.Background()).Execute()
	if err != nil {
		return nil, fmt.Errorf("failed to get stacks: %w", err)
	}

	if httpResp.StatusCode < http.StatusOK || httpResp.StatusCode >= http.StatusMultipleChoices {
		return nil, fmt.Errorf("unexpected return code %d", httpResp.StatusCode)
	}

	stacks := []Stack{}
	for _, stack := range resp.Items {
		stacks = append(
			stacks,
			Stack{
				LogsInstanceID:    int(stack.HlInstanceId),
				MetricsInstanceID: int(stack.HmInstancePromId),
				PromURL:           stack.HmInstancePromUrl,
				LogsURL:           stack.HlInstanceUrl,
				StackID:           int(stack.Id),
				Slug:              stack.Slug,
				StackURL:          stack.Url,
			},
		)
	}

	return stacks, nil
}
