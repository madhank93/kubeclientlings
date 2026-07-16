// opt4
//
// ownerReferences are how Kubernetes garbage collection works: delete the
// owner and everything pointing at it gets cleaned up. A valid reference
// needs the owner's UID — the value the server assigned at creation — not
// just its name. This is exactly what controllers set on every object they
// create.
//
// Give the child a proper owner reference, then watch GC cascade.
//
// I AM NOT DONE
package main

import (
	"context"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/madhank93/kubeclientlings/internal/exkit"
)

func main() {
	ctx, cancel, cs, ns := exkit.Begin("opt4")
	defer cancel()

	owner, err := cs.CoreV1().ConfigMaps(ns).Create(ctx, &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{Name: "owner", Namespace: ns},
		Data:       map[string]string{"role": "owner"},
	}, metav1.CreateOptions{})
	if err != nil {
		exkit.Failf("creating owner: %v", err)
	}

	child := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "child",
			Namespace: ns,
			// The name alone doesn't identify an owner — names get reused,
			// UIDs never do. The server rejects this reference because it
			// is missing the owner's server-assigned identity.
			OwnerReferences: []metav1.OwnerReference{{
				APIVersion: "v1",
				Kind:       "ConfigMap",
				Name:       owner.Name,
			}},
		},
		Data: map[string]string{"role": "child"},
	}
	if _, err := cs.CoreV1().ConfigMaps(ns).Create(ctx, child, metav1.CreateOptions{}); err != nil {
		exkit.Failf("creating child: %v", err)
	}

	if err := cs.CoreV1().ConfigMaps(ns).Delete(ctx, "owner", metav1.DeleteOptions{}); err != nil {
		exkit.Failf("deleting owner: %v", err)
	}

	exkit.WaitFor(ctx, "garbage collection to delete the child", func(ctx context.Context) (bool, error) {
		_, err := cs.CoreV1().ConfigMaps(ns).Get(ctx, "child", metav1.GetOptions{})
		if apierrors.IsNotFound(err) {
			return true, nil
		}
		if err != nil {
			return false, err
		}
		return false, nil
	})

	exkit.Successf("owner deleted → child garbage-collected. That's the ownerReference contract")
}
