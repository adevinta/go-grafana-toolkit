package client

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"testing"

	testutils "github.com/adevinta/go-testutils-toolkit"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var testStack *Stack = &Stack{
	StackID:  1234,
	Slug:     "test-stack",
	StackURL: "https://test-stack.grafana.net",
}

func TestClientGetStacks(t *testing.T) {
	os.Setenv("GRAFANA_CLOUD_TOKEN", "fake-token")
	defer os.Unsetenv("GRAFANA_CLOUD_TOKEN")

	t.Run("return two existing stacks", func(t *testing.T) {
		client, err := NewCloudClientWithHttpClient(&http.Client{
			Transport: testutils.RoundTripperFunc(func(req *http.Request) (*http.Response, error) {
				assert.Equal(t, "GET", req.Method)
				assert.Equal(t, "https://grafana.com/api/instances", req.URL.String())
				return testutils.NewHTTPResponseBuilder().
					WithJsonBody(map[string]interface{}{
						"items": []map[string]interface{}{
							{
								"id":                111,
								"slug":              "stack-1",
								"url":               "https://stack1.grafana.net",
								"hlInstanceId":      222,
								"hlInstanceUrl":     "https://logs1.grafana.net",
								"hmInstancePromId":  333,
								"hmInstancePromUrl": "https://prom1.grafana.net",
							},
							{
								"id":                444,
								"slug":              "stack-2",
								"url":               "https://stack2.grafana.net",
								"hlInstanceId":      555,
								"hlInstanceUrl":     "https://logs2.grafana.net",
								"hmInstancePromId":  666,
								"hmInstancePromUrl": "https://prom2.grafana.net",
							},
						},
					}).
					WithStatusCode(http.StatusOK).Build(), nil
			}),
		})

		assert.NoError(t, err)

		stacks, err := client.ListStacks()
		assert.NoError(t, err)
		assert.Len(t, stacks, 2)

		assert.Equal(t, 111, stacks[0].StackID)
		assert.Equal(t, "stack-1", stacks[0].Slug)
		assert.Equal(t, 444, stacks[1].StackID)
		assert.Equal(t, "stack-2", stacks[1].Slug)
	})
}

func TestClientGetStack(t *testing.T) {
	os.Setenv("GRAFANA_CLOUD_TOKEN", "fake-token")
	defer os.Unsetenv("GRAFANA_CLOUD_TOKEN")

	t.Run("return an existing stack", func(t *testing.T) {
		client, err := NewCloudClientWithHttpClient(&http.Client{
			Transport: testutils.RoundTripperFunc(func(req *http.Request) (*http.Response, error) {
				assert.Equal(t, "GET", req.Method)
				assert.Equal(t, "https://grafana.com/api/instances?slug=test-stack", req.URL.String())
				return testutils.NewHTTPResponseBuilder().
					WithJsonBody(map[string]interface{}{
						"items": []map[string]interface{}{
							{
								"id":                1234,
								"slug":              "test-stack",
								"url":               "https://my-stack.grafana.net",
								"hlInstanceId":      2345,
								"hlInstanceUrl":     "https://logs.grafana.net",
								"hmInstancePromId":  3456,
								"hmInstancePromUrl": "https://prom.grafana.net",
							},
						},
					}).
					WithStatusCode(http.StatusOK).Build(), nil
			}),
		})

		assert.NoError(t, err)

		stack, err := client.GetStack("test-stack")
		assert.NoError(t, err)
		assert.NotNil(t, stack)
		assert.Equal(t, 1234, stack.StackID)
		assert.Equal(t, 2345, stack.LogsInstanceID)
		assert.Equal(t, 3456, stack.MetricsInstanceID)
		assert.Equal(t, "test-stack", stack.Slug)
		assert.Equal(t, "https://my-stack.grafana.net", stack.StackURL)
		assert.Equal(t, "https://logs.grafana.net", stack.LogsURL)
		assert.Equal(t, "https://prom.grafana.net", stack.PromURL)
	})

	t.Run("return no available stack", func(t *testing.T) {
		client, err := NewCloudClientWithHttpClient(&http.Client{
			Transport: testutils.RoundTripperFunc(func(req *http.Request) (*http.Response, error) {
				assert.Equal(t, "GET", req.Method)
				assert.Equal(t, "https://grafana.com/api/instances?slug=test-stack", req.URL.String())
				return testutils.NewHTTPResponseBuilder().
					WithJsonBody(map[string]interface{}{
						"items": []map[string]interface{}{},
					}).
					WithStatusCode(http.StatusOK).Build(), nil
			}),
		})

		assert.NoError(t, err)

		stack, err := client.GetStack("test-stack")
		assert.Nil(t, stack)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "stack not found")
	})
}

