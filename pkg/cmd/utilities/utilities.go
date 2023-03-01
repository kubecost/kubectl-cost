package utilities

import (
	"fmt"
	"strings"

	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/client-go/rest"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

const (
	EnvPrefix = "KUBECTL_COST"
)

// KubeOptions provides information required to communicate
// with the Kubernetes API
type KubeOptions struct {
	configFlags *genericclioptions.ConfigFlags

	RestConfig *rest.Config
	args       []string

	// Namespace should be the currently-configured defaultNamespace of the client.
	// This allows e.g. predict to fill in the defaultNamespace if one is not provided
	// in the workload spec.
	DefaultNamespace string

	genericclioptions.IOStreams
}

// NewCommonCostOptions creates the default set of cost options
func NewKubeOptions(streams genericclioptions.IOStreams) *KubeOptions {
	return &KubeOptions{
		configFlags: genericclioptions.NewConfigFlags(true),

		IOStreams: streams,
	}
}

// Complete sets all information required for getting cost information
func (o *KubeOptions) Complete(cmd *cobra.Command, args []string) error {
	o.args = args

	var err error

	o.RestConfig, err = o.configFlags.ToRESTConfig()
	if err != nil {
		return fmt.Errorf("converting to REST config: %s", err)
	}

	o.DefaultNamespace, _, err = o.configFlags.ToRawKubeConfigLoader().Namespace()
	if err != nil {
		return fmt.Errorf("retrieving default namespace: %s", err)
	}

	return nil
}

// Validate ensures that all required arguments and flag values are provided
func (o *KubeOptions) Validate() error {

	return nil
}

// AddKubeOptionsFlags sets up the cobra command with the flags from
// KubeOptions' configFlags so that a kube client can be built to a
// user's specification. Its one modification is to change the name
// of the namespace flag to kubecost-namespace because we want to
// "behave as expected", i.e. --namespace affects the request to the
// kubecost API, not the request to the k8s API.
func AddKubeOptionsFlags(cmd *cobra.Command, ko *KubeOptions) {
	// By setting Namespace to nil, AddFlags won't create
	// the --namespace flag, which we want to use for scoping
	// kubecost requests (for some subcommands). We can then
	// create a differently-named flag for the same variable.
	ko.configFlags.Namespace = nil
	ko.configFlags.AddFlags(cmd.Flags())

	// Reset Namespace to a valid string to avoid a nil pointer
	// deref.
	// emptyStr := ""
	// ko.configFlags.Namespace = &emptyStr
}

// Binds the flag with viper environment variable and ensures the order of precendence
// command line > environment variable > default value
func BindAFlagToViperEnv(cmd *cobra.Command, v *viper.Viper, flag string) {
	flagPtr := cmd.Flags().Lookup(flag)
	envVarSuffix := strings.ToUpper(strings.ReplaceAll(flagPtr.Name, "-", "_"))
	v.BindEnv(flagPtr.Name, fmt.Sprintf("%s_%s", EnvPrefix, envVarSuffix))
	if !flagPtr.Changed && v.IsSet(flagPtr.Name) {
		val := v.Get(flagPtr.Name)
		cmd.Flags().Set(flagPtr.Name, fmt.Sprintf("%v", val))
	}
}
