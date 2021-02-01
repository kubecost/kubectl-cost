# kubectl-cost

`kubectl-cost` is a [kubectl plugin](https://kubernetes.io/docs/tasks/extend-kubectl/kubectl-plugins/) that provides easy CLI access to some of the [kubecost APIs](https://github.com/kubecost/docs/blob/master/apis.md).

## Installation

#### This software requires that you have a running deployment of kubecost in your cluster. See [our docs](https://docs.kubecost.com/install) for installation instructions.

Build:

The build currently relies on this repository having a sibling directory containing the [cost-model](https://github.com/kubecost/cost-model/) repository. A future version will not have that requirement.

``` sh
make build
```

Install:

``` sh
chmod +x cmd/kubectl-cost
cp cmd/kubectl-cost /somewhere/in/your/PATH/kubectl-cost
```

As long as the binary is still named `kubectl-cost` and is somewhere in your `PATH`, it will be usable.

## Usage

Open up a terminal and run `kubectl cost`. You can expect it to read your config like `kubectl` does and it offers some standard `kubectl` options like `kubeconfig`, `namespace`, `client-key`, etc. that behave as usual. Options that are specific to the tool and not accessing your cluster are as follows:

```
--window string               the window of data to query (default "yesterday")
--cost-namespace string       filter results to only include a specific namespace, leave blank for all namespaces
```

Run `kubectl cost --help` for usage examples and the full set of flags.
