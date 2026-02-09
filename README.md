[![REUSE status](https://api.reuse.software/badge/github.com/kyma-project/rt-bootstrapper)](https://api.reuse.software/info/github.com/kyma-project/rt-bootstrapper)
[![Go Report Card](https://goreportcard.com/badge/github.com/kyma-project/rt-bootstrapper)](https://goreportcard.com/report/github.com/kyma-project/rt-bootstrapper)
[![unit tests](https://badgers.space/github/checks/kyma-project/rt-bootstrapper/main/unit-tests)](https://github.com/kyma-project/rt-bootstrapper/actions/workflows/unit-tests.yaml)
[![Coverage Status](https://coveralls.io/repos/github/kyma-project/rt-bootstrapper/badge.svg?branch=main)](https://coveralls.io/github/kyma-project/rt-bootstrapper?branch=main)
[![golangci lint](https://badgers.space/github/checks/kyma-project/rt-bootstrapper/main/golangci-lint)](https://github.com/kyma-project/rt-bootstrapper/actions/workflows/lint.yaml)
[![latest release](https://badgers.space/github/release/kyma-project/rt-bootstrapper)](https://github.com/kyma-project/rt-bootstrapper/releases/latest)

# RT Bootstrapper

Test if it will not trigger vulncheck

This repository contains the source code for the RT Bootstrapper Kyma component used to configure Kyma runtime components running in markets with individual infrastructure setups.

## Overview

RT Bootstrapper contains two functional parts:

- Kubernetes admission webhook that intercepts the creation of Pods.
  It modifies the Pod specifications to include necessary configurations, modifies image paths to use the configured remote registry, and provides pull secrets with credentials.

- Kubernetes Controller that watches for namespaces and ensures that the secrets with required credentials are present and synchronized in those namespaces.



> [!NOTE]
> This component is implemented as part of the Kyma runtime delivery.  
> Installing RT Bootstrapper in SAP BTP, Kyma runtime, or in a self-managed Kyma runtime cluster may negatively impact your workloads.

## Installation

### Prerequisites

- SAP BTP, Kyma runtime instance
- Access to the Kyma runtime cluster with kubeconfig

### Installation with Kyma Control Plane

In environments with individual infrastructure setups, RT Bootstrapper is installed and configured automatically by Kyma Control Plane in all provisioned Kyma runtimes.

### Installation with kubectl

To enable RT Bootstrapper in your Kyma cluster, apply the release manifest using kubectl:  

```bash
kubectl apply -f https://github.com/kyma-project/rt-bootstrapper/releases/latest/download/rt-bootstrapper.yaml
```
## Architectural Decisions
See the [Architectural Decisions](./docs/contributor/architectural-decisions.md) file.
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

2. Create a new k3d cluster and run RT Bootstrapper from the main branch:

    ```bash
    k3d cluster create test-cluster
    make deploy
    ```
## Usage

To use RT Bootstrapper, label your Kubernetes namespaces and Pods accordingly.   
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
