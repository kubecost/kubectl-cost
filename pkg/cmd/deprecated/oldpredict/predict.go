package oldpredict

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"strings"

	"github.com/kubecost/kubectl-cost/pkg/cmd/utilities"
	"github.com/kubecost/kubectl-cost/pkg/query"

	"github.com/opencost/opencost/pkg/log"

	"github.com/spf13/cobra"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/apimachinery/pkg/runtime"
	yamlutil "k8s.io/apimachinery/pkg/util/yaml"
	"k8s.io/client-go/rest"

	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/client-go/kubernetes/scheme"
)

const (
	resourceGPUKey = "gpu"
	hoursPerMonth  = 730
)

// PredictOptions contains options specific to prediction queries.
type PredictOptions struct {
	window string

	clusterID string

	// The file containing the workload definition to be predicted.
	filepath string

	showCostPerResourceHr bool
	noDiff                bool

	query.QueryBackendOptions
}

func NewCmdOldPredict(
	streams genericclioptions.IOStreams,
) *cobra.Command {
	kubeO := utilities.NewKubeOptions(streams)
	predictO := &PredictOptions{}

	cmd := &cobra.Command{
		Use:   "oldpredict",
		Short: "Estimate the monthly cost of a workload based on tracked cluster resource costs.",
		RunE: func(c *cobra.Command, args []string) error {
			if err := kubeO.Complete(c, args); err != nil {
				return fmt.Errorf("complete k8s options: %s", err)
			}
			if err := kubeO.Validate(); err != nil {
				return fmt.Errorf("validate k8s options: %s", err)
			}

			if err := predictO.Complete(kubeO.RestConfig); err != nil {
				return fmt.Errorf("complete: %s", err)
			}
			if err := predictO.Validate(); err != nil {
				return fmt.Errorf("validate: %s", err)
			}

			return runCostPredict(kubeO, predictO)
		},
	}
	cmd.Flags().StringVarP(&predictO.filepath, "filepath", "f", "", "The file containing the workload definition whose cost should be predicted. E.g. a file might be 'test-deployment.yaml' containing an apps/v1 Deployment definition. '-' can also be passed, in which case workload definitions will be read from stdin.")
	cmd.Flags().StringVarP(&predictO.clusterID, "cluster-id", "c", "", "The cluster ID (in Kubecost) of the presumed cluster which the workload will be deployed to. This is used to determine resource costs. Defaults to local cluster.")
	cmd.Flags().BoolVar(&predictO.showCostPerResourceHr, "show-cost-per-resource-hr", false, "Show the calculated cost per resource-hr (e.g. $/byte-hour) used for the cost prediction.")
	cmd.Flags().StringVar(&predictO.window, "window", "2d", "The window of cost data to base resource costs on. See https://github.com/kubecost/docs/blob/master/allocation.md#querying for a detailed explanation of what can be passed here.")
	cmd.Flags().BoolVar(&predictO.noDiff, "no-diff", false, "Set true to not attempt a cost difference with a matching in-cluster workload, if one can be found.")

	query.AddQueryBackendOptionsFlags(cmd, &predictO.QueryBackendOptions)
	utilities.AddKubeOptionsFlags(cmd, kubeO)

	cmd.SilenceUsage = true

	return cmd
}

func (predictO *PredictOptions) Validate() error {
	if predictO.filepath != "-" {
		if _, err := os.Stat(predictO.filepath); errors.Is(err, os.ErrNotExist) {
			return fmt.Errorf("file '%s' does not exist, not a valid option", predictO.filepath)
		}
	}

	if err := predictO.QueryBackendOptions.Validate(); err != nil {
		return fmt.Errorf("validating query options: %s", err)
	}

	return nil
}

func (predictO *PredictOptions) Complete(restConfig *rest.Config) error {
	if err := predictO.QueryBackendOptions.Complete(restConfig); err != nil {
		return fmt.Errorf("complete backend opts: %s", err)
	}
	return nil
}

