package publisher

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"gopkg.in/yaml.v3"
)

func TestYAMLUnmarshal(t *testing.T) {
	t.Run("when single dashboard reference is provided", func(t *testing.T) {
		var config PublisherConfig
		err := yaml.Unmarshal([]byte(`
commonDashboards:
  localFolder: /local_folder_1
  grafanaFolder: Common
`), &config)
		assert.NoError(t, err)
		assert.Equal(t, DashboardReferences{
			{
				LocalFolder:   "/local_folder_1",
				GrafanaFolder: "Common",
			},
		}, config.CommonDashboards)
	})

	t.Run("when multiple dashboard references are provided", func(t *testing.T) {
		var config PublisherConfig
		err := yaml.Unmarshal([]byte(`
commonDashboards:
- localFolder: /local_folder_1
  grafanaFolder: Common
- localFolder: /local_folder_2
  grafanaFolder: Custom
`), &config)
		assert.NoError(t, err)
		assert.Equal(t, DashboardReferences{
			{
				LocalFolder:   "/local_folder_1",
				GrafanaFolder: "Common",
			},
			{
				LocalFolder:   "/local_folder_2",
				GrafanaFolder: "Custom",
			},
		}, config.CommonDashboards)
	})
}
