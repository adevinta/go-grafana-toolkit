// Package publisher provides functionality to publish Grafana dashboards to multiple Grafana Cloud stacks.
// It supports publishing common dashboards to all stacks and custom dashboards to specific stacks.
// The publisher can operate in test mode (single stack) or production mode (all non-excluded stacks).
package publisher

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	grafana "github.com/adevinta/go-grafana-toolkit/client"
	log "github.com/adevinta/go-log-toolkit"
	system "github.com/adevinta/go-system-toolkit"
	"github.com/spf13/afero"
	"gopkg.in/yaml.v3"
)

const configFilePath = "publisher-config.yaml"

// Publisher manages the publishing of Grafana dashboards to multiple stacks.
// It uses a configuration file to determine which dashboards to publish and to which stacks.
type Publisher struct {
	config *PublisherConfig
	gcc    grafana.GrafanaCloudClient
}

// IsConfigured checks if the publisher configuration file exists.
func IsConfigured() bool {
	_, err := system.DefaultFileSystem.Stat(configFilePath)
	return err == nil
}

// loadPublisherConfig reads and parses the publisher configuration file.
// Returns an error if the file cannot be read or parsed.
func loadPublisherConfig() (*PublisherConfig, error) {
	_, err := system.DefaultFileSystem.Stat(configFilePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open config file %s: %w", configFilePath, err)
	}

	data, err := afero.ReadFile(system.DefaultFileSystem, configFilePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file %s: %w", configFilePath, err)
	}

	var cfg PublisherConfig
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("failed to parse config file %s: %w", configFilePath, err)
	}

	cfg.initExclusionsMap()
	return &cfg, nil
}

// NewPublisher creates a new Publisher instance.
// It loads the configuration from the publisher-config.yaml file.
// Returns an error if the configuration file cannot be loaded or parsed.
func NewPublisher() (*Publisher, error) {
	return newPublisher(nil)
}

func NewPublisherWithCloudClient(gcc grafana.GrafanaCloudClient) (*Publisher, error) {
	return newPublisher(gcc)
}

func newPublisher(gcc grafana.GrafanaCloudClient) (*Publisher, error) {
	cfg, err := loadPublisherConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to load config: %w", err)
	}

	return &Publisher{
		config: cfg,
		gcc:    gcc,
	}, nil
}

// Publish synchronizes dashboards with Grafana Cloud stacks according to the configuration.
// If syncAllStacks is true, it publishes to all non-excluded stacks.
// If syncAllStacks is false, it publishes only to the test stack.
// Requires GRAFANA_CLOUD_TOKEN environment variable to be set.
// Returns an error if the synchronization fails.
func (p Publisher) Publish(syncAllStacks bool) error {

	if _, ok := os.LookupEnv("GRAFANA_CLOUD_TOKEN"); !ok {
		fmt.Fprint(os.Stderr, "GRAFANA_CLOUD_TOKEN not set, skipping grafana sync")
		return nil
	}

	if p.gcc == nil {
		cloudClient, err := grafana.NewCloudClient()
		if err != nil {
			return fmt.Errorf("failed to create Grafana Cloud client: %w", err)
		}
		p.gcc = cloudClient
	}

	stacksWithCommonDashboards, err := p.gcc.ListStacks()
	if err != nil {
		return fmt.Errorf("failed to list stacks: %w", err)
	}

	stacks := grafana.Stacks{}

	for _, stack := range stacksWithCommonDashboards {
		if _, ok := p.config.ExclusionsMap()[stack.Slug]; !ok {
			log.DefaultLogger.WithField("stack", stack.Slug).Println("is not excluded, adding it to the candidates")
			stacks = append(stacks, stack)
		} else {
			log.DefaultLogger.WithField("stack", stack.Slug).Println("is excluded, skipping")
		}
	}

	stacksWithCommonDashboards = stacks
	var stacksWithCustomDashboards grafana.Stacks
	if syncAllStacks {
		log.DefaultLogger.Println("Syncing all stacks")
		stacksWithCustomDashboards = grafana.Stacks{stackByName(&stacksWithCommonDashboards, p.config.CustomStack)}
	} else {
		log.DefaultLogger.Printf("Syncing only %s stack", p.config.TestStack)
		testStack := stackByName(&stacksWithCommonDashboards, p.config.TestStack)
		stacksWithCommonDashboards = grafana.Stacks{testStack}
		stacksWithCustomDashboards = grafana.Stacks{testStack}
	}

	for _, customDashboard := range p.config.CustomDashboards {
		localFolder := customDashboard.LocalFolder
		grafanaFolder := customDashboard.GrafanaFolder
		if localFolder != "" && grafanaFolder != "" {
			err = syncDashboards(&stacksWithCustomDashboards, localFolder, grafanaFolder, p.gcc)
			if err != nil {
				return fmt.Errorf("sync failed (%s -> %s): %w", localFolder, grafanaFolder, err)
			}
		}
	}

	for _, commonDashboard := range p.config.CommonDashboards {
		localFolder := commonDashboard.LocalFolder
		grafanaFolder := commonDashboard.GrafanaFolder
		if localFolder != "" && grafanaFolder != "" {
			err = syncDashboards(&stacksWithCommonDashboards, localFolder, grafanaFolder, p.gcc)
			if err != nil {
				return fmt.Errorf("sync failed (%s -> %s): %w", localFolder, grafanaFolder, err)
			}
		}
	}

	return nil
}

