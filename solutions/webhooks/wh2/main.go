// wh2
//
// A mutating webhook changes the object by returning a JSON patch. The gotcha
// is invisible: the AdmissionResponse must announce its PatchType, or the
// apiserver silently ignores the patch bytes — no error, the mutation just
// never happens. A JSON patch always travels with PatchType = JSONPatch.
//
// Build a response that injects a default label, and declare its patch type.
package main

import (
	"encoding/json"

	admissionv1 "k8s.io/api/admission/v1"
	"k8s.io/apimachinery/pkg/types"

	"github.com/madhank93/kubeclientlings/internal/exkit"
)

func mutate(review *admissionv1.AdmissionReview) *admissionv1.AdmissionResponse {
	resp := &admissionv1.AdmissionResponse{UID: review.Request.UID, Allowed: true}

	// RFC 6902 JSON Patch: an ARRAY of {op, path, value}, not the
	// object-shaped merge patch from pods4. Note this "add" on
	// /metadata/labels replaces the whole map — target
	// /metadata/labels/injected to add one key, but only when the map
	// already exists (JSON Patch won't create intermediate objects).
	patch := []map[string]any{
		{"op": "add", "path": "/metadata/labels", "value": map[string]string{"injected": "true"}},
	}
	resp.Patch, _ = json.Marshal(patch)

	// Without this, the apiserver drops the patch silently — no error, no
	// mutation. It's a *PatchType, hence the variable (or ptr.To(...)).
	pt := admissionv1.PatchTypeJSONPatch
	resp.PatchType = &pt

	return resp
}

func main() {
	review := &admissionv1.AdmissionReview{
		Request: &admissionv1.AdmissionRequest{UID: types.UID("req-9")},
	}

	resp := mutate(review)

	exkit.AssertTrue("the response carries a patch body", len(resp.Patch) > 0)
	exkit.AssertTrue("the response declares its patch type", resp.PatchType != nil)
	exkit.AssertEqual("and the patch type is JSONPatch", *resp.PatchType, admissionv1.PatchTypeJSONPatch)

	exkit.Successf("a mutating response must set PatchType=JSONPatch or the patch is silently ignored")
}