func buildCloudClient(t *testing.T) (GrafanaCloudClient, error) {
	return NewCloudClientWithHttpClient(&http.Client{
		Transport: testutils.RoundTripperFunc(func(req *http.Request) (*http.Response, error) {
			assert.Equal(t, "POST", req.Method)
			assert.Equal(t, "application/json", req.Header.Get("Content-Type"))
			require.NotNil(t, req.Body)
			var payload map[string]interface{}
			if err := json.NewDecoder(req.Body).Decode(&payload); err != nil {
				t.Errorf("failed to decode request body: %v", err)
				return nil, fmt.Errorf("failed to decode request body: %w", err)
			}

			switch req.URL.String() {
			case "https://grafana.com/api/instances/1234/api/serviceaccounts":
				assert.Contains(t, payload, "name")
				assert.Contains(t, payload, "role")
				assert.Equal(t, "Editor", payload["role"])
				assert.NotEmpty(t, payload["name"])
				return testutils.NewHTTPResponseBuilder().
					WithJsonBody(map[string]interface{}{
						"id":   5678,
						"name": payload["name"],
						"role": "Editor",
					}).
					WithStatusCode(http.StatusOK).Build(), nil

			case "https://grafana.com/api/instances/1234/api/serviceaccounts/5678/tokens":
				assert.Contains(t, payload, "name")
				assert.Contains(t, payload, "secondsToLive")
				assert.NotEmpty(t, payload["name"])
				assert.NotEmpty(t, payload["secondsToLive"])
				return testutils.NewHTTPResponseBuilder().
					WithJsonBody(map[string]interface{}{
						"id":   9012,
						"key":  "fake-token-key",
						"name": "temp-token-cpr-dashboard-editor-20230101_0000",
					}).
					WithStatusCode(http.StatusOK).Build(), nil
			default:
				t.Errorf("unexpected request: %s", req.URL.String())
				return nil, fmt.Errorf("unexpected request: %s", req.URL.String())
			}
		}),
	})
}

func TestNewStackClient(t *testing.T) {
	os.Setenv("GRAFANA_CLOUD_TOKEN", "fake-token")
	defer os.Unsetenv("GRAFANA_CLOUD_TOKEN")

	t.Run("should create stack client successfully", func(t *testing.T) {
		cloudClient, err := buildCloudClient(t)
		assert.NoError(t, err)

		stackClient, err := cloudClient.NewStackClient(testStack)
		assert.NoError(t, err)
		assert.NotNil(t, stackClient)
	})

	t.Run("should fail when service account creation fails", func(t *testing.T) {
		cloudClient, err := NewCloudClientWithHttpClient(&http.Client{
			Transport: testutils.RoundTripperFunc(func(req *http.Request) (*http.Response, error) {
				assert.Equal(t, "POST", req.Method)
				assert.Equal(t, fmt.Sprintf("https://grafana.com/api/instances/%d/api/serviceaccounts", testStack.StackID), req.URL.String())
				return testutils.NewHTTPResponseBuilder().
					WithJsonBody(map[string]interface{}{"error": "internal error"}).
					WithStatusCode(http.StatusInternalServerError).Build(), nil
			}),
		})

		assert.NoError(t, err)

		stackClient, err := cloudClient.NewStackClient(testStack)
		assert.Error(t, err)
		assert.Nil(t, stackClient)
		assert.Contains(t, err.Error(), "failed to create Stack Client")
	})
}

