// opt5
//
// A List with no Limit fetches EVERYTHING in one response. On a namespace with
// 200k pods that is a multi-hundred-megabyte body the apiserver has to hold in
// memory, serialize and send — the single easiest way for a client to hurt a
// cluster. Limit asks for one page; the response comes back with a CONTINUE
// token, and you keep passing it back until it is empty.
//
// Page through all the configmaps instead of demanding them in one gulp.
package main

import (
	"fmt"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/madhank93/kubeclientlings/internal/exkit"
)

func main() {
	ctx, cancel, cs, ns := exkit.Begin("opt5")
	defer cancel()

	// 12 configmaps, all labelled so the namespace's own kube-root-ca.crt
	// doesn't join the count.
	for i := 1; i <= 12; i++ {
		cm := &corev1.ConfigMap{ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("page-%02d", i),
			Namespace: ns,
			Labels:    map[string]string{"paged": "yes"},
		}}
		if _, err := cs.CoreV1().ConfigMaps(ns).Create(ctx, cm, metav1.CreateOptions{}); err != nil {
			exkit.Failf("creating %s: %v", cm.Name, err)
		}
	}

	var names []string
	pages := 0

	// The canonical paging loop. Continue starts empty ("give me the first
	// page") and is refilled from each response until the server hands back
	// an empty token, which means "that was the last page".
	cont := ""
	for {
		list, err := cs.CoreV1().ConfigMaps(ns).List(ctx, metav1.ListOptions{
			LabelSelector: "paged=yes",
			Limit:         5,
			Continue:      cont,
		})
		if err != nil {
			exkit.Failf("listing configmaps: %v", err)
		}
		pages++
		for _, cm := range list.Items {
			names = append(names, cm.Name)
		}
		// The cursor for the next request. Empty => no more pages.
		cont = list.Continue
		if cont == "" {
			break
		}
	}

	exkit.AssertEqual("configmaps collected across all pages", len(names), 12)
	exkit.AssertEqual("pages fetched (12 items, 5 per page)", pages, 3)
	exkit.AssertEqual("first item of the first page", names[0], "page-01")
	exkit.AssertEqual("last item of the last page", names[11], "page-12")
	exkit.Successf("paged through %d configmaps in %d requests — bounded memory, happy apiserver", len(names), pages)
}
