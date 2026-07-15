// sub2
//
// When a resource has the status subresource enabled, status becomes a
// separate write path. The main endpoint IGNORES any change to status, and
// only UpdateStatus (the /status subresource) can write it. This split is
// what stops a controller writing status and a user writing spec from
// clobbering each other.
//
// A custom resource is perfect for seeing this cleanly — nothing else
// reconciles it. Write the Widget's status through the right endpoint.
package main

import (
	"context"

	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"

	"github.com/madhank93/clientlings/internal/exkit"
)

const crdName = "widgets.clientlings.dev"

func main() {
	ctx, cancel, _, ns := exkit.Begin("sub2")
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
			Group: "clientlings.dev",
			Names: apiextensionsv1.CustomResourceDefinitionNames{
				Plural: "widgets", Singular: "widget", Kind: "Widget", ListKind: "WidgetList",
			},
			Scope: apiextensionsv1.NamespaceScoped,
			Versions: []apiextensionsv1.CustomResourceDefinitionVersion{{
				Name: "v1alpha1", Served: true, Storage: true,
				// Enabling the status subresource is what splits the write paths.
				Subresources: &apiextensionsv1.CustomResourceSubresources{
					Status: &apiextensionsv1.CustomResourceSubresourceStatus{},
				},
				Schema: &apiextensionsv1.CustomResourceValidation{
					OpenAPIV3Schema: &apiextensionsv1.JSONSchemaProps{
						Type: "object",
						Properties: map[string]apiextensionsv1.JSONSchemaProps{
							"spec": {
								Type:       "object",
								Properties: map[string]apiextensionsv1.JSONSchemaProps{"size": {Type: "integer"}},
							},
							"status": {
								Type:       "object",
								Properties: map[string]apiextensionsv1.JSONSchemaProps{"phase": {Type: "string"}},
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
	gvr := schema.GroupVersionResource{Group: "clientlings.dev", Version: "v1alpha1", Resource: "widgets"}

	widget := &unstructured.Unstructured{Object: map[string]any{
		"apiVersion": "clientlings.dev/v1alpha1",
		"kind":       "Widget",
		"metadata":   map[string]any{"name": "first", "namespace": ns},
		"spec":       map[string]any{"size": int64(3)},
	}}
	if _, err := dyn.Resource(gvr).Namespace(ns).Create(ctx, widget, metav1.CreateOptions{}); err != nil {
		exkit.Failf("creating Widget: %v", err)
	}

	// Set the status and persist it through the /status subresource.
	got, err := dyn.Resource(gvr).Namespace(ns).Get(ctx, "first", metav1.GetOptions{})
	if err != nil {
		exkit.Failf("getting the Widget: %v", err)
	}
	if err := unstructured.SetNestedField(got.Object, "Ready", "status", "phase"); err != nil {
		exkit.Failf("setting status.phase: %v", err)
	}
	if _, err := dyn.Resource(gvr).Namespace(ns).UpdateStatus(ctx, got, metav1.UpdateOptions{}); err != nil {
		exkit.Failf("updating status: %v", err)
	}

	final, err := dyn.Resource(gvr).Namespace(ns).Get(ctx, "first", metav1.GetOptions{})
	if err != nil {
		exkit.Failf("re-reading the Widget: %v", err)
	}
	phase, _, _ := unstructured.NestedString(final.Object, "status", "phase")
	exkit.AssertEqual("status.phase written through the subresource", phase, "Ready")

	exkit.Successf("status is a separate write path — only UpdateStatus can move it")
}