func TestEnsureFolder(t *testing.T) {
	os.Setenv("GRAFANA_CLOUD_TOKEN", "fake-token")
	defer os.Unsetenv("GRAFANA_CLOUD_TOKEN")

	t.Run("should return existing folder", func(t *testing.T) {
		cloudClient, err := buildCloudClient(t)
		assert.NoError(t, err)

		stackClient, err := cloudClient.NewStackClientWithHttpClient(testStack, &http.Client{
			Transport: testutils.RoundTripperFunc(func(req *http.Request) (*http.Response, error) {
				assert.Equal(t, "GET", req.Method)
				assert.Equal(t, "https://test-stack.grafana.net/api/folders", req.URL.String())
				return testutils.NewHTTPResponseBuilder().
					WithJsonBody([]map[string]interface{}{{
						"uid":   "test-uid",
						"title": "test",
					}}).
					WithStatusCode(http.StatusOK).Build(), nil
			}),
		})

		assert.NoError(t, err)

		folder, err := stackClient.EnsureFolder(nil, "test")
		assert.NoError(t, err)
		assert.Equal(t, "test", folder.Title)
		assert.Equal(t, "test-uid", folder.UID)
	})

	t.Run("should create folder when it doesn't exist", func(t *testing.T) {
		cloudClient, err := buildCloudClient(t)
		assert.NoError(t, err)

		stackClient, err := cloudClient.NewStackClientWithHttpClient(testStack, &http.Client{
			Transport: testutils.RoundTripperFunc(func(req *http.Request) (*http.Response, error) {
				assert.Equal(t, "https://test-stack.grafana.net/api/folders", req.URL.String())
				switch req.Method {
				case "GET":
					return testutils.NewHTTPResponseBuilder().
						WithJsonBody([]map[string]interface{}{}).
						WithStatusCode(http.StatusOK).Build(), nil
				case "POST":
					require.NotNil(t, req.Body)
					var payload map[string]interface{}
					if err := json.NewDecoder(req.Body).Decode(&payload); err != nil {
						t.Errorf("failed to decode request body: %v", err)
						return nil, fmt.Errorf("failed to decode request body: %w", err)
					}
					assert.Contains(t, payload, "title")
					assert.Equal(t, "test", payload["title"])
					return testutils.NewHTTPResponseBuilder().
						WithJsonBody(map[string]interface{}{
							"uid":   "new-folder-uid",
							"title": "test",
						}).
						WithStatusCode(http.StatusOK).Build(), nil
				default:
					t.Errorf("unexpected request: %s %s", req.Method, req.URL.String())
					return nil, fmt.Errorf("unexpected request: %s %s", req.Method, req.URL.String())
				}
			}),
		})

		assert.NoError(t, err)

		folder, err := stackClient.EnsureFolder(nil, "test")
		assert.NoError(t, err)
		assert.Equal(t, "test", folder.Title)
		assert.Equal(t, "new-folder-uid", folder.UID)
	})

	t.Run("should fail when folder creation fails", func(t *testing.T) {
		cloudClient, err := buildCloudClient(t)
		assert.NoError(t, err)

		stackClient, err := cloudClient.NewStackClientWithHttpClient(testStack, &http.Client{
			Transport: testutils.RoundTripperFunc(func(req *http.Request) (*http.Response, error) {
				assert.Equal(t, "https://test-stack.grafana.net/api/folders", req.URL.String())
				switch req.Method {
				case "GET":
					return testutils.NewHTTPResponseBuilder().
						WithJsonBody([]map[string]interface{}{}).
						WithStatusCode(http.StatusOK).Build(), nil
				case "POST":
					return testutils.NewHTTPResponseBuilder().
						WithJsonBody(map[string]interface{}{"message": "internal error"}).
						WithStatusCode(http.StatusInternalServerError).Build(), nil
				default:
					t.Errorf("unexpected request: %s %s", req.Method, req.URL.String())
					return nil, fmt.Errorf("unexpected request: %s %s", req.Method, req.URL.String())
				}
			}),
		})

		assert.NoError(t, err)

		folder, err := stackClient.EnsureFolder(nil, "test-uid")
		assert.Error(t, err)
		assert.Nil(t, folder)
		assert.Contains(t, err.Error(), "failed to create folder")
	})
}

func TestDeleteDashboard(t *testing.T) {
	os.Setenv("GRAFANA_CLOUD_TOKEN", "fake-token")
	defer os.Unsetenv("GRAFANA_CLOUD_TOKEN")

	t.Run("should successfully delete an existing dashboard", func(t *testing.T) {
		cloudClient, err := buildCloudClient(t)
		assert.NoError(t, err)

		stackClient, err := cloudClient.NewStackClientWithHttpClient(testStack, &http.Client{
			Transport: testutils.RoundTripperFunc(func(req *http.Request) (*http.Response, error) {
				assert.Equal(t, "https://test-stack.grafana.net/api/dashboards/uid/test-dashboard", req.URL.String())
				assert.Equal(t, "DELETE", req.Method)
				return testutils.NewHTTPResponseBuilder().
					WithJsonBody(map[string]interface{}{
						"title":   "Test Dashboard",
						"message": "Dashboard Test Dashboard deleted",
						"id":      1,
					}).
					WithStatusCode(http.StatusOK).Build(), nil
			}),
		})

		assert.NoError(t, err)

		err = stackClient.DeleteDashboard("test-dashboard")
		assert.NoError(t, err)
	})

	t.Run("should handle non-existing dashboard", func(t *testing.T) {
		cloudClient, err := buildCloudClient(t)
		assert.NoError(t, err)

		stackClient, err := cloudClient.NewStackClientWithHttpClient(testStack, &http.Client{
			Transport: testutils.RoundTripperFunc(func(req *http.Request) (*http.Response, error) {
				assert.Equal(t, "https://test-stack.grafana.net/api/dashboards/uid/non-existent", req.URL.String())
				assert.Equal(t, "DELETE", req.Method)
				return testutils.NewHTTPResponseBuilder().
					WithJsonBody(map[string]interface{}{
						"message": "Dashboard not found",
					}).
					WithStatusCode(http.StatusNotFound).Build(), nil
			}),
		})

		assert.NoError(t, err)

		err = stackClient.DeleteDashboard("non-existent")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to delete dashboard")
	})

	t.Run("should handle server errors", func(t *testing.T) {
		cloudClient, err := buildCloudClient(t)
		assert.NoError(t, err)

		stackClient, err := cloudClient.NewStackClientWithHttpClient(testStack, &http.Client{
			Transport: testutils.RoundTripperFunc(func(req *http.Request) (*http.Response, error) {
				assert.Equal(t, "https://test-stack.grafana.net/api/dashboards/uid/test-dashboard", req.URL.String())
				assert.Equal(t, "DELETE", req.Method)
				return testutils.NewHTTPResponseBuilder().
					WithJsonBody(map[string]interface{}{
						"message": "Internal server error",
					}).
					WithStatusCode(http.StatusInternalServerError).Build(), nil
			}),
		})

		assert.NoError(t, err)

		err = stackClient.DeleteDashboard("test-dashboard")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to delete dashboard")
	})
}

