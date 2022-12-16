# kubectl-cost

`kubectl-cost` is a [kubectl plugin](https://kubernetes.io/docs/tasks/extend-kubectl/kubectl-plugins/) that provides easy CLI access to Kubernetes cost allocation metrics via the [kubecost APIs](https://github.com/kubecost/docs/blob/master/apis.md). It allows developers, devops, and others to quickly determine the cost & efficiency for any Kubernetes workload.

> If you use [OpenCost](https://github.com/opencost/opencost), most of `kubectl cost` works! See [OpenCost documentation](https://www.opencost.io/docs/kubectl-cost) for examples. Let us know how it goes, and open an issue if you encounter any problems!

<img src="assets/regular.gif" alt="Standard CLI Usage" width="600">

<img src="assets/tui.gif" alt="TUI Usage" width="600">

## Installation

1. Install Kubecost

    This software requires that you have a running deployment of [Kubecost](https://kubecost.com/) in your cluster. The recommend path is to use Helm but there are [alternative install options](https://docs.kubecost.com/install).

    #### Helm 3

    ```
    helm repo add kubecost https://kubecost.github.io/cost-analyzer/
    helm upgrade -i --create-namespace kubecost kubecost/cost-analyzer --namespace kubecost --set kubecostToken="a3ViZWN0bEBrdWJlY29zdC5jb20=xm343yadf98"
    ```

2. Install kubectl cost

    #### Krew
    If you have [Krew](https://krew.sigs.k8s.io/), the `kubectl` plugin manager, installed:
    ``` sh
    kubectl krew install cost
    ```

    The Krew manifest can be found [here](https://github.com/kubernetes-sigs/krew-index/blob/master/plugins/cost.yaml).

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

There are several supported subcommands: `namespace`, `deployment`, `controller`, `label`, `pod`, `node`, and `tui`, which display cost information aggregated by the name of the subcommand (see Examples). Each subcommand has two primary modes, rate and non-rate. Rate (the default) displays the projected monthly cost based on the activity during the window. Non-rate (`--historical`) displays the total cost for the duration of the window.

The exception to these descriptions is `kubectl cost tui`, which displays a TUI and is currently limited to only monthly rate projections. It currently supports all of the previously mentioned aggregations except label. These limitations are because the TUI is an experimental feature - if you like it, let us know! We'd be happy to dedicate time to expanding its functionality.


#### Examples
Show the projected monthly rate for each namespace
with all cost components displayed.
``` sh
kubectl cost namespace --show-all-resources
```
Example output:
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

Predict the cost of a YAML spec based on its requests:
``` sh
read -r -d '' DEF << EndOfMessage
apiVersion: apps/v1
kind: Deployment
metadata:
  name: nginx-deployment
  labels:
    app: nginx
spec:
  replicas: 3
  selector:
    matchLabels:
      app: nginx
  template:
    metadata:
      labels:
        app: nginx
    spec:
      containers:
      - name: nginx
        image: nginx:1.14.2
        resources:
          requests:
            cpu: "3"
            memory: "2Gi"
EndOfMessage
echo "$DEF" | kubectl cost predict -f -
```
Example output:
```
+-----------------------------+-----+-----+------------+-----------+------------+
| WORKLOAD                    | CPU | MEM | CPU/MO     | MEM/MO    | TOTAL/MO   |
+-----------------------------+-----+-----+------------+-----------+------------+
| Deployment/nginx-deployment | 9   | 6Gi | 209.51 USD | 18.73 USD | 228.24 USD |
+-----------------------------+-----+-----+------------+-----------+------------+
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

Predict the cost of the Deployment defined in k8s-deployment.yaml.
``` sh
kubectl cost predict -f 'k8s-deployment.yaml' \
  --show-cost-per-resource-hr
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

Alternatively, kubectl cost can show cost by the asset type.
To view node cost with breakdowns of RAM and CPU cost for a 
window of 7 days.
``` sh
kubectl cost node \
  --historical \
  --window 7d \
  --show-cpu \
  --show-memory
```

Which yields an output with this format:
```
+-------------+---------------------------------------------+---------------+--------------+---------------+
| CLUSTER     | NAME                                        | CPU COST      | RAM COST     | TOTAL COST    |
+-------------+---------------------------------------------+---------------+--------------+---------------+
| cluster-one | gke-test-cluster-default-pool-d6266c7c-dqms |      4.128570 |     2.128920 |      6.257491 |
|             | gke-test-cluster-pool-1-9bb98ef8-3w6g       |      4.128570 |     2.128920 |      6.257491 |
|             | gke-test-cluster-pool-1-9bb98ef8-cf3j       |      4.128570 |     2.128924 |      6.257495 |
|             | gke-test-cluster-pool-1-9bb98ef8-kdsf       |      4.128570 |     2.128924 |      6.257495 |
+-------------+---------------------------------------------+---------------+--------------+---------------+
| SUMMED      |                                             | USD 16.514280 | USD 8.515688 | USD 25.029972 |
+-------------+---------------------------------------------+---------------+--------------+---------------+
```

#### Flags
See `kubectl cost [subcommand] --help` for the full set of flags. Each
subcommand has its own set of flags for adjusting query behavior and output.

There are several flags that modify the behavior of queries to the backing
Kubecost/OpenCost APIs:
```
    -r, --release-name string                 The name of the Helm release, used to template service names if they are unset. For example, if Kubecost is installed with 'helm install kubecost2 kubecost/cost-analyzer', then this should be set to 'kubecost2'. (default "kubecost")
    --service-name string                 The name of the Kubecost cost analyzer service. By default, it is derived from the Helm release name and should not need to be overridden.
    --service-port int               The port of the service at which the APIs are running. If using OpenCost, you may want to set this to 9003. (default 9090)
    -N, --kubecost-namespace string           The namespace that Kubecost is deployed in. Requests to the API will be directed to this namespace. Defaults to the Helm release name.

    --use-proxy                   Instead of temporarily port-forwarding, proxy a request to Kubecost through the Kubernetes API server.

    --allocation-path string         URL path at which Allocation queries can be served from the configured service. If using OpenCost, you may want to set this to '/allocation/compute' (default "/model/allocation")
    --predict-resource-cost-path string   URL path at which Resource Cost Prediction queries can be served from the configured service. (default "/model/prediction/resourcecost")
```


`kubectl cost` has to interact with the Kubernetes API server. It tries to use your existing kubeconfig. These flags are common to `kubectl` and allow you to customize this behavior:
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

## If something breaks

`kubectl cost` logs some of its behavior at the `debug` log level. If something isn't working as you'd expect, try setting `--log-level debug` before opening a bug report.

## Implementation Quirks

In order to provide a seamless experience for standard Kubernetes
configurations, `kubectl-cost` temporarily forwards a port on your system to a
Kubecost pod and uses that port to proxy requests. The port will only be bound
to `localhost` and will only be open for the duration of the `kubectl cost` run.
Due to Linux default conventions, the port may appear as held for a little while
after the run (see TCP's `TIME_WAIT`).

If you don't want a port to be temporarily forwarded, there is legacy behavior
exposed with the flag `--use-proxy` or using environment
`KUBECTL_COST_USE_PROXY` that will instead use the Kubernetes API server to
proxy requests to Kubecost. This behavior has its own pitfalls, especially with
security policies that would prevent the API server from communicating with
services. If you'd like to test this behavior, to make sure it will work with
your cluster:

``` sh
kubectl proxy --port 8080
```

``` sh
curl -G 'http://localhost:8080/api/v1/namespaces/kubecost/services/kubecost-cost-analyzer:tcp-model/proxy/getConfigs'
```

> If you are running an old version of Kubecost, you may have to replace `tcp-model` with `model`

If that `curl` succeeds, `--use-proxy` flag in CLI or setting up environment variable `KUBECTL_COST_USE_PROXY` should work for you.

Otherwise:
- There may be an underlying problem with your Kubecost install, try `kubectl port-forward`ing the `kubecost-cost-analyzer` service, port 9090, and querying [one of our APIs](https://github.com/kubecost/docs/blob/master/apis.md).
- Your problem could be a security configuration that is preventing the API server communicating with certain namespaces or proxying requests in general.
- If you're still having problems, hit us up on Slack (see below) or open an issue on this repo.

## Requirements
A cluster running Kubernetes version 1.8 or higher

Have questions? Join our [Slack community](https://join.slack.com/t/kubecost/shared_invite/enQtNTA2MjQ1NDUyODE5LWFjYzIzNWE4MDkzMmUyZGU4NjkwMzMyMjIyM2E0NGNmYjExZjBiNjk1YzY5ZDI0ZTNhZDg4NjlkMGRkYzFlZTU) or contact us via email at [team@kubecost.com](team@kubecost.com)!
