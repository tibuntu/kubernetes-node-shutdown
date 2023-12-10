# Kubernetes Node Shutdown

A small application to watch nodes and to shut them down if certain criteria is met.

## What problem is this solving?

I wrote this tool because I have a couple of nodes in my private Kubernetes cluster, that are dedicated to on/off workloads (E.g.: Plex Media Server). In order to save some energy costs, I want to shut them down if this kind of workload is unused and therefore the node isn't utilized. Because running proper cluster autoscalers like Karpenter isn't possible, I came up with this solution.

## Usage and deployment

### Environment variables

The following environment variables can be used to control the behaviour:

| Key | Description | Default |
| --- | ----------- | ------- |
| MEMORY_THRESHOLD | Threshold in megabytes that will cause a shutdown | None |
| CPU_THRESHOLD | Threshold in mili CPU that will cause a shutdown | None |
| SHUTDOWN_DELAY_MINUTES | Delay in minutes before performing the shutdown | None |
| SSH_USERNAME | Username used to establish the SSH connection | None |
| SSH_PRIVATE_KEY_PATH | Path to the private key that is used for the SSH authentication | None |
| SSH_PORT | Port used to establish the SSH connection | 22 |
| NODE_NAMES | Comma seperated list of nodes to watch | None |
| DRY_RUN_MODE | Watch resource usage of the nodes, but do not initiate a shutdown | false |
| TZ | Set a timezone that is used for logging | UTC |

### Deployment via Helm

A [Helm Chart](https://artifacthub.io/packages/helm/tibuntu/kubernetes-node-shutdown) is available and will receive regular updates.

## About

`kubernetes-node-shutdown` is currently maintained by [tibuntu][profile].

Anyone that wants to contribute is highly welcome!

[profile]: https://github.com/tibuntu

## License

GNU General Public License v3.0