func TestUploadDashboard(t *testing.T) {
	os.Setenv("GRAFANA_CLOUD_TOKEN", "fake-token")
	defer os.Unsetenv("GRAFANA_CLOUD_TOKEN")

	t.Run("should successfully upload dashboard", func(t *testing.T) {
		cloudClient, err := buildCloudClient(t)
		assert.NoError(t, err)

		stackClient, err := cloudClient.NewStackClientWithHttpClient(testStack, &http.Client{
			Transport: testutils.RoundTripperFunc(func(req *http.Request) (*http.Response, error) {
				assert.Equal(t, "https://test-stack.grafana.net/api/dashboards/db", req.URL.String())
				assert.Equal(t, "POST", req.Method)
				return testutils.NewHTTPResponseBuilder().
					WithJsonBody(map[string]interface{}{
						"id":      1,
						"uid":     "test-dashboard",
						"status":  "success",
						"version": 1,
					}).
					WithStatusCode(http.StatusOK).Build(), nil
			}),
		})

		assert.NoError(t, err)

		dashboard := &Dashboard{
			UID:       "test-dashboard",
			FolderUID: "test-folder",
			Dashboard: map[string]interface{}{
				"title": "Test Dashboard",
				"uid":   "test-dashboard",
			},
		}

		err = stackClient.UploadDashboard(dashboard)
		assert.NoError(t, err)
	})

	t.Run("should handle server errors", func(t *testing.T) {
		cloudClient, err := buildCloudClient(t)
		assert.NoError(t, err)

		stackClient, err := cloudClient.NewStackClientWithHttpClient(testStack, &http.Client{
			Transport: testutils.RoundTripperFunc(func(req *http.Request) (*http.Response, error) {
				assert.Equal(t, "https://test-stack.grafana.net/api/dashboards/db", req.URL.String())
				assert.Equal(t, "POST", req.Method)
				return testutils.NewHTTPResponseBuilder().
					WithJsonBody(map[string]interface{}{
						"message": "Internal server error",
					}).
					WithStatusCode(http.StatusInternalServerError).Build(), nil
			}),
		})

		assert.NoError(t, err)

		dashboard := &Dashboard{
			UID:       "test-dashboard",
			FolderUID: "test-folder",
			Dashboard: map[string]interface{}{
				"title": "Test Dashboard",
				"uid":   "test-dashboard",
			},
		}

		err = stackClient.UploadDashboard(dashboard)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to updload dashboard")
	})

	t.Run("should handle validation errors", func(t *testing.T) {
		cloudClient, err := buildCloudClient(t)
		assert.NoError(t, err)

		stackClient, err := cloudClient.NewStackClientWithHttpClient(testStack, &http.Client{
			Transport: testutils.RoundTripperFunc(func(req *http.Request) (*http.Response, error) {
				assert.Equal(t, "https://test-stack.grafana.net/api/dashboards/db", req.URL.String())
				assert.Equal(t, "POST", req.Method)
				return testutils.NewHTTPResponseBuilder().
					WithJsonBody(map[string]interface{}{
						"message": "Invalid dashboard format",
					}).
					WithStatusCode(http.StatusBadRequest).Build(), nil
			}),
		})

		assert.NoError(t, err)

		dashboard := &Dashboard{
			UID:       "test-dashboard",
			FolderUID: "test-folder",
			Dashboard: map[string]interface{}{
				"invalid": "dashboard",
			},
		}

		err = stackClient.UploadDashboard(dashboard)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to updload dashboard")
	})
}
