// setup2
//
// A *rest.Config is just credentials + an address. To actually talk to the
// API you wrap it in a clientset: one typed client per API group/version
// (CoreV1(), AppsV1(), ...). Two classic mistakes are made on this line:
// passing the wrong config, and throwing the error away.
//
// Create the clientset from the loaded config, handle the error, and list
// the cluster's nodes.
package main

import (
	"context"
	"fmt"
	"os"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
)

func main() {
	rules := clientcmd.NewDefaultClientConfigLoadingRules()
	overrides := &clientcmd.ConfigOverrides{CurrentContext: "kind-clientlings"}
	config, err := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(rules, overrides).ClientConfig()
	if err != nil {
		fmt.Printf("❌ could not load kubeconfig: %v\n", err)
		os.Exit(1)
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		fmt.Printf("❌ could not create clientset: %v\n", err)
		os.Exit(1)
	}

	nodes, err := clientset.CoreV1().Nodes().List(context.Background(), metav1.ListOptions{})
	if err != nil {
		fmt.Printf("❌ could not list nodes: %v\n", err)
		os.Exit(1)
	}

	if len(nodes.Items) != 3 {
		fmt.Printf("❌ expected the 3 kind nodes, got %d — are you talking to the right cluster?\n", len(nodes.Items))
		os.Exit(1)
	}

	for _, node := range nodes.Items {
		fmt.Printf("✓ node: %s\n", node.Name)
	}
	fmt.Println("\n🎉 clientset works — you just made your first API call with client-go")
}
