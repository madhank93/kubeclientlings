// test1
//
// You do not need a cluster to test client-go code — the fake clientset
// implements the same typed interface backed by an in-memory object tracker.
// Seed it with objects and every Get/List/Update works against that tracker.
//
// The one rule: the tracker keys objects by namespace AND name, exactly as
// seeded. Query the wrong namespace and you get an ordinary NotFound.
package main

import (
	"context"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"

	"github.com/madhank93/kubeclientlings/internal/exkit"
)

func main() {
	ctx := context.Background()

	cs := fake.NewClientset(
		&corev1.Pod{ObjectMeta: metav1.ObjectMeta{Name: "seeded", Namespace: "demo"}},
	)

	got, err := cs.CoreV1().Pods("demo").Get(ctx, "seeded", metav1.GetOptions{})
	if err != nil {
		exkit.Failf("getting the seeded pod: %v", err)
	}
	exkit.AssertEqual("the pod name the fake tracker returned", got.Name, "seeded")

	list, err := cs.CoreV1().Pods("demo").List(ctx, metav1.ListOptions{})
	if err != nil {
		exkit.Failf("listing pods: %v", err)
	}
	exkit.AssertEqual("pods the fake tracker holds in demo", len(list.Items), 1)

	exkit.Successf("fake.NewClientset is the real typed interface over an in-memory tracker — no cluster, no flakes")
}