type failedStack struct {
	stack *grafana.Stack
	err   error
}

// syncDashboards synchronizes dashboards from a local folder to specified Grafana stacks.
// It handles both dashboard creation/updates and deletions.
// Returns an error if the synchronization fails.
func syncDashboards(grafanaStacks *grafana.Stacks, localFolder, grafanaFolder string, gcc grafana.GrafanaCloudClient) error {

	stackSlugs := []string{}
	for _, stack := range *grafanaStacks {
		stackSlugs = append(stackSlugs, stack.Slug)
	}

	log.DefaultLogger.WithField("stacks", stackSlugs).WithField("localFolder", localFolder).WithField("grafanaFolder", grafanaFolder).Println("Syncing dashboards...")

	_, err := system.DefaultFileSystem.Stat(localFolder)
	if err != nil {
		if !os.IsNotExist(err) {
			return fmt.Errorf("Failed to discover %s: %w", localFolder, err)
		}
		log.DefaultLogger.WithField("localFolder", localFolder).WithField("grafanaFolder", grafanaFolder).Info("Local folder not present, skipping sync.")
		return nil
	}

	failedStacks := []failedStack{}

	for _, stack := range *grafanaStacks {
		err := syncDashboardsForStack(&stack, localFolder, grafanaFolder, gcc)
		if err != nil {
			failedStacks = append(failedStacks, failedStack{
				stack: &stack,
				err:   err,
			})
		}
	}

	if len(failedStacks) > 0 {
		log.DefaultLogger.Errorf("Number of failed stacks: %d.", len(failedStacks))

		for _, failedStack := range failedStacks {
			log.DefaultLogger.WithField("failedStack", failedStack.stack.Slug).Errorf("Failed to sync dashboards: %v", failedStack.err)
		}

		log.DefaultLogger.WithField("localFolder", localFolder).WithField("grafanaFolder", grafanaFolder).Println("Retrying...")

		for _, failedStack := range failedStacks {
			err := syncDashboardsForStack(failedStack.stack, localFolder, grafanaFolder, gcc)
			if err != nil {
				return fmt.Errorf("Retry of stack %s failed: %w", failedStack.stack.Slug, err)
			}
		}
	}

	return nil
}

// stackByName finds a stack by its name in the provided list of stacks.
// Returns an empty Stack if not found.
func stackByName(stacks *grafana.Stacks, name string) grafana.Stack {
	for _, stack := range *stacks {
		if stack.Slug == name {
			return stack
		}
	}
	return grafana.Stack{}
}

