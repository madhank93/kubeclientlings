// test2
//
// Real code must survive API failures — but you cannot make a live server
// return "etcd unavailable" on demand. Reactors can: a reactor intercepts
// matching calls on the fake client and returns whatever you want. That is
// how you test the sad path.
//
// A reactor matches on (verb, resource). PrependReactor puts it at the front
// of the chain so it fires first, and returning handled=true short-circuits
// the default tracker.
//
// I AM NOT DONE
package main

import (
	"context"
	"fmt"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/fake"
	k8stesting "k8s.io/client-go/testing"

	"github.com/madhank93/kubeclientlings/internal/exkit"
)

func main() {
	ctx := context.Background()

	cs := fake.NewClientset()

	// This reactor is registered for the "update" verb — but the call below
	// is a CREATE. A reactor only fires when its verb matches, so this never
	// runs, the create hits the tracker and succeeds, and err is nil. Match
	// the verb to the call you want to break.
	cs.PrependReactor("update", "pods", func(action k8stesting.Action) (bool, runtime.Object, error) {
		return true, nil, apierrors.NewInternalError(fmt.Errorf("etcd unavailable"))
	})

	_, err := cs.CoreV1().Pods("demo").Create(ctx,
		&corev1.Pod{ObjectMeta: metav1.ObjectMeta{Name: "doomed", Namespace: "demo"}},
		metav1.CreateOptions{})

	exkit.AssertTrue("create surfaced the injected failure", err != nil)
	exkit.AssertTrue("and it was an internal server error", apierrors.IsInternalError(err))

	exkit.Successf("reactors fake the sad path — (verb, resource) must match the call you want to break")
}
