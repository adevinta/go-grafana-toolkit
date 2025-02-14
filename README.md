# Grafana Modules

A collection of Go modules for working with Grafana Cloud, focusing on dashboard management and Grafana API interactions.

## Modules

### Publisher

The Dashboard Publisher module provides programmatic control for deploying Grafana dashboards across multiple Grafana Cloud stacks. 

It supports:

- Publishing common dashboards to multiple stacks
- Managing stack-specific dashboards
- Test deployments to a designated stack
- Dashboard deletion tracking
- Automatic datasource variable handling

[Learn more about the Publisher](./publisher/README.md)

### Client

A Go module that provides a convenient wrapper around:
- Grafana Cloud API (using `grafana-com-public-clients/go/gcom`)
- Grafana HTTP API (using `grafana-openapi-client-go`)

[Learn more about the Client](./client/README.md)

## Usage

1. Import the modules in your Go code:
   ```go
   import (
       "github.com/adevinta/go-grafana-toolkit/publisher"
       "github.com/adevinta/go-grafana-toolkit/client"
   )
   ```

2. Set up your Grafana Cloud API token in the `GRAFANA_CLOUD_TOKEN` env varible.

3. Configure your dashboard publishing strategy using `publisher-config.yaml`

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request.

## License

This project is licensed under the MIT License - see the LICENSE file for details.
