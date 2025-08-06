package publisher

import (
	"fmt"
	"os"
	"testing"

	grafana "github.com/adevinta/go-grafana-toolkit/client"
	system "github.com/adevinta/go-system-toolkit"
	testutils "github.com/adevinta/go-testutils-toolkit"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

var (
	testStack grafana.Stack = grafana.Stack{
		Slug:     "test-stack",
		StackID:  1,
		StackURL: "https://test-stack.grafana.net",
	}

	customStack grafana.Stack = grafana.Stack{
		Slug:     "custom-stack",
		StackID:  2,
		StackURL: "https://custom-stack.grafana.net",
	}

	rootFolder    *grafana.Folder = &grafana.Folder{UID: "root-folder-uid", Title: "root"}
	rootSubfolder *grafana.Folder = &grafana.Folder{UID: "root-folder-uid-2", Title: "folder"}
	commonFolder  *grafana.Folder = &grafana.Folder{UID: "common-folder-uid", Title: "Common"}

	customFolder *grafana.Folder = &grafana.Folder{UID: "custom-folder-uid", Title: "Custom"}
	nilFolder    *grafana.Folder = nil
)

func TestIsConfigured(t *testing.T) {
	t.Run("when no config file exists", func(t *testing.T) {
		system.DefaultFileSystem = afero.NewMemMapFs()
		defer func() { system.DefaultFileSystem = afero.NewOsFs() }()

		assert.False(t, IsConfigured(""), "should return false when config file doesn't exist")
	})

	t.Run("when config file exists", func(t *testing.T) {
		system.DefaultFileSystem = afero.NewMemMapFs()
		defer func() { system.DefaultFileSystem = afero.NewOsFs() }()

		testutils.EnsureYAMLFileContent(t, system.DefaultFileSystem, "publisher-config.yaml", map[string]interface{}{
			"commonDashboards": map[string]string{
				"localFolder":   "/local_folder_1",
				"grafanaFolder": "Common",
			},
			"testStack": "test-stack",
		})

		assert.True(t, IsConfigured(""), "should return true when config file exists")
	})
}

func TestPublish(t *testing.T) {
	os.Setenv("GRAFANA_CLOUD_TOKEN", "fake-token")
	defer os.Unsetenv("GRAFANA_CLOUD_TOKEN")

	system.DefaultFileSystem = afero.NewMemMapFs()
	defer func() { system.DefaultFileSystem = afero.NewOsFs() }()

	testutils.EnsureYAMLFileContent(t, system.DefaultFileSystem, "publisher-config.yaml", map[string]interface{}{
		"commonDashboards": map[string]string{
			"localFolder":   "/local_folder_1",
			"grafanaFolder": "Common",
		},
		"customDashboards": map[string]string{
			"localFolder":   "/local_folder_2",
			"grafanaFolder": "Custom",
		},
		"customStack": "custom-stack",
		"testStack":   "test-stack",
		"tags":        []string{"tag1", "tag2"},
		"rootFolder":  "root/folder",
		"idSuffix":    "-suffix",
	})

	require.True(t, IsConfigured(""))
	require.NoError(t, system.DefaultFileSystem.MkdirAll("/local_folder_1", 0777))
	require.NoError(t, system.DefaultFileSystem.MkdirAll("/local_folder_2", 0777))

	testutils.EnsureFileContent(t, system.DefaultFileSystem, "/local_folder_1/common_dashboard.json", `{
		"dashboard": {
			"uid": "common-dash-uid",
			"title": "Common Dashboard"
		}
	}`)

	testutils.EnsureFileContent(t, system.DefaultFileSystem, "/local_folder_2/custom_dashboard.json", `{
		"dashboard": {
			"uid": "custom-dash-uid",
			"title": "Custom Dashboard"
		}
	}`)

	stacks := grafana.Stacks{testStack, customStack}

	t.Run("Publish to all stacks", func(t *testing.T) {
		cloudClient := new(MockCloudClient)
		testStackClient := new(MockStackClient)
		customStackClient := new(MockStackClient)

		// Capture uploaded dashboards
		testStackUploadedDashboards := make(map[string]*grafana.Dashboard)
		customStackUploadedDashboards := make(map[string]*grafana.Dashboard)

		cloudClient.
			On("ListStacks").
			Return(stacks, nil).
			Once()
		cloudClient.
			On("NewStackClient", &testStack).
			Return(testStackClient, nil)
		cloudClient.
			On("NewStackClient", &customStack).
			Return(customStackClient, nil)

		// Stack expectations
		// - test-stack stores common only dashboards
		// - custom-stack stores common and custom dashboards
		testStackClient.
			On("EnsureFolder", nilFolder, "root").
			Return(rootFolder, nil)
		testStackClient.
			On("EnsureFolder", rootFolder, "folder").
			Return(rootSubfolder, nil)

		testStackClient.
			On("EnsureFolder", rootSubfolder, "Common").
			Return(commonFolder, nil)

		testStackClient.
			On("UploadDashboard", mock.AnythingOfType("*client.Dashboard")).
			Run(func(args mock.Arguments) {
				dashboard := args.Get(0).(*grafana.Dashboard)
				testStackUploadedDashboards[dashboard.UID] = dashboard
			}).
			Return(nil)

		testStackClient.On("Cleanup").Return(nil)

		customStackClient.
			On("EnsureFolder", nilFolder, "root").
			Return(rootFolder, nil)
		customStackClient.
			On("EnsureFolder", rootFolder, "folder").
			Return(rootSubfolder, nil)
		customStackClient.
			On("EnsureFolder", rootSubfolder, "Common").
			Return(commonFolder, nil)

		customStackClient.
			On("EnsureFolder", rootSubfolder, "Custom").
			Return(customFolder, nil)

		customStackClient.
			On("UploadDashboard", mock.AnythingOfType("*client.Dashboard")).
			Run(func(args mock.Arguments) {
				dashboard := args.Get(0).(*grafana.Dashboard)
				customStackUploadedDashboards[dashboard.UID] = dashboard
			}).
			Return(nil)

		customStackClient.On("Cleanup").Return(nil)

		pub, err := NewPublisherWithCloudClient(cloudClient)
		require.NoError(t, err)

		err = pub.Publish(true)
		assert.NoError(t, err)

		cloudClient.AssertExpectations(t)
		testStackClient.AssertExpectations(t)
		customStackClient.AssertExpectations(t)

		assert.Equal(
			t,
			map[string]*grafana.Dashboard{
				"common-dash-uid-suffix": {
					FolderUID: "common-folder-uid",
					UID:       "common-dash-uid-suffix",
					Dashboard: map[string]interface{}{
						"uid":       "common-dash-uid-suffix",
						"folderUid": "common-folder-uid",
						"title":     "Common Dashboard",
						"tags":      []any{"tag1", "tag2"},
					},
				},
			},
			testStackUploadedDashboards,
		)

		assert.Equal(
			t,
			map[string]*grafana.Dashboard{
				"common-dash-uid-suffix": {
					FolderUID: "common-folder-uid",
					UID:       "common-dash-uid-suffix",
					Dashboard: map[string]interface{}{
						"uid":       "common-dash-uid-suffix",
						"folderUid": "common-folder-uid",
						"title":     "Common Dashboard",
						"tags":      []any{"tag1", "tag2"},
					},
				},
				"custom-dash-uid-suffix": {
					FolderUID: "custom-folder-uid",
					UID:       "custom-dash-uid-suffix",
					Dashboard: map[string]interface{}{
						"uid":       "custom-dash-uid-suffix",
						"folderUid": "custom-folder-uid",
						"title":     "Custom Dashboard",
						"tags":      []any{"tag1", "tag2"},
					},
				},
			},
			customStackUploadedDashboards,
		)
	})

	t.Run("Publish to test stack only", func(t *testing.T) {
		cloudClient := new(MockCloudClient)
		testStackClient := new(MockStackClient)

		testStackUploadedDashboards := make(map[string]*grafana.Dashboard)

		cloudClient.
			On("ListStacks").
			Return(stacks, nil).
			Once()
		cloudClient.
			On("NewStackClient", &testStack).
			Return(testStackClient, nil)

		// Stack expectations
		// - test-stack now stores common and custom dashboards
		// - nothing is stored in custom-stack

		testStackClient.
			On("EnsureFolder", nilFolder, "root").
			Return(rootFolder, nil)
		testStackClient.
			On("EnsureFolder", rootFolder, "folder").
			Return(rootSubfolder, nil)
		testStackClient.
			On("EnsureFolder", rootSubfolder, "Common").
			Return(commonFolder, nil)

		testStackClient.
			On("EnsureFolder", rootSubfolder, "Custom").
			Return(customFolder, nil)

		testStackClient.
			On("UploadDashboard", mock.AnythingOfType("*client.Dashboard")).
			Run(func(args mock.Arguments) {
				dashboard := args.Get(0).(*grafana.Dashboard)
				testStackUploadedDashboards[dashboard.UID] = dashboard
			}).
			Return(nil)

		testStackClient.On("Cleanup").Return(nil)

		pub, err := NewPublisherWithCloudClient(cloudClient)
		require.NoError(t, err)

		err = pub.Publish(false)
		assert.NoError(t, err)

		cloudClient.AssertExpectations(t)
		testStackClient.AssertExpectations(t)

		assert.Equal(
			t,
			map[string]*grafana.Dashboard{
				"common-dash-uid-suffix": {
					FolderUID: "common-folder-uid",
					UID:       "common-dash-uid-suffix",
					Dashboard: map[string]interface{}{
						"uid":       "common-dash-uid-suffix",
						"folderUid": "common-folder-uid",
						"title":     "Common Dashboard",
						"tags":      []any{"tag1", "tag2"},
					},
				},
				"custom-dash-uid-suffix": {
					FolderUID: "custom-folder-uid",
					UID:       "custom-dash-uid-suffix",
					Dashboard: map[string]interface{}{
						"uid":       "custom-dash-uid-suffix",
						"folderUid": "custom-folder-uid",
						"title":     "Custom Dashboard",
						"tags":      []any{"tag1", "tag2"},
					},
				},
			},
			testStackUploadedDashboards,
		)
	})
}

func TestDashboardsHaveDataSourceNamesAndStackIDsInjected(t *testing.T) {
	os.Setenv("GRAFANA_CLOUD_TOKEN", "fake-token")
	defer os.Unsetenv("GRAFANA_CLOUD_TOKEN")

	system.DefaultFileSystem = afero.NewMemMapFs()
	defer func() { system.DefaultFileSystem = afero.NewOsFs() }()

	testutils.EnsureYAMLFileContent(t, system.DefaultFileSystem, "publisher-config.yaml", map[string]interface{}{
		"commonDashboards": map[string]string{
			"localFolder":   "/local_folder_1",
			"grafanaFolder": "Common",
		},
		"testStack": "test-stack",
	})

	require.True(t, IsConfigured(""))

	require.NoError(t, system.DefaultFileSystem.MkdirAll("/local_folder_1", 0777))
	testutils.EnsureFileContent(t, system.DefaultFileSystem, "/local_folder_1/dashboard1.json", `{
		"dashboard":{
			"uid":"dash-1",
			 "templating": {
				"list": [
					{"type": "datasource", "name": "PROMPRO"},
					{"type": "datasource", "name": "P1EUW1"},
					{"type": "datasource", "name": "LOGSPRO"},
					{"type": "datasource", "name": "LOGUSAGE"},
					{
						"type": "custom", "name": "STACKID",
						"current": {"selected": true, "text": "0123"},
						"options": [{"selected": false}]
					}
				]
			}
		}
	}`)

	cloudClient := new(MockCloudClient)
	testStackClient := new(MockStackClient)

	var uploadedDashboard *grafana.Dashboard

	cloudClient.
		On("ListStacks").
		Return(grafana.Stacks{testStack}, nil).
		Once()
	cloudClient.
		On("NewStackClient", &testStack).
		Return(testStackClient, nil)

	testStackClient.
		On("EnsureFolder", nilFolder, "Common").
		Return(commonFolder, nil)

	testStackClient.
		On("GetDataSource", "grafanacloud-test-stack-logs").
		Return(&grafana.Datasource{
			Name: "grafanacloud-test-stack-logs",
			User: "123456", // This will be used as the stack ID
		}, nil).
		Once()

	testStackClient.
		On("UploadDashboard", mock.AnythingOfType("*client.Dashboard")).
		Run(func(args mock.Arguments) {
			uploadedDashboard = args.Get(0).(*grafana.Dashboard)
		}).
		Return(nil).
		Once()

	testStackClient.On("Cleanup").Return(nil)

	pub, err := NewPublisherWithCloudClient(cloudClient)
	require.NoError(t, err)

	err = pub.Publish(true)
	assert.NoError(t, err)

	cloudClient.AssertExpectations(t)
	testStackClient.AssertExpectations(t)

	// Verify uploaded dashboard has injected datasource names and stack IDs
	assert.Equal(t, map[string]interface{}{
		"uid":       "dash-1",
		"folderUid": "common-folder-uid",
		"templating": map[string]interface{}{
			"list": []interface{}{
				map[string]interface{}{
					"type": "datasource",
					"name": "PROMPRO",
					"current": map[string]interface{}{
						"selected": false,
						"text":     "grafanacloud-test-stack-prom",
						"value":    "grafanacloud-test-stack-prom",
					},
				},
				map[string]interface{}{
					"type": "datasource",
					"name": "P1EUW1",
					"current": map[string]interface{}{
						"selected": false,
						"text":     "grafanacloud-test-stack-prom",
						"value":    "grafanacloud-test-stack-prom",
					},
				},
				map[string]interface{}{
					"type": "datasource",
					"name": "LOGSPRO",
					"current": map[string]interface{}{
						"selected": false,
						"text":     "grafanacloud-test-stack-logs",
						"value":    "grafanacloud-test-stack-logs",
					},
				},
				map[string]interface{}{
					"type": "datasource",
					"name": "LOGUSAGE",
					"current": map[string]interface{}{
						"selected": false,
						"text":     "grafanacloud-test-stack-usage-insights",
						"value":    "grafanacloud-usage-insights",
					},
				},
				map[string]interface{}{
					"type": "custom",
					"name": "STACKID",
					"current": map[string]interface{}{
						"selected": false,
						"text":     "123456",
						"value":    "123456",
					},
					"options": []map[string]interface{}{
						{
							"selected": true,
							"text":     "123456",
							"value":    "123456",
						},
					},
					"query": "123456",
				},
			},
		},
	}, uploadedDashboard.Dashboard)
}

func TestDashboardsAreDeleted(t *testing.T) {
	os.Setenv("GRAFANA_CLOUD_TOKEN", "fake-token")
	defer os.Unsetenv("GRAFANA_CLOUD_TOKEN")

	system.DefaultFileSystem = afero.NewMemMapFs()
	defer func() { system.DefaultFileSystem = afero.NewOsFs() }()

	testutils.EnsureYAMLFileContent(t, system.DefaultFileSystem, "publisher-config.yaml", map[string]interface{}{
		"commonDashboards": map[string]string{
			"localFolder":   "/local_folder_1",
			"grafanaFolder": "Common",
		},
		"testStack": "test-stack",
	})

	require.True(t, IsConfigured(""))

	require.NoError(t, system.DefaultFileSystem.MkdirAll("/local_folder_1", 0777))
	testutils.EnsureFileContent(t, system.DefaultFileSystem, "/local_folder_1/dashboard1.json.deleted", `{"dashboard": {"uid": "dash-1"}}`)

	cloudClient := new(MockCloudClient)
	testStackClient := new(MockStackClient)

	cloudClient.
		On("ListStacks").
		Return(grafana.Stacks{testStack}, nil).
		Once()
	cloudClient.
		On("NewStackClient", &testStack).
		Return(testStackClient, nil)

	testStackClient.
		On("EnsureFolder", nilFolder, "Common").
		Return(commonFolder, nil)

	testStackClient.
		On("GetDashboard", "dash-1").
		Return(&grafana.Dashboard{
			UID:       "dash-1",
			Dashboard: map[string]interface{}{"uid": "dash-1"},
		}, nil).
		Once()

	testStackClient.
		On("DeleteDashboard", "dash-1").
		Return(nil).
		Once()

	testStackClient.On("Cleanup").Return(nil)

	pub, err := NewPublisherWithCloudClient(cloudClient)
	require.NoError(t, err)

	err = pub.Publish(true)
	assert.NoError(t, err)

	cloudClient.AssertExpectations(t)
	testStackClient.AssertExpectations(t)
}

func TestPublishRetriesOncePerStack(t *testing.T) {
	os.Setenv("GRAFANA_CLOUD_TOKEN", "fake-token")
	defer os.Unsetenv("GRAFANA_CLOUD_TOKEN")

	system.DefaultFileSystem = afero.NewMemMapFs()
	defer func() { system.DefaultFileSystem = afero.NewOsFs() }()

	testutils.EnsureYAMLFileContent(t, system.DefaultFileSystem, "publisher-config.yaml", map[string]interface{}{
		"commonDashboards": map[string]string{
			"localFolder":   "/local_folder_1",
			"grafanaFolder": "Common",
		},
		"testStack": "test-stack",
	})

	require.True(t, IsConfigured(""))

	require.NoError(t, system.DefaultFileSystem.MkdirAll("/local_folder_1", 0777))
	testutils.EnsureFileContent(t, system.DefaultFileSystem, "/local_folder_1/dashboard1.json", `{
        "dashboard": {
            "uid": "dash-1",
            "title": "Test Dashboard"
        }
    }`)

	cloudClient := new(MockCloudClient)
	testStackClient := new(MockStackClient)

	var uploadAttempts []*grafana.Dashboard

	cloudClient.
		On("ListStacks").
		Return(grafana.Stacks{testStack}, nil).
		Once()
	cloudClient.
		On("NewStackClient", &testStack).
		Return(testStackClient, nil)

	testStackClient.
		On("EnsureFolder", nilFolder, "Common").
		Return(commonFolder, nil)

	testStackClient.
		On("UploadDashboard", mock.AnythingOfType("*client.Dashboard")).
		Run(func(args mock.Arguments) {
			dashboard := args.Get(0).(*grafana.Dashboard)
			uploadAttempts = append(uploadAttempts, dashboard)
		}).
		Return(fmt.Errorf("first attempt failed")).
		Once()

	testStackClient.
		On("UploadDashboard", mock.AnythingOfType("*client.Dashboard")).
		Run(func(args mock.Arguments) {
			dashboard := args.Get(0).(*grafana.Dashboard)
			uploadAttempts = append(uploadAttempts, dashboard)
		}).
		Return(nil).
		Once()

	testStackClient.On("Cleanup").Return(nil)

	pub, err := NewPublisherWithCloudClient(cloudClient)
	require.NoError(t, err)

	err = pub.Publish(true)
	assert.NoError(t, err)

	cloudClient.AssertExpectations(t)
	testStackClient.AssertExpectations(t)

	// Verify retry behavior
	assert.Len(t, uploadAttempts, 2, "should attempt upload twice")

	// Verify both attempts were for the same dashboard
	for _, attempt := range uploadAttempts {
		dash := attempt.Dashboard.(map[string]interface{})
		assert.Equal(t, "dash-1", dash["uid"], "both attempts should be for the same dashboard")
	}
}
