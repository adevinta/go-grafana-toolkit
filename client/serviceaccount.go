package client

import (
	"context"
	"fmt"
	"net/http"
	"strconv"

	"github.com/grafana/grafana-com-public-clients/go/gcom"
)

// ServiceAccountClient defines all operations related to creating,
// retrieving, and deleting service accounts in Grafana Cloud.
type ServiceAccountClient interface {
	// CreateServiceAccount creates a new service account in the specified Grafana instance
	// with the given name and role.
	CreateServiceAccount(instanceId int, saName string, roleName string) (*ServiceAccount, error)

	// DeleteServiceAccount removes a service account from the specified Grafana instance.
	DeleteServiceAccount(instanceId int, saId int) error
}

// ServiceAccount represents a Grafana service account with its associated
// properties such as ID, name, role, and status.
type ServiceAccount struct {
	Id             int    `json:"id,omitempty"`
	IsDisabled     bool   `json:"isDisabled,omitempty"`
	Name           string `json:"name,omitempty"`
	OrgId          int    `json:"orgId,omitempty"`
	Role           string `json:"role,omitempty"`
	NumberOfTokens int    `json:"tokens,omitempty"`
}

func (c *CloudClient) CreateServiceAccount(instanceId int, saName string, roleName string) (*ServiceAccount, error) {

	saReq := *gcom.NewPostInstanceServiceAccountsRequest(saName, roleName)

	xRequestId := "sa-name-" + saName

	req := c.gComClient.InstancesAPI.PostInstanceServiceAccounts(context.Background(), strconv.Itoa(instanceId)).PostInstanceServiceAccountsRequest(saReq).XRequestId(xRequestId)
	dto, httpResp, err := req.Execute()

	if err != nil {
		return nil, fmt.Errorf("failed to create service account: %w", err)
	}

	if httpResp.StatusCode < http.StatusOK || httpResp.StatusCode >= http.StatusMultipleChoices {
		return nil, fmt.Errorf("unexpected return code %d", httpResp.StatusCode)
	}

	return &ServiceAccount{
		Id:             int(dto.GetId()),
		IsDisabled:     dto.GetIsDisabled(),
		Name:           dto.GetName(),
		OrgId:          int(dto.GetOrgId()),
		Role:           dto.GetRole(),
		NumberOfTokens: int(dto.GetTokens()),
	}, nil
}

func (c *CloudClient) DeleteServiceAccount(instanceId int, saId int) error {

	xRequestId := "sa-id-" + strconv.Itoa(saId)
	httpResp, err := c.gComClient.InstancesAPI.DeleteInstanceServiceAccount(context.Background(), strconv.Itoa(instanceId), strconv.Itoa(saId)).XRequestId(xRequestId).Execute()

	if err != nil {
		return fmt.Errorf("failed to delete service account: %w", err)
	}

	if httpResp.StatusCode < http.StatusOK || httpResp.StatusCode >= http.StatusMultipleChoices {
		return fmt.Errorf("unexpected return code %d", httpResp.StatusCode)
	}

	return nil
}
