// dyn3
//
// CRDs extend the API at runtime — and the dynamic client is how you talk
// to them without generated code. One catch: a custom resource must be
// created with a GVR the CRD actually SERVES. The version in the GVR must
// be one of the CRD's spec.versions with served=true, or the server 404s
// as if the resource didn't exist at all.
//
// Create the Widget with the version the CRD serves.
package main

import (
	"context"

	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"

	"github.com/madhank93/kubeclientlings/internal/exkit"
)

const crdName = "widgets.kubeclientlings.dev"

func main() {
	ctx, cancel, _, ns := exkit.Begin("dyn3")
	defer cancel()

	apiext := exkit.MustAPIExt()

	// CRDs are cluster-scoped: clean up any copy a previous run left.
	err := apiext.ApiextensionsV1().CustomResourceDefinitions().Delete(ctx, crdName, metav1.DeleteOptions{})
	if err != nil && !apierrors.IsNotFound(err) {
		exkit.Failf("cleaning up CRD from a previous run: %v", err)
	}
	if err == nil {
		exkit.WaitFor(ctx, "old CRD to terminate", func(ctx context.Context) (bool, error) {
			_, err := apiext.ApiextensionsV1().CustomResourceDefinitions().Get(ctx, crdName, metav1.GetOptions{})
			return apierrors.IsNotFound(err), nil
		})
	}

	crd := &apiextensionsv1.CustomResourceDefinition{
		ObjectMeta: metav1.ObjectMeta{Name: crdName},
		Spec: apiextensionsv1.CustomResourceDefinitionSpec{
			Group: "kubeclientlings.dev",
			Names: apiextensionsv1.CustomResourceDefinitionNames{
				Plural: "widgets", Singular: "widget", Kind: "Widget", ListKind: "WidgetList",
			},
			Scope: apiextensionsv1.NamespaceScoped,
			Versions: []apiextensionsv1.CustomResourceDefinitionVersion{{
				Name: "v1alpha1", Served: true, Storage: true,
				Schema: &apiextensionsv1.CustomResourceValidation{
					OpenAPIV3Schema: &apiextensionsv1.JSONSchemaProps{
						Type: "object",
						Properties: map[string]apiextensionsv1.JSONSchemaProps{
							"spec": {
								Type: "object",
								Properties: map[string]apiextensionsv1.JSONSchemaProps{
									"size": {Type: "integer"},
								},
							},
						},
					},
				},
			}},
		},
	}
	if _, err := apiext.ApiextensionsV1().CustomResourceDefinitions().Create(ctx, crd, metav1.CreateOptions{}); err != nil {
		exkit.Failf("creating CRD: %v", err)
	}

	exkit.WaitFor(ctx, "CRD to be Established", func(ctx context.Context) (bool, error) {
		got, err := apiext.ApiextensionsV1().CustomResourceDefinitions().Get(ctx, crdName, metav1.GetOptions{})
		if err != nil {
			return false, err
		}
		for _, cond := range got.Status.Conditions {
			if cond.Type == apiextensionsv1.Established && cond.Status == apiextensionsv1.ConditionTrue {
				return true, nil
			}
		}
		return false, nil
	})

	dyn := exkit.MustDynamic()
	// The CRD above serves exactly one version: v1alpha1.
	gvr := schema.GroupVersionResource{Group: "kubeclientlings.dev", Version: "v1alpha1", Resource: "widgets"}

	widget := &unstructured.Unstructured{Object: map[string]any{
		"apiVersion": "kubeclientlings.dev/v1alpha1",
		"kind":       "Widget",
		"metadata":   map[string]any{"name": "first", "namespace": ns},
		"spec":       map[string]any{"size": int64(3)},
	}}
	if _, err := dyn.Resource(gvr).Namespace(ns).Create(ctx, widget, metav1.CreateOptions{}); err != nil {
		exkit.Failf("creating Widget: %v", err)
	}

	got, err := dyn.Resource(gvr).Namespace(ns).Get(ctx, "first", metav1.GetOptions{})
	if err != nil {
		exkit.Failf("getting Widget back: %v", err)
	}
	size, found, err := unstructured.NestedInt64(got.Object, "spec", "size")
	if err != nil || !found {
		exkit.Failf("reading spec.size: found=%v err=%v", found, err)
	}

	exkit.AssertEqual("the Widget's size", size, int64(3))
	exkit.Successf("your own API type, served by the cluster — no code generation involved")
}