// syncDashboardsForStack synchronizes dashboards for a single Grafana stack.
// Handles folder creation, dashboard uploads, and dashboard deletions.
// Returns an error if any operation fails.
func syncDashboardsForStack(stack *grafana.Stack, localFolder, grafanaFolder string, gcc grafana.GrafanaCloudClient) error {

	sc, err := gcc.NewStackClient(stack)

	if err != nil {
		return fmt.Errorf("failed to get grafana stack client for stack %v, error: %w", stack.Slug, err)
	}

	defer sc.Cleanup()

	folder, err := sc.EnsureFolder(grafanaFolder)

	if err != nil {
		return fmt.Errorf("could not ensure folder %s: %w", grafanaFolder, err)
	}

	err = afero.Walk(system.DefaultFileSystem, localFolder, func(path string, info os.FileInfo, err error) error {

		if err != nil {
			return err
		}

		if info == nil {
			return errors.New("nil info handler for path: " + path)
		}

		if info.IsDir() {
			return nil
		}

		switch filepath.Ext(path) {
		case ".json":
			log.DefaultLogger.WithField("dashboard", path).WithField("destination", stack.Slug).Println("Syncing dashboard")
			fd, err := system.DefaultFileSystem.Open(path)

			if err != nil {
				return err
			}

			defer fd.Close()

			dashboard := map[string]interface{}{}
			err = json.NewDecoder(fd).Decode(&dashboard)

			if err != nil {
				return err
			}

			if dashboard["dashboard"] == nil {
				return fmt.Errorf("unable to find dashboard in %s", path)
			}

			dash := dashboard["dashboard"].(map[string]interface{})
			delete(dash, "folderId")
			dash["folderUid"] = folder.UID

			if dash["templating"] != nil {

				templating := dash["templating"].(map[string]interface{})
				parameters := templating["list"].([]interface{})

				for _, param := range parameters {
					parameter := param.(map[string]interface{})
					if parameter["type"] == "datasource" {
						switch parameter["name"] {
						case "PROMPRO", "P1EUW1":
							datasourceName := fmt.Sprintf("grafanacloud-%s-prom", stack.Slug)
							parameter["current"] = map[string]interface{}{
								"selected": false,
								"text":     datasourceName,
								"value":    datasourceName,
							}
						case "LOGSPRO":
							datasourceName := fmt.Sprintf("grafanacloud-%s-logs", stack.Slug)
							parameter["current"] = map[string]interface{}{
								"selected": false,
								"text":     datasourceName,
								"value":    datasourceName,
							}
						case "LOGUSAGE":
							datasourceName := fmt.Sprintf("grafanacloud-%s-usage-insights", stack.Slug)
							parameter["current"] = map[string]interface{}{
								"selected": false,
								"text":     datasourceName,
								"value":    "grafanacloud-usage-insights",
							}
						}
					}

					if parameter["type"] == "custom" {
						if parameter["name"] == "STACKID" {
							datasourceName := fmt.Sprintf("grafanacloud-%s-logs", stack.Slug)
							datasource, err := sc.GetDataSource(datasourceName)
							if err != nil {
								return err
							}

							stackid := datasource.User

							parameter["current"] = map[string]interface{}{
								"selected": false,
								"text":     stackid,
								"value":    stackid,
							}
							parameter["options"] = []map[string]interface{}{
								{
									"selected": true,
									"text":     stackid,
									"value":    stackid,
								},
							}
							parameter["query"] = stackid
						}
					}
				}
			}

			// Grafana API will return 404 if 'id' is present, use just uid.
			delete(dash, "id")

			err = sc.UploadDashboard(&grafana.Dashboard{
				FolderUID: folder.UID,
				Dashboard: dash,
			})

			if err != nil {
				return fmt.Errorf("failed to upload dashboard %s: %w", folder.UID, err)
			}

		case ".deleted":
			log.DefaultLogger.WithField("dashboard", path).WithField("destination", stack.Slug).Println("Deleting dashboard")
			fd, err := system.DefaultFileSystem.Open(path)
			if err != nil {
				return err
			}
			defer fd.Close()
			dashboard := map[string]interface{}{}
			err = json.NewDecoder(fd).Decode(&dashboard)
			if err != nil {
				return err
			}
			if dashboard["dashboard"] == nil {
				return fmt.Errorf("unable to find dashboard in %s", path)
			}
			dash := dashboard["dashboard"].(map[string]interface{})
			if dash["uid"] == nil {
				return fmt.Errorf("unable to find dashboard uid in %s", path)
			}
			dashboardUID, ok := dash["uid"].(string)
			if !ok {
				return fmt.Errorf("dashboard uid %s is not a string in path %s", dashboardUID, path)
			}

			_, err = sc.GetDashboard(dashboardUID)
			if err == nil {
				err = sc.DeleteDashboard(dashboardUID)
				if err != nil {
					return err
				}
			}

		default:
			return fmt.Errorf("unsupported file extension %s for path %v", filepath.Ext(path), path)
		}
		return nil
	})

	if err != nil {
		return err
	}

	return nil
}
