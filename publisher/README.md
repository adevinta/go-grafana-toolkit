# Grafana Publisher

A Go module for programmatically publishing Grafana dashboards to multiple Grafana Cloud stacks.

## Configuration

The publisher module requires a `publisher-config.yaml` file in the root directory. This file defines which dashboards to publish and to which stacks.

### Configuration File Structure

```yaml
# List of stack slugs to exclude from publishing
exclusions:
  - "stackname1"
  - "stackname2"

# Common dashboards that will be published to all non-excluded stacks
commonDashboards:
  localFolder: "path/to/common/dashboards"    # Local folder containing dashboard JSON files
  grafanaFolder: "Common-Folder-Name"         # Destination folder name in Grafana

# Custom dashboards that will be published only to the custom-stack
customDashboards:
  localFolder: "path/to/custom/dashboards"    # Local folder containing dashboard JSON files
  grafanaFolder: "Custom-Folder-Name"         # Destination folder name in Grafana

# Stack slug for custom dashboards
customStack: "stackname"

# Additional tags added to all dashboards
tags:
- automated

# A subfolder where to synchronize the dashboards
rootFolder: some/base/folder

# Stack slug for testing
testStack: "teststackname"

# Append a suffix to each dashboard ID to ensure unicity in the stack
idSuffix: "-pr-1234"
```

## Integration

### Prerequisites

1. Set the `GRAFANA_CLOUD_TOKEN` environment variable with your Grafana Cloud API token:
   ```bash
   export GRAFANA_CLOUD_TOKEN=your-token-here
   ```

### Using the Module

The publisher supports two modes:

1. Test mode - publishes to test stack only:
   ```go
   publisher.Publish(false)
   ```

2. All stacks mode - publishes to all non-excluded stacks:
   ```go
   publisher.Publish(true)
   ```

### Implementation Example

```go
package main

import (
    "github.com/adevinta/go-grafana-toolkit/publisher"
    "log"
)

func main() {
    p, err := publisher.NewPublisher()
    if err != nil {
        log.Fatal(err)
    }

    // Publish to test stack only
    if err := p.Publish(false); err != nil {
        log.Fatal(err)
    }
}
```

## Dashboard Files

- Place dashboard JSON files in the configured local folders
- To delete a dashboard, create a copy of its JSON file with the `.deleted` extension

## Supported datasources

### Metrics datasources

sFor metrics, ensure your dashboard has a datasource variable named `$PROMPRO`.
This variable must be used in all metrics queries instead of the hard-coded datasource.

### Logs datasources

For logs, ensure your dashboard has a datasource variable named `$LOGSPRO`.
This variable must be used in all log queries instead of the hard-coded datasource.

### Log usage datasources

For usage metrics about your stack log ingestion, ensure your dashboard uses a datasource variable named `$LOGUSAGE`.

## File Types

The publisher supports two types of files:
- `.json` - Dashboard definitions to be created/updated
- `.deleted` - Dashboard definitions to be removed

## Error Handling

- The publisher will retry failed uploads for individual stacks
- Detailed logs are provided for any failures
- The process will stop if retries fail
