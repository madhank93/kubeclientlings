// Package exkit is the shared scaffolding every exercise builds on: config
// loading pinned to the local kind cluster, fresh per-exercise namespaces,
// wait/assert helpers with friendly failure messages, and canned fixtures.
//
// exkit is always correct — the bug is always in the exercise's own use of
// client-go, never in here.
package exkit

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	corev1 "k8s.io/api/core/v1"
	apiextensionsclient "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

const (
	// DefaultContext is the kubeconfig context every exercise talks to.
	DefaultContext = "kind-clientlings"

	// ExerciseLabel marks namespaces created by exercises so `mise run nuke`
	// can sweep them all without touching anything else.
	ExerciseLabel = "clientlings.dev/exercise"

	// BeginTimeout bounds a whole exercise run via the context Begin returns.
	BeginTimeout = 90 * time.Second
)

// Context returns the kubeconfig context exercises use: CLIENTLINGS_CONTEXT
// if set, otherwise kind-clientlings.
func Context() string {
	if ctx := os.Getenv("CLIENTLINGS_CONTEXT"); ctx != "" {
		return ctx
	}
	return DefaultContext
}

// RESTConfig loads kubeconfig with the standard clientcmd rules ($KUBECONFIG,
// ~/.kube/config) pinned to Context(). As a safety net it refuses any context
// that does not start with "kind-", so an exercise can never touch a real
// cluster no matter what the environment points at.
func RESTConfig() (*rest.Config, error) {
	name := Context()
	if !strings.HasPrefix(name, "kind-") {
		return nil, fmt.Errorf(
			"refusing to use context %q: exercises only run against local kind clusters (context must start with \"kind-\")", name)
	}

	rules := clientcmd.NewDefaultClientConfigLoadingRules()
	overrides := &clientcmd.ConfigOverrides{CurrentContext: name}
	cfg, err := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(rules, overrides).ClientConfig()
	if err != nil {
		return nil, fmt.Errorf("loading kubeconfig for context %q: %w (is the cluster up? try `clientlings up`)", name, err)
	}
	return cfg, nil
}

// MustRESTConfig is RESTConfig or exit 1 with the error message.
func MustRESTConfig() *rest.Config {
	cfg, err := RESTConfig()
	if err != nil {
		Failf("%v", err)
	}
	return cfg
}

// MustClientset returns a typed clientset for the exercise cluster.
func MustClientset() kubernetes.Interface {
	cs, err := kubernetes.NewForConfig(MustRESTConfig())
	if err != nil {
		Failf("creating clientset: %v", err)
	}
	return cs
}

// MustDynamic returns a dynamic (unstructured) client for the exercise cluster.
func MustDynamic() dynamic.Interface {
	dc, err := dynamic.NewForConfig(MustRESTConfig())
	if err != nil {
		Failf("creating dynamic client: %v", err)
	}
	return dc
}

// MustAPIExt returns an apiextensions clientset (CRD management).
func MustAPIExt() apiextensionsclient.Interface {
	c, err := apiextensionsclient.NewForConfig(MustRESTConfig())
	if err != nil {
		Failf("creating apiextensions clientset: %v", err)
	}
	return c
}

// MustDiscovery returns a discovery client for the exercise cluster.
func MustDiscovery() discovery.DiscoveryInterface {
	d, err := discovery.NewDiscoveryClientForConfig(MustRESTConfig())
	if err != nil {
		Failf("creating discovery client: %v", err)
	}
	return d
}

// Begin sets up an isolated, re-runnable workspace for one exercise: it
// deletes the namespace clx-<name> if a previous run left it behind, waits
// for it to be gone, and recreates it fresh. The returned context carries a
// 90s deadline covering the whole exercise.
func Begin(name string) (context.Context, context.CancelFunc, kubernetes.Interface, string) {
	cs := MustClientset()
	ctx, cancel := context.WithTimeout(context.Background(), BeginTimeout)
	ns := "clx-" + name

	// A previous run (especially a failed finalizers exercise) may have left a
	// ConfigMap carrying a finalizer. That finalizer would wedge the namespace
	// delete below in Terminating forever, so strip finalizers first. This is
	// best-effort: if the namespace does not exist yet the list simply fails.
	if cms, err := cs.CoreV1().ConfigMaps(ns).List(ctx, metav1.ListOptions{}); err == nil {
		for i := range cms.Items {
			if len(cms.Items[i].Finalizers) > 0 {
				cms.Items[i].Finalizers = nil
				_, _ = cs.CoreV1().ConfigMaps(ns).Update(ctx, &cms.Items[i], metav1.UpdateOptions{})
			}
		}
	}

	err := cs.CoreV1().Namespaces().Delete(ctx, ns, metav1.DeleteOptions{})
	if err != nil && !apierrors.IsNotFound(err) {
		cancel()
		Failf("cleaning up namespace %s from a previous run: %v", ns, err)
	}
	if err == nil {
		WaitFor(ctx, fmt.Sprintf("namespace %s from the previous run to terminate", ns),
			func(ctx context.Context) (bool, error) {
				_, err := cs.CoreV1().Namespaces().Get(ctx, ns, metav1.GetOptions{})
				if apierrors.IsNotFound(err) {
					return true, nil
				}
				return false, nil
			})
	}

	_, err = cs.CoreV1().Namespaces().Create(ctx, &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name:   ns,
			Labels: map[string]string{ExerciseLabel: name},
		},
	}, metav1.CreateOptions{})
	if err != nil {
		cancel()
		Failf("creating namespace %s: %v", ns, err)
	}

	return ctx, cancel, cs, ns
}

// WaitFor polls cond every second until it returns true, errors, or ctx
// expires — printing what it is waiting for so a stuck exercise is
// self-explanatory.
func WaitFor(ctx context.Context, desc string, cond func(ctx context.Context) (bool, error)) {
	fmt.Printf("… waiting for %s\n", desc)
	err := wait.PollUntilContextCancel(ctx, time.Second, true, cond)
	if err != nil {
		Failf("timed out waiting for %s: %v", desc, err)
	}
}

// AssertEqual fails the exercise with a friendly diff-style message unless
// got == want.
func AssertEqual[T comparable](desc string, got, want T) {
	if got != want {
		Failf("%s:\n  got:  %v\n  want: %v", desc, got, want)
	}
	fmt.Printf("✓ %s\n", desc)
}

// AssertTrue fails the exercise unless ok is true.
func AssertTrue(desc string, ok bool) {
	if !ok {
		Failf("%s: expected true", desc)
	}
	fmt.Printf("✓ %s\n", desc)
}

// Failf prints the failure and exits non-zero: the runner shows the message
// and keeps the exercise in the Failing state.
func Failf(format string, args ...any) {
	fmt.Printf("\n❌ "+format+"\n", args...)
	os.Exit(1)
}

// Successf prints the final success line; exiting 0 marks the run as passed.
func Successf(format string, args ...any) {
	fmt.Printf("\n🎉 "+format+"\n", args...)
}
