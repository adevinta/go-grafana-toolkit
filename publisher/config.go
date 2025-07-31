package publisher

import (
	"fmt"

	"gopkg.in/yaml.v3"
)

type DashboardReferences []DashboardReference

type DashboardReference struct {
	LocalFolder   string `yaml:"localFolder"`
	GrafanaFolder string `yaml:"grafanaFolder"`
}

// UnmarshalYAML implements custom unmarshaling for DashboardReferences
// to support both single DashboardReference and list of DashboardReference
func (dr *DashboardReferences) UnmarshalYAML(value *yaml.Node) error {
	// Check if the value is a sequence (list)
	if value.Kind == yaml.SequenceNode {
		// It's a list, unmarshal normally
		var refs []DashboardReference
		if err := value.Decode(&refs); err != nil {
			return fmt.Errorf("failed to unmarshal dashboard references list: %w", err)
		}
		*dr = DashboardReferences(refs)
		return nil
	}

	// Check if the value is a mapping (single object)
	if value.Kind == yaml.MappingNode {
		// It's a single DashboardReference, unmarshal and wrap in slice
		var ref DashboardReference
		if err := value.Decode(&ref); err != nil {
			return fmt.Errorf("failed to unmarshal single dashboard reference: %w", err)
		}
		*dr = DashboardReferences{ref}
		return nil
	}

	return fmt.Errorf("dashboard references must be either a single object (kind: %v) or a list of objects (kind: %v), but got kind: %v", yaml.MappingNode, yaml.SequenceNode, value.Kind)
}

// MarshalYAML implements custom marshaling for DashboardReferences
// Always marshals as a list for consistency
func (dr DashboardReferences) MarshalYAML() (interface{}, error) {
	// Always marshal as a list, even if it's a single item
	return []DashboardReference(dr), nil
}

type PublisherConfig struct {
	Exclusions    []string            `yaml:"exclusions,omitempty"`
	exclusionsMap map[string]struct{} `yaml:"-"` // Private field, not marshaled

	CommonDashboards DashboardReferences `yaml:"commonDashboards"`

	CustomDashboards DashboardReferences `yaml:"customDashboards"`

	CustomStack string `yaml:"customStack"`
	TestStack   string `yaml:"testStack"`
}

func (c *PublisherConfig) initExclusionsMap() {
	c.exclusionsMap = make(map[string]struct{}, len(c.Exclusions))
	for _, e := range c.Exclusions {
		c.exclusionsMap[e] = struct{}{}
	}
}

func (c *PublisherConfig) ExclusionsMap() map[string]struct{} {
	return c.exclusionsMap
}
