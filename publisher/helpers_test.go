package publisher

import (
	"fmt"
	"net/http"

	grafana "github.com/adevinta/go-grafana-toolkit/client"
	"github.com/stretchr/testify/mock"
)

type MockStackClient struct {
	mock.Mock
}

func (m *MockStackClient) UploadDashboard(dashboard *grafana.Dashboard) error {
	args := m.Called(dashboard)
	return args.Error(0)
}

func (m *MockStackClient) GetDashboard(uid string) (*grafana.Dashboard, error) {
	args := m.Called(uid)
	return args.Get(0).(*grafana.Dashboard), args.Error(1)
}

func (m *MockStackClient) DeleteDashboard(uid string) error {
	args := m.Called(uid)
	return args.Error(0)
}

func (m *MockStackClient) EnsureFolder(folder string) (*grafana.Folder, error) {
	args := m.Called(folder)
	return args.Get(0).(*grafana.Folder), args.Error(1)
}

func (m *MockStackClient) GetDataSource(name string) (*grafana.Datasource, error) {
	args := m.Called(name)
	return args.Get(0).(*grafana.Datasource), args.Error(1)
}

func (m *MockStackClient) Cleanup() error {
	args := m.Called()
	return args.Error(0)
}

type MockCloudClient struct {
	mock.Mock
}

func (m *MockCloudClient) ListStacks() (grafana.Stacks, error) {
	args := m.Called()
	return args.Get(0).(grafana.Stacks), args.Error(1)
}

func (m *MockCloudClient) CreateServiceAccount(id int, name string, role string) (*grafana.ServiceAccount, error) {
	args := m.Called(id, name, role)
	fmt.Println("called CreateServiceAccount: ", id, name, role)
	return args.Get(0).(*grafana.ServiceAccount), args.Error(1)
}

func (m *MockCloudClient) GetStack(slug string) (*grafana.Stack, error) {
	args := m.Called(slug)
	fmt.Println("called GetStack: ", slug)
	return args.Get(0).(*grafana.Stack), args.Error(1)
}

func (m *MockCloudClient) CreateToken(stackID int, tokenID int, role string) (*grafana.Token, error) {
	args := m.Called(stackID, tokenID, role)
	fmt.Println("called CreateToken: ", stackID, tokenID, role)
	return args.Get(0).(*grafana.Token), args.Error(1)
}

func (m *MockCloudClient) DeleteServiceAccount(id int, accountID int) error {
	args := m.Called(id, accountID)
	fmt.Println("called DeleteServiceAccount: ", id, accountID)
	return args.Error(0)
}

func (m *MockCloudClient) NewStackClient(stack *grafana.Stack) (grafana.GrafanaStackClient, error) {
	args := m.Called(stack)
	fmt.Println("called NewStackClient: ", stack)
	return args.Get(0).(grafana.GrafanaStackClient), args.Error(1)
}

func (m *MockCloudClient) NewStackClientWithHttpClient(stack *grafana.Stack, httpClient *http.Client) (grafana.GrafanaStackClient, error) {
	args := m.Called(stack)
	fmt.Println("called NewStackClientWithHttpClient: ", stack)
	return args.Get(0).(grafana.GrafanaStackClient), args.Error(1)
}
