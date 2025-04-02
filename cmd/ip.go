package cmd

import (
    "context"
    "fmt"
    "os"
    "path/filepath"

    "github.com/spf13/cobra"
    metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
    "k8s.io/client-go/kubernetes"
    "k8s.io/client-go/tools/clientcmd"
)

// Pod bilgilerini tutacak yapı
type PodInfo struct {
    Name     string
    IP       string
    NodeName string
    NodeIP   string
}

// ip komutu tanımı
var ipCmd = &cobra.Command{
    Use:   "ip",
    Short: "List pod IPs and node information",
    Long:  `List all pods in the default namespace with their IPs and node information.`,
    Run: func(cmd *cobra.Command, args []string) {
        fmt.Println("Connecting to Kubernetes cluster...")
        podInfos, err := getPodAndNodeInfo()
        if err != nil {
            fmt.Printf("Error connecting to Kubernetes cluster: %v\n", err)
            os.Exit(1)
        }
        printPodInfo(podInfos)
    },
}

func getPodAndNodeInfo() ([]PodInfo, error) {
    kubeconfig := os.Getenv("KUBECONFIG")
    if kubeconfig == "" {
        homeDir, err := os.UserHomeDir()
        if err != nil {
            return nil, fmt.Errorf("error getting home directory: %v", err)
        }
        kubeconfig = filepath.Join(homeDir, ".kube", "config")
    }

    config, err := clientcmd.BuildConfigFromFlags("", kubeconfig)
    if err != nil {
        return nil, fmt.Errorf("error building kubeconfig: %v", err)
    }

    clientset, err := kubernetes.NewForConfig(config)
    if err != nil {
        return nil, fmt.Errorf("error creating kubernetes client: %v", err)
    }

    pods, err := clientset.CoreV1().Pods("default").List(context.TODO(), metav1.ListOptions{})
    if err != nil {
        return nil, fmt.Errorf("error listing pods: %v", err)
    }

    nodes, err := clientset.CoreV1().Nodes().List(context.TODO(), metav1.ListOptions{})
    if err != nil {
        return nil, fmt.Errorf("error listing nodes: %v", err)
    }

    nodeIPMap := make(map[string]string)
    for _, node := range nodes.Items {
        for _, addr := range node.Status.Addresses {
            if addr.Type == "InternalIP" {
                nodeIPMap[node.Name] = addr.Address
                break
            }
        }
    }

    var podInfos []PodInfo
    for _, pod := range pods.Items {
        if pod.Status.PodIP == "" {
            continue
        }

        podInfo := PodInfo{
            Name:     pod.Name,
            IP:       pod.Status.PodIP,
            NodeName: pod.Spec.NodeName,
        }

        if nodeIP, exists := nodeIPMap[pod.Spec.NodeName]; exists {
            podInfo.NodeIP = nodeIP
        }

        podInfos = append(podInfos, podInfo)
    }

    return podInfos, nil
}

// Yazdırma
func printPodInfo(podInfos []PodInfo) {
    fmt.Printf("%-40s %-15s %-30s %-15s\n", "POD NAME", "POD IP", "NODE NAME", "NODE IP")
    fmt.Println("--------------------------------------------------------------------------------")
    for _, info := range podInfos {
        fmt.Printf("%-40s %-15s %-30s %-15s\n",
            info.Name,
            info.IP,
            info.NodeName,
            info.NodeIP)
    }
}

