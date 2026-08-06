// setup1
//
// Every client-go program starts the same way: find the kubeconfig, build a
// *rest.Config from it. kubectl — and every controller you will ever write —
// resolves credentials via the standard loading rules: the $KUBECONFIG env
// var first, then ~/.kube/config. Hard-coding a path breaks the moment the
// code runs on another machine.
//
// Make this program load the kubeconfig using the standard rules, pinned to
// the kind-kubeclientlings context.
package main

import (
	"fmt"
	"os"

	"k8s.io/client-go/tools/clientcmd"
)

func main() {
	// Search order kubectl itself uses: $KUBECONFIG (merged left-to-right),
	// then ~/.kube/config. Nothing is read yet — this is just the recipe.
	rules := clientcmd.NewDefaultClientConfigLoadingRules()

	// The in-process equivalent of `kubectl --context=…`, so we reach the kind
	// cluster whatever the user's current context happens to be.
	overrides := &clientcmd.ConfigOverrides{CurrentContext: "kind-kubeclientlings"}

	// "Deferred" = the files are opened here, at ClientConfig(), not above.
	// "NonInteractive" = never prompt on a terminal; error out instead.
	config, err := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(rules, overrides).ClientConfig()
	if err != nil {
		fmt.Printf("❌ could not load kubeconfig: %v\n", err)
		os.Exit(1)
	}

	// A config that "loaded" but resolved to nothing has an empty Host — catch
	// it here rather than as a confusing dial error on the first request.
	if config.Host == "" {
		fmt.Println("❌ the rest.Config has no API server address — kubeconfig not loaded correctly")
		os.Exit(1)
	}

	fmt.Printf("✓ loaded kubeconfig, API server: %s\n", config.Host)
	fmt.Println("\n🎉 you built your first rest.Config — the passport every client-go call carries")
}
