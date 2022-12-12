package cmd

import (
	"context"
	"errors"
	"fmt"
	"io/ioutil"
	"os"

	"github.com/kubecost/kubectl-cost/pkg/query"

	"github.com/kubecost/opencost/pkg/log"

	"github.com/spf13/cobra"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"

	// yamlutil "k8s.io/apimachinery/pkg/util/yaml"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/client-go/kubernetes/scheme"
)

// PredictOptions contains options specific to prediction queries.
type PredictOptions struct {
	// TODO: window

	// TODO: idle/no idle

	clusterID string

	// The file containing the workload definition to be predicted.
	filepath string

	showCostPerResourceHr bool

	query.QueryBackendOptions
}

func newCmdPredict(
	streams genericclioptions.IOStreams,
) *cobra.Command {
	kubeO := NewKubeOptions(streams)
	predictO := &PredictOptions{}

	cmd := &cobra.Command{
		Use:   "predict",
		Short: "Estimate the monthly cost of a workload based on tracked cluster resource costs.",
		RunE: func(c *cobra.Command, args []string) error {
			if err := kubeO.Complete(c, args); err != nil {
				return err
			}
			if err := kubeO.Validate(); err != nil {
				return err
			}

			if err := predictO.Validate(); err != nil {
				return err
			}

			return runCostPredict(kubeO, predictO)
		},
	}
	cmd.Flags().StringVarP(&predictO.filepath, "filepath", "f", "", "The file containing the workload definition whose cost should be predicted. E.g. a file might be 'test-deployment.yaml' containing an apps/v1 Deployment definition.")
	cmd.Flags().StringVarP(&predictO.clusterID, "cluster-id", "c", "", "The cluster ID (in Kubecost) of the presumed cluster which the workload will be deployed to. This is used to determine resource costs. Defaults to all clusters.")
	cmd.Flags().BoolVar(&predictO.showCostPerResourceHr, "show-cost-per-resource-hr", false, "Show the calculated cost per resource-hr (e.g. $/byte-hour) used for the cost prediction.")

	addQueryBackendOptionsFlags(cmd, &predictO.QueryBackendOptions)
	addKubeOptionsFlags(cmd, kubeO)

	return cmd
}

func (predictO *PredictOptions) Validate() error {
	if _, err := os.Stat(predictO.filepath); errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("file '%s' does not exist, not a valid option", predictO.filepath)
	}
	return nil
}

func sumContainerResources(replicas int, spec v1.PodSpec) v1.ResourceList {
	podMemory := resource.NewQuantity(0, resource.BinarySI)
	podCPU := resource.NewMilliQuantity(0, resource.DecimalSI)

	for _, cntr := range spec.Containers {
		requests := cntr.Resources.Requests
		if ram, ok := requests[corev1.ResourceMemory]; ok {
			podMemory.Add(ram)
		}
		if cpu, ok := requests[corev1.ResourceCPU]; ok {
			podCPU.Add(cpu)
		}
	}

	totalMemory := resource.NewQuantity(0, resource.BinarySI)
	totalCPU := resource.NewMilliQuantity(0, resource.DecimalSI)
	for i := 0; i < replicas; i++ {
		totalMemory.Add(*podMemory)
		totalCPU.Add(*podCPU)
	}

	return v1.ResourceList{
		v1.ResourceCPU:    *totalCPU,
		v1.ResourceMemory: *totalMemory,
	}
}

func runCostPredict(ko *KubeOptions, no *PredictOptions) error {
	b, err := ioutil.ReadFile(no.filepath)
	if err != nil {
		return fmt.Errorf("failed to read file '%s': %s", no.filepath, err)
	}

	// https://github.com/kubernetes/client-go/issues/193#issuecomment-343138889
	// https://github.com/kubernetes/client-go/issues/193#issuecomment-377140518
	decode := scheme.Codecs.UniversalDeserializer().Decode
	obj, _, err := decode(b, nil, nil)
	if err != nil {
		return fmt.Errorf("decoding from file: %s", err)
	}

	var totalResources v1.ResourceList
	var name string
	switch typed := obj.(type) {
	case *appsv1.Deployment:
		replicas := 1
		if typed.Spec.Replicas == nil {
			log.Warnf("replicas is nil, defaulting to 1")
		} else {
			replicas = int(*typed.Spec.Replicas)
		}
		name = typed.Name
		totalResources = sumContainerResources(replicas, typed.Spec.Template.Spec)
	case *appsv1.StatefulSet:
		replicas := 1
		if typed.Spec.Replicas == nil {
			log.Warnf("replicas is nil, defaulting to 1")
		} else {
			replicas = int(*typed.Spec.Replicas)
		}
		name = typed.Name
		totalResources = sumContainerResources(replicas, typed.Spec.Template.Spec)
	case *v1.Pod:
		name = typed.Name
		totalResources = sumContainerResources(1, typed.Spec)
	case *appsv1.DaemonSet:
		name = typed.Name
		return fmt.Errorf("DaemonSets are not supported because scheduling-dependent workloads are not yet supported")
	default:
		return fmt.Errorf("unsupported type: %T", obj)
	}

	currencyCode, err := query.QueryCurrencyCode(query.CurrencyCodeParameters{
		RestConfig:          ko.restConfig,
		Ctx:                 context.Background(),
		QueryBackendOptions: no.QueryBackendOptions,
	})
	if err != nil {
		log.Debugf("failed to get currency code, displaying as empty string: %s", err)
		currencyCode = ""
	}

	memStr := "0"
	cpuStr := "0"
	if mem, ok := totalResources[v1.ResourceMemory]; ok {
		ptr := &mem
		memStr = ptr.String()
		log.Debugf("mem asapprox: %f", ptr.AsApproximateFloat64())
	}
	if cpu, ok := totalResources[v1.ResourceCPU]; ok {
		ptr := &cpu
		cpuStr = ptr.String()
	}
	log.Debugf("mem: '%s', cpu: '%s'", memStr, cpuStr)
	prediction, err := query.QueryPredictResourceCost(query.ResourcePredictParameters{
		RestConfig:          ko.restConfig,
		Ctx:                 context.Background(),
		QueryBackendOptions: no.QueryBackendOptions,
		QueryParams: map[string]string{
			"window":          "2d", // TODO: flag
			"clusterID":       no.clusterID,
			"requestedMemory": memStr,
			"requestedCPU":    cpuStr,
		},
	})
	if err != nil {
		return fmt.Errorf("prediction query failed: %s", err)
	}

	writePredictionTable(ko.Out, name, currencyCode, cpuStr, memStr, prediction, no.showCostPerResourceHr)
	return nil
}
