// setup1
//
// Every client-go program starts the same way: find the kubeconfig, build a
// *rest.Config from it. kubectl — and every controller you will ever write —
// resolves credentials via the standard loading rules: the $KUBECONFIG env
// var first, then ~/.kube/config. Hard-coding a path breaks the moment the
// code runs on another machine.
//
// Make this program load the kubeconfig using the standard rules, pinned to
// the kind-clientlings context.
package main

import (
	"fmt"
	"os"

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

	if config.Host == "" {
		fmt.Println("❌ the rest.Config has no API server address — kubeconfig not loaded correctly")
		os.Exit(1)
	}

	fmt.Printf("✓ loaded kubeconfig, API server: %s\n", config.Host)
	fmt.Println("\n🎉 you built your first rest.Config — the passport every client-go call carries")
}