func sumContainerResources(replicas int, spec v1.PodSpec) v1.ResourceList {
	podMemory := resource.NewQuantity(0, resource.BinarySI)
	podCPU := resource.NewMilliQuantity(0, resource.DecimalSI)
	podGPU := resource.NewQuantity(0, resource.DecimalSI)

	for _, cntr := range spec.Containers {
		requests := cntr.Resources.Requests
		if ram, ok := requests[corev1.ResourceMemory]; ok {
			podMemory.Add(ram)
		}
		if cpu, ok := requests[corev1.ResourceCPU]; ok {
			podCPU.Add(cpu)
		}

		// GPU is only defined in limits:
		// https://kubernetes.io/docs/tasks/manage-gpus/scheduling-gpus/
		limits := cntr.Resources.Limits
		// https://github.com/RadeonOpenCompute/k8s-device-plugin/blob/master/example/pod/alexnet-gpu.yaml
		if amdGPU, ok := limits["amd.com/gpu"]; ok {

			podGPU.Add(amdGPU)
		}
		// https://github.com/intel/intel-device-plugins-for-kubernetes/blob/1380d24ee9766942f97dcce813b9868565a29218/README.md#L235
		// https://github.com/intel/intel-device-plugins-for-kubernetes/blob/1380d24ee9766942f97dcce813b9868565a29218/demo/intelgpu-job.yaml#L22
		if intelGPU, ok := limits["gpu.intel.com/i915"]; ok {
			podGPU.Add(intelGPU)
		}
		// https://github.com/NVIDIA/k8s-device-plugin#running-gpu-jobs
		if nvidiaGPU, ok := limits["nvidia.com/gpu"]; ok {
			podGPU.Add(nvidiaGPU)
		}
	}

	totalMemory := resource.NewQuantity(0, resource.BinarySI)
	totalCPU := resource.NewMilliQuantity(0, resource.DecimalSI)
	totalGPU := resource.NewQuantity(0, resource.DecimalSI)
	for i := 0; i < replicas; i++ {
		totalMemory.Add(*podMemory)
		totalCPU.Add(*podCPU)
		totalGPU.Add(*podGPU)
	}

	result := v1.ResourceList{
		v1.ResourceCPU:    *totalCPU,
		v1.ResourceMemory: *totalMemory,
	}

	// Only include GPU data if we have any. Put under the generic "gpu"
	// name because we don't (currently) distinguish between providers.
	if !totalGPU.IsZero() {
		result[resourceGPUKey] = *totalGPU
	}

	return result
}

type prePredictionData struct {
	workloadNamespace string
	workloadType      string
	workloadName      string

	totalRequested v1.ResourceList
}

type predictRowData struct {
	workloadNamespace string
	workloadType      string
	workloadName      string

	totalCPURequested    string
	totalMemoryRequested string
	totalGPURequested    string

	requestedCPUCoreHours    float64
	requestedMemoryByteHours float64
	requestedGPUHours        float64

	cpuCostMonthly    float64
	memoryCostMonthly float64
	gpuCostMonthly    float64
	totalCostMonthly  float64

	cpuCostChangeMonthly    float64
	memoryCostChangeMonthly float64
}

