package publisher

type PublisherConfig struct {
	Exclusions    []string            `yaml:"exclusions,omitempty"`
	exclusionsMap map[string]struct{} `yaml:"-"` // Private field, not marshaled

	CommonDashboard struct {
		LocalFolder   string `yaml:"localFolder"`
		GrafanaFolder string `yaml:"grafanaFolder"`
	} `yaml:"commonDashboards"`

	CustomDashboard struct {
		LocalFolder   string `yaml:"localFolder"`
		GrafanaFolder string `yaml:"grafanaFolder"`
	} `yaml:"customDashboards"`

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
