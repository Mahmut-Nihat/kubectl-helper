package cmd

import (
	"fmt"
	"strings"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/cli-runtime/pkg/resource"
)

// PodInfo holds the essential Pod data we want to display.
type PodInfo struct {
	Name      string
	Namespace string
	IP        string
	NodeName  string
	NodeIP    string
}

// namespaceFlag holds the namespace requested by the user via -n/--namespace
var namespaceFlag string

// configFlags is used to handle kubeconfig-based flags.
var configFlags = genericclioptions.NewConfigFlags(true)

// ipCmd is the main Cobra command for listing Pods by partial name match.
var ipCmd = &cobra.Command{
	Use:   "ip [SEARCH_PATTERN]",
	Short: "List pods containing [SEARCH_PATTERN] in their name, along with IP and node info.",
	// We bind our custom runFunc for command execution.
	RunE: runFunc(configFlags),

	// Add flags to the
}

// Add the namespace flag to ipCmd right here.
func init() {
	// This registers the -n/--namespace flag with our ipCmd.
	ipCmd.Flags().StringVarP(&namespaceFlag, "namespace", "n", "",
		"Namespace to filter pods. Searches all namespaces if omitted.")
}

// runFunc returns a function that searches for pods (in one or all namespaces)
// and filters them by the provided SEARCH_PATTERN.
func runFunc(configFlags *genericclioptions.ConfigFlags) func(cmd *cobra.Command, args []string) error {
	return func(cmd *cobra.Command, args []string) error {
		if len(args) < 1 {
			return fmt.Errorf("please provide a search pattern, for example:\n  ./api-deneme ip nginx\nor:\n  ./api-deneme ip -n dev nginx")
		}
		searchTerm := args[0]

		// Retrieve the namespace from kubeconfig (for informational printing only)
		// clientCfg := configFlags.ToRawKubeConfigLoader()
		// kubeconfigNamespace, _, err := clientCfg.Namespace()
		// if err != nil {
		// 	return fmt.Errorf("failed to determine namespace from kubeconfig: %w", err)
		// }

		// Decide if we use the namespaceFlag or all namespaces
		var rb *resource.Builder
		if namespaceFlag != "" {
			rb = resource.NewBuilder(configFlags).
				Unstructured().
				ResourceTypeOrNameArgs(true, "pods").
				NamespaceParam(namespaceFlag). // specific namespace
				ContinueOnError().
				Flatten()
		} else {
			rb = resource.NewBuilder(configFlags).
				Unstructured().
				ResourceTypeOrNameArgs(true, "pods").
				AllNamespaces(true). // all namespaces
				ContinueOnError().
				Flatten()
		}

		var matchingPods []PodInfo

		err := rb.Do().Visit(func(info *resource.Info, visitErr error) error {
			if visitErr != nil {
				return visitErr
			}
			podInfo, convertErr := convertObjectToPodInfo(info.Object)
			if convertErr != nil {
				// Skip objects we can't convert
				return nil
			}
			// If the pod name contains the search term, add it to the list.
			if strings.Contains(strings.ToLower(podInfo.Name), strings.ToLower(searchTerm)) {
				matchingPods = append(matchingPods, podInfo)
			}
			return nil
		})
		if err != nil {
			return fmt.Errorf("failed to retrieve pods: %w", err)
		}

		if len(matchingPods) == 0 {
			fmt.Printf("No pods found matching the pattern: %s\n", searchTerm)
			return nil
		}

		printColoredTable(matchingPods)
		return nil
	}
}

// convertObjectToPodInfo attempts to convert the provided runtime.Object to PodInfo.
func convertObjectToPodInfo(obj runtime.Object) (PodInfo, error) {
	// Convert to unstructured if needed.
	unstructuredObj, ok := obj.(*unstructured.Unstructured)
	if !ok {
		objMap, err := runtime.DefaultUnstructuredConverter.ToUnstructured(obj)
		if err != nil {
			return PodInfo{}, fmt.Errorf("failed to convert object to unstructured: %w", err)
		}
		unstructuredObj = &unstructured.Unstructured{Object: objMap}
	}

	// Safely extract fields from the unstructured object.
	spec, specOK := unstructuredObj.Object["spec"].(map[string]interface{})
	status, statusOK := unstructuredObj.Object["status"].(map[string]interface{})
	if !specOK || !statusOK {
		return PodInfo{}, fmt.Errorf("object does not contain 'spec' or 'status' in expected format")
	}

	podName := unstructuredObj.GetName()

	podNamespace := unstructuredObj.GetNamespace()

	// Pod IP
	podIP, _ := status["podIP"].(string)

	// Node Name
	nodeNameRaw := spec["nodeName"]
	nodeName := nodeNameRaw.(string)

	// Node IP
	hostIPRaw := status["hostIP"]
	hostIP := hostIPRaw.(string)

	return PodInfo{
		Name:      podName,
		Namespace: podNamespace,
		IP:        podIP,
		NodeName:  nodeName,
		NodeIP:    hostIP,
	}, nil
}

// printColoredTable prints the table of matching pods using color for headers and lines.
func printColoredTable(pods []PodInfo) {
	// Prepare colored objects from github.com/fatih/color
	headerColor := color.New(color.FgCyan, color.Bold)
	lineColor := color.New(color.FgCyan)

	// Print the header line.
	fmt.Println()
	// Print the header with colors.
	headerColor.Printf("%-30s %-20s %-20s %-30s %-20s\n", "NAME", "NAMESPACE", "POD IP", "NODE NAME", "NODE IP")

	// Print a separator line in color.
	line := strings.Repeat("-", 120)
	lineColor.Println(line)

	// Print each pod line in default color (you could also choose different colors if you want).
	for _, p := range pods {
		fmt.Printf("%-30s %-20s %-20s %-30s %-20s\n", p.Name, p.Namespace, p.IP, p.NodeName, p.NodeIP)
	}
	fmt.Println()
}
