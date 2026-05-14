[![REUSE status](https://api.reuse.software/badge/github.com/kyma-project/rt-bootstrapper)](https://api.reuse.software/info/github.com/kyma-project/rt-bootstrapper)
[![Go Report Card](https://goreportcard.com/badge/github.com/kyma-project/rt-bootstrapper)](https://goreportcard.com/report/github.com/kyma-project/rt-bootstrapper)
[![golangci lint](https://badgers.space/github/checks/kyma-project/rt-bootstrapper/main/golangci-lint)](https://github.com/kyma-project/rt-bootstrapper/actions/workflows/lint.yaml)

# Runtime Bootstrapper

This repository contains the source code for the Runtime Bootstrapper (rt-bootstrapper) Kyma component used to configure Kyma runtime components running in markets with individual infrastructure setups.

## Overview

Runtime Bootstrapper contains two functional parts:

- Kubernetes admission webhook that intercepts the creation of Pods.
  It modifies the Pod specifications to include necessary configurations, modifies image paths to use the configured remote registry, and provides pull secrets with credentials.

- Kubernetes Controller that watches for namespaces and ensures that the Secrets with required credentials are present and synchronized in those namespaces.

> [!NOTE]
> This component is implemented as part of the SAP BTP, Kyma runtime delivery.  
> Installing Runtime Bootstrapper in Kyma runtime, or in a self-managed Kyma cluster may negatively impact your workloads.

## Installation

### Prerequisites

- SAP BTP, Kyma runtime instance
- Access to the Kyma runtime cluster with a kubeconfig

### Installation with Kyma Control Plane

In environments with individual infrastructure setups, Runtime Bootstrapper is installed and configured automatically by Kyma Control Plane (KCP) in all provisioned Kyma runtimes.

### Installation with kubectl

To enable Runtime Bootstrapper in your Kyma cluster, apply the release manifest using kubectl:  

```bash
kubectl apply -f https://github.com/kyma-project/rt-bootstrapper/releases/latest/download/rt-bootstrapper.yaml
```

## Architectural Documentation<!--??-->

For architecture details, see [Architectural Documentation of Runtime Bootstrapper](./docs/contributor/README.md).<!--??-->

## Development

### Prerequisites

- Access to a Kubernetes cluster
- [Go](https://go.dev/)
- [k3d](https://k3d.io/)
- [Docker](https://www.docker.com/)
- [kubectl](https://kubernetes.io/docs/tasks/tools/)
- [Kubebuilder](https://book.kubebuilder.io/)
- [yq](https://mikefarah.gitbook.io/yq)

### Installation in the k3d Cluster Using Make Targets

1. Clone the project.

    ```bash
    git clone https://github.com/kyma-project/rt-boostrapper.git && cd rt-boostrapper/
    ```

2. Create a new k3d cluster and run Runtime Bootstrapper from the main branch.

    ```bash
    k3d cluster create test-cluster
    make deploy
    ```

## Usage

To use Runtime Bootstrapper, label your Kubernetes namespaces and Pods accordingly.<!--why 'accordingly'? wouldn't properly be a better choice?-->
The admission webhook intercepts the creation of these resources and applies the necessary configurations.

## Contributing

<!--- mandatory section - do not change this! --->

See the [Contributing Rules](CONTRIBUTING.md).

## Code of Conduct
<!--- mandatory section - do not change this! --->

See the [Code of Conduct](CODE_OF_CONDUCT.md) document.

## License

<!--- mandatory section - do not change this! --->

See the [license](./LICENSE) file.
