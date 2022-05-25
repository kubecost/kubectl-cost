package query

// QueryBackendOptions holds common options for managing the query backend used
// by kubectl-cost, like service name, namespace, etc.
type QueryBackendOptions struct {
	// If set, will proxy a request through the K8s API server
	// instead of port forwarding.
	UseProxy bool

	// The name of the cost-analyzer service in the cluster,
	// in case user is running a non-standard name (like the
	// staging helm chart). Combines with
	// commonOptions.configFlags.Namespace to direct the API
	// request.
	ServiceName string

	// The namespace in which Kubecost is running
	KubecostNamespace string
}