func runCostPredict(ko *utilities.KubeOptions, no *PredictOptions) error {
	var b []byte
	var err error

	if no.filepath == "-" {
		reader := bufio.NewReader(ko.In)

		scratch := make([]byte, 1024)
		for {
			n, err := reader.Read(scratch)
			b = append(b, scratch[:n]...)
			if err == io.EOF {
				break
			} else if err != nil {
				return fmt.Errorf("reading from stdin: %s", err)
			}
		}
	} else {
		b, err = ioutil.ReadFile(no.filepath)
		if err != nil {
			return fmt.Errorf("failed to read file '%s': %s", no.filepath, err)
		}
	}

	// This looping decode lets us handle multiple definitions in a single file,
	// as usually separated with '---'
	//
	// https://gist.github.com/pytimer/0ad436972a073bb37b8b6b8b474520fc
	decoder := yamlutil.NewYAMLOrJSONDecoder(bytes.NewReader(b), 100)

	var objs []runtime.Object
	for {
		var rawObj runtime.RawExtension
		if err = decoder.Decode(&rawObj); err != nil {
			if errors.Is(err, io.EOF) {
				break
			}
			return fmt.Errorf("decoding file data as K8s object: %s", err)
		}

		// https://github.com/kubernetes/client-go/issues/193#issuecomment-343138889
		// https://github.com/kubernetes/client-go/issues/193#issuecomment-377140518
		obj, _, err := scheme.Codecs.UniversalDeserializer().Decode(rawObj.Raw, nil, nil)
		if err != nil {
			log.Warnf("decoding: %s", err)
			break
		}

		// Flatten lists
		if l, ok := obj.(*v1.List); ok {
			for _, rawObj := range l.Items {
				obj, _, err := scheme.Codecs.UniversalDeserializer().Decode(rawObj.Raw, nil, nil)
				if err != nil {
					log.Warnf("decoding inside list: %s", err)
					continue
				}

				// don't handle nested lists for now
				objs = append(objs, obj)
			}
			continue
		}
		objs = append(objs, obj)
	}

	var prePred []prePredictionData
	for _, obj := range objs {
		var totalResources v1.ResourceList
		var name string
		var namespace string
		var kind string
		switch typed := obj.(type) {
		case *appsv1.Deployment:
			replicas := 1
			if typed.Spec.Replicas == nil {
				log.Warnf("replicas is nil, defaulting to 1")
			} else {
				replicas = int(*typed.Spec.Replicas)
			}
			name = typed.Name
			kind = "Deployment"
			namespace = typed.Namespace
			totalResources = sumContainerResources(replicas, typed.Spec.Template.Spec)
		case *appsv1.StatefulSet:
			replicas := 1
			if typed.Spec.Replicas == nil {
				log.Warnf("replicas is nil, defaulting to 1")
			} else {
				replicas = int(*typed.Spec.Replicas)
			}
			name = typed.Name
			kind = "StatefulSet"
			namespace = typed.Namespace
			totalResources = sumContainerResources(replicas, typed.Spec.Template.Spec)
		case *v1.Pod:
			name = typed.Name
			kind = "Pod"
			namespace = typed.Namespace
			totalResources = sumContainerResources(1, typed.Spec)
		case *appsv1.DaemonSet:
			name = typed.Name
			kind = "DaemonSet"
			namespace = typed.Namespace
			log.Warnf("DaemonSets are not supported because scheduling-dependent workloads are not yet supported. Skipping %s/%s.", kind, name)
			continue
		default:
			return fmt.Errorf("unsupported type: %T", obj)
		}

		// If the workload doesn't have a namespace specified, try to use the
		// one we retrieved in K8s config
		if namespace == "" {
			namespace = ko.DefaultNamespace
		}

		prePred = append(prePred, prePredictionData{
			workloadNamespace: namespace,
			workloadType:      kind,
			workloadName:      name,
			totalRequested:    totalResources,
		})
	}

	var rows []predictRowData
	for _, pre := range prePred {
		totalResources := pre.totalRequested
		memStr := "0"
		cpuStr := "0"
		gpuStr := "0"
		var cpuCoreHours, memoryByteHours, gpuHours float64
		if mem, ok := totalResources[v1.ResourceMemory]; ok {
			ptr := &mem
			memStr = ptr.String()
			memoryByteHours = mem.AsApproximateFloat64() * hoursPerMonth
		}
		if cpu, ok := totalResources[v1.ResourceCPU]; ok {
			ptr := &cpu
			cpuStr = ptr.String()
			cpuCoreHours = cpu.AsApproximateFloat64() * hoursPerMonth
		}
		if gpu, ok := totalResources[resourceGPUKey]; ok {
			ptr := &gpu
			gpuStr = ptr.String()
			gpuHours = gpu.AsApproximateFloat64() * hoursPerMonth
		}

		if no.noDiff {
			queryParams := map[string]string{
				"window":          no.window,
				"clusterID":       no.clusterID,
				"requestedMemory": memStr,
				"requestedCPU":    cpuStr,
				"requestedGPU":    gpuStr,
			}
			prediction, err := query.QueryPredictResourceCost(query.ResourcePredictParameters{
				RestConfig:          ko.RestConfig,
				Ctx:                 context.Background(),
				QueryBackendOptions: no.QueryBackendOptions,
				QueryParams:         queryParams,
			})
			if err != nil {
				return fmt.Errorf("prediction query failed: %s", err)
			}

			rows = append(rows, predictRowData{
				workloadName:      pre.workloadName,
				workloadType:      pre.workloadType,
				workloadNamespace: pre.workloadNamespace,

				totalCPURequested:    cpuStr,
				totalMemoryRequested: memStr,
				totalGPURequested:    gpuStr,

				requestedCPUCoreHours:    cpuCoreHours,
				requestedMemoryByteHours: memoryByteHours,
				requestedGPUHours:        gpuHours,

				cpuCostMonthly:    prediction.MonthlyCostCPU,
				memoryCostMonthly: prediction.MonthlyCostMemory,
				gpuCostMonthly:    prediction.MonthlyCostGPU,
				totalCostMonthly:  prediction.MonthlyCostTotal,

				cpuCostChangeMonthly:    prediction.MonthlyCostCPU,
				memoryCostChangeMonthly: prediction.MonthlyCostMemory,
			})
		} else {
			queryParams := map[string]string{
				"window":          no.window,
				"clusterID":       no.clusterID,
				"requestedMemory": memStr,
				"requestedCPU":    cpuStr,
				"requestedGPU":    gpuStr,

				"controllerNamespace": pre.workloadNamespace,
				"controllerKind":      strings.ToLower(pre.workloadType),
				"controllerName":      pre.workloadName,
			}
			prediction, err := query.QueryPredictResourceCostDiff(query.ResourceDiffPredictParameters{
				RestConfig:          ko.RestConfig,
				Ctx:                 context.Background(),
				QueryBackendOptions: no.QueryBackendOptions,
				QueryParams:         queryParams,
			})
			if err != nil {
				return fmt.Errorf("prediction query failed: %s", err)
			}

			rows = append(rows, predictRowData{
				workloadName:      pre.workloadName,
				workloadType:      pre.workloadType,
				workloadNamespace: pre.workloadNamespace,

				totalCPURequested:    cpuStr,
				totalMemoryRequested: memStr,
				totalGPURequested:    gpuStr,

				requestedCPUCoreHours:    cpuCoreHours,
				requestedMemoryByteHours: memoryByteHours,
				requestedGPUHours:        gpuHours,

				cpuCostMonthly:    prediction.MonthlyCostCPU,
				memoryCostMonthly: prediction.MonthlyCostMemory,
				gpuCostMonthly:    prediction.MonthlyCostGPU,
				totalCostMonthly:  prediction.MonthlyCostTotal,

				cpuCostChangeMonthly:    prediction.MonthlyCostCPUDiff,
				memoryCostChangeMonthly: prediction.MonthlyCostMemoryDiff,
			})
		}
	}
	currencyCode, err := query.QueryCurrencyCode(query.CurrencyCodeParameters{
		Ctx:                 context.Background(),
		QueryBackendOptions: no.QueryBackendOptions,
	})
	if err != nil {
		log.Debugf("failed to get currency code, displaying as empty string: %s", err)
		currencyCode = ""
	}

	writePredictionTable(ko.Out, rows, PredictionTableOptions{
		CurrencyCode:          currencyCode,
		ShowCostPerResourceHr: no.showCostPerResourceHr,
		NoDiff:                no.noDiff,
	})
	return nil
}
