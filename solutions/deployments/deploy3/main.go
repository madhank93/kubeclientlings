// deploy3
//
// Patching a container image is the everyday "deploy a new version" call.
// containers is a LIST, and strategic merge patch needs the list's merge
// key — the container NAME — to know which element you mean. Omit it and
// the server rejects the patch.
//
// Patch the web container to run a newer nginx.
package main

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	"github.com/madhank93/clientlings/internal/exkit"
)

const newImage = "nginx:1.28-alpine"

func main() {
	ctx, cancel, cs, ns := exkit.Begin("deploy3")
	defer cancel()

	if _, err := cs.AppsV1().Deployments(ns).Create(ctx, exkit.NginxDeployment(ns, "web", 1), metav1.CreateOptions{}); err != nil {
		exkit.Failf("creating deployment: %v", err)
	}

	patch := []byte(`{"spec":{"template":{"spec":{"containers":[{"name":"web","image":"` + newImage + `"}]}}}}`)
	_, err := cs.AppsV1().Deployments(ns).Patch(ctx, "web", types.StrategicMergePatchType, patch, metav1.PatchOptions{})
	if err != nil {
		exkit.Failf("patching image: %v", err)
	}

	got, err := cs.AppsV1().Deployments(ns).Get(ctx, "web", metav1.GetOptions{})
	if err != nil {
		exkit.Failf("re-reading deployment: %v", err)
	}
	exkit.AssertEqual("container image", got.Spec.Template.Spec.Containers[0].Image, newImage)
	exkit.Successf("image patched — the merge key (container name) told the server which element to update")
}
