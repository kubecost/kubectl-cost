# kubectl-cost

`kubectl-cost` is a [kubectl plugin](https://kubernetes.io/docs/tasks/extend-kubectl/kubectl-plugins/) that provides easy CLI access to Kubernetes cost allocation metrics via the [kubecost APIs](https://github.com/kubecost/docs/blob/master/apis.md). It allows developers, devops, and others to quickly determine the cost & efficiency for any Kubernetes workload.

<img src="assets/regular.gif" alt="Standard CLI Usage" width="600">

<img src="assets/tui.gif" alt="TUI Usage" width="600">

## Installation

1. [Install Kubecost](https://docs.kubecost.com/install)

    This software requires that you have a running deployment of [Kubecost](https://kubecost.com/) in your cluster. 

2. Install kubectl cost

    #### Krew
    If you have [Krew](https://krew.sigs.k8s.io/), the `kubectl` plugin manager, installed:
    ``` sh
    kubectl krew install cost
    ```

    The manifest can be found [here](https://github.com/kubernetes-sigs/krew-index/blob/master/plugins/cost.yaml)

    #### Linux/MacOS

    ``` sh
    os=$(uname | tr '[:upper:]' '[:lower:]') && \
    arch=$(uname -m | tr '[:upper:]' '[:lower:]' | sed -e s/x86_64/amd64/) && \
    curl -s -L https://github.com/kubecost/kubectl-cost/releases/latest/download/kubectl-cost-$os-$arch.tar.gz | tar xz -C /tmp && \
    chmod +x /tmp/kubectl-cost && \
    sudo mv /tmp/kubectl-cost /usr/local/bin/kubectl-cost
    ```

    #### Latest Release

    If you prefer to download from GitHub, or are on Windows, go to the [releases page](https://github.com/kubecost/kubectl-cost/releases) and download the appropriate binary for your system. Rename it to `kubectl-cost` and put it in your `PATH`.

    As long as the binary is still named `kubectl-cost` and is somewhere in your `PATH`, it will be usable. This is because `kubectl` automatically finds plugins by looking for executables prefixed with `kubectl-` in your `PATH`.

   Alternatively, you can view [DEVELOPMENT.md](DEVELOPMENT.md) if you would like to build from source.

## Usage

There are several supported subcommands: `namespace`, `deployment`, `controller`, `label`, `pod`, and `tui`, which display cost information aggregated by the name of the subcommand (see Examples). Each subcommand has two primary modes, rate and non-rate. Rate (the default) displays the projected monthly cost based on the activity during the window. Non-rate (`--historical`) displays the total cost for the duration of the window.

The exception to these descriptions is `kubectl cost tui`, which displays a TUI and is currently limited to only monthly rate projections. It currently supports all of the previously mentioned aggregations except label. These limitations are because the TUI is an experimental feature - if you like it, let us know! We'd be happy to dedicate time to expanding its functionality.


#### Examples
Show the projected monthly rate for each namespace
with all cost components displayed.
``` sh
kubectl cost namespace --show-all-resources
```
Here is sample output:
```
+-------------------+-----------+----------+----------+-------------+----------+----------+----------+-------------+--------------------+
| NAMESPACE         | CPU       | CPU EFF. | MEMORY   | MEMORY EFF. | GPU      | PV       | NETWORK  | SHARED COST | MONTHLY RATE (ALL) |
+-------------------+-----------+----------+----------+-------------+----------+----------+----------+-------------+--------------------+
| kube-system       | 29.366083 | 0.066780 | 5.226317 | 0.928257    | 0.000000 | 0.000000 | 0.000000 | 137.142857  |         171.735257 |
| kubecost-stage    | 6.602761  | 0.158069 | 1.824703 | 1.594699    | 0.000000 | 2.569600 | 0.000000 | 137.142857  |         148.139922 |
| kubecost          | 6.499445  | 0.116629 | 1.442334 | 1.461370    | 0.000000 | 2.569600 | 0.000000 | 137.142857  |         147.654236 |
| default           | 3.929377  | 0.000457 | 0.237937 | 0.283941    | 0.000000 | 0.000000 | 0.000000 | 137.142857  |         141.310171 |
| logging           | 0.770976  | 0.003419 | 0.645843 | 0.260154    | 0.000000 | 0.000000 | 0.000000 | 137.142857  |         138.559676 |
| frontend-services | 0.710425  | 0.003660 | 0.595008 | 0.244802    | 0.000000 | 0.000000 | 0.000000 | 137.142857  |         138.448290 |
| data-science      | 0.000284  | 2.000000 | 0.009500 | 2.000000    | 0.000000 | 0.000000 | 0.000000 | 137.142857  |         137.152641 |
+-------------------+-----------+----------+----------+-------------+----------+----------+----------+-------------+--------------------+
| SUMMED            | 47.879350 |          | 9.981644 |             | 0.000000 | 5.139200 | 0.000000 | 960.000000  |        1023.000194 |
+-------------------+-----------+----------+----------+-------------+----------+----------+----------+-------------+--------------------+
```

Show how much each namespace cost over the past 5 days
with additional CPU and memory cost and without efficiency.
``` sh
kubectl cost namespace \
  --historical \
  --window 5d \
  --show-cpu \
  --show-memory \
  --show-efficiency=false
```

Show the projected monthly rate for each controller
based on the last 5 days of activity with PV (persistent
volume) cost breakdown.
``` sh
kubectl cost controller --window 5d --show-pv
```

Show costs over the past 5 days broken down by the value
of the `app` label:
``` sh
kubectl cost label --historical -l app
```

Show the projected monthly rate for each deployment
based on the last month of activity with CPU, memory,
GPU, PV, and network cost breakdown.
``` sh
kubectl cost deployment --window month -A
```

Show the projected monthly rate for each deployment
in the `kubecost` namespace based on the last 3 days
of activity with CPU cost breakdown.
``` sh
kubectl cost deployment \
  --window 3d \
  --show-cpu \
  -n kubecost
```

The same, but with a non-standard Kubecost deployment
in the namespace `kubecost-staging` with the cost
analyzer service called `kubecost-staging-cost-analyzer`.
``` sh
kubectl cost deployment \
  --window 3d \
  --show-cpu \
  -n kubecost \
  -N kubecost-staging \
  --service-name kubecost-staging-cost-analyzer
```

Show how much each pod in the "kube-system" namespace
cost yesterday, including CPU-specific cost.
``` sh
kubectl cost pod \
  --historical \
  --window yesterday \
  --show-cpu \
  -n kube-system
```


#### Flags
See `kubectl cost [subcommand] --help` for the full set of flags.

The following flags modify the behavior of the subcommands:
```
    --historical                  show the total cost during the window instead of the projected monthly rate based on the data in the window"
    --show-cpu                    show data for CPU cost
    --show-efficiency             show efficiency of cost alongside CPU and memory cost (default true)
    --show-gpu                    show data for GPU cost
    --show-memory                 show data for memory cost
    --show-network                show data for network cost
    --show-pv                     show data for PV (physical volume) cost
    --show-shared                 show shared cost data
-A, --show-all-resources          Equivalent to --show-cpu --show-memory --show-gpu --show-pv --show-network.
    --window string               The window of data to query. See https://github.com/kubecost/docs/blob/master/allocation.md#querying for a detailed explanation of what can be passed here. (default "yesterday")
    --service-name string         The name of the kubecost cost analyzer service. Change if you're running a non-standard deployment, like the staging helm chart. (default "kubecost-cost-analyzer")
-n, --namespace string            Limit results to only one namespace. Defaults to all namespaces.
-N, --kubecost-namespace string   The namespace that kubecost is deployed in. Requests to the API will be directed to this namespace. (default "kubecost")
```


`kubectl cost` has to interact with the Kubernetes API server. It tries to use your kubeconfig. These flags are common to `kubectl` and allow you to customize this behavior.
``` sh
      --as string                      Username to impersonate for the operation
      --as-group stringArray           Group to impersonate for the operation, this flag can be repeated to specify multiple groups.
      --cache-dir string               Default cache directory (default "/home/delta/.kube/cache")
      --certificate-authority string   Path to a cert file for the certificate authority
      --client-certificate string      Path to a client certificate file for TLS
      --client-key string              Path to a client key file for TLS
      --cluster string                 The name of the kubeconfig cluster to use
      --context string                 The name of the kubeconfig context to use
  -h, --help                           help for cost
      --insecure-skip-tls-verify       If true, the server's certificate will not be checked for validity. This will make your HTTPS connections insecure
      --kubeconfig string              Path to the kubeconfig file to use for CLI requests.
      --request-timeout string         The length of time to wait before giving up on a single server request. Non-zero values should contain a corresponding time unit (e.g. 1s, 2m, 3h). A value of zero means don't timeout requests. (default "0")
  -s, --server string                  The address and port of the Kubernetes API server
      --tls-server-name string         Server name to use for server certificate validation. If it is not provided, the hostname used to contact the server is used
      --token string                   Bearer token for authentication to the API server
      --user string                    The name of the kubeconfig user to use
```

## Implementation Quirks

In order to provide a seamless experience for standard Kubernetes configurations, `kubectl-cost` talks to the Kubernetes API server based on your Kubeconfig and uses the API server to proxy a request to the Kubecost API. If you get an error like `failed to proxy get kubecost`, there is something going wrong with this behavior.

- There may be an underlying problem with your Kubecost install, try `kubectl port-forward`ing the `kubecost-cost-analyzer` service, port 9090, and querying [one of our APIs](https://github.com/kubecost/docs/blob/master/apis.md).
- Your problem could be a security configuration that is preventing the API server communicating with certain namespaces or proxying requests in general.
- If you're still having problems, hit us up on Slack (see below) or open an issue on this repo.

## Requirements
A cluster running Kubernetes version 1.8 or higher

Have questions? Join our [Slack community](https://join.slack.com/t/kubecost/shared_invite/enQtNTA2MjQ1NDUyODE5LWFjYzIzNWE4MDkzMmUyZGU4NjkwMzMyMjIyM2E0NGNmYjExZjBiNjk1YzY5ZDI0ZTNhZDg4NjlkMGRkYzFlZTU) or contact us via email at [team@kubecost.com](team@kubecost.com)!
