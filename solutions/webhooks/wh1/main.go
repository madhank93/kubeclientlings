// wh1
//
// A validating webhook is a plain function: apiserver sends an AdmissionReview
// describing the object under review, your handler returns an AdmissionResponse
// that allows or denies it. Two rules that trip everyone up: you MUST echo the
// request UID back, and Allowed defaults to false — so an "allow" path has to
// set it explicitly.
//
// This handler denies any pod that lacks a "team" label. Test it directly —
// no cluster, no TLS, just the function.
package main

import (
	"encoding/json"

	admissionv1 "k8s.io/api/admission/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"

	"github.com/madhank93/kubeclientlings/internal/exkit"
)

func validate(review *admissionv1.AdmissionReview) *admissionv1.AdmissionResponse {
	resp := &admissionv1.AdmissionResponse{UID: review.Request.UID}

	var pod corev1.Pod
	if err := json.Unmarshal(review.Request.Object.Raw, &pod); err != nil {
		resp.Allowed = false
		resp.Result = &metav1.Status{Message: "could not decode pod: " + err.Error()}
		return resp
	}

	// Deny when the team label is ABSENT.
	if _, ok := pod.Labels["team"]; !ok {
		resp.Allowed = false
		resp.Result = &metav1.Status{Message: "pods must carry a team label"}
		return resp
	}

	resp.Allowed = true
	return resp
}

func reviewFor(pod *corev1.Pod) *admissionv1.AdmissionReview {
	raw, _ := json.Marshal(pod)
	return &admissionv1.AdmissionReview{
		Request: &admissionv1.AdmissionRequest{
			UID:    types.UID("req-123"),
			Object: runtime.RawExtension{Raw: raw},
		},
	}
}

func main() {
	denied := validate(reviewFor(&corev1.Pod{ObjectMeta: metav1.ObjectMeta{Name: "nolabel"}}))
	exkit.AssertEqual("the response echoes the request UID", string(denied.UID), "req-123")
	exkit.AssertTrue("a pod with no team label is denied", !denied.Allowed)

	allowed := validate(reviewFor(&corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{Name: "ok", Labels: map[string]string{"team": "payments"}},
	}))
	exkit.AssertTrue("a pod carrying a team label is allowed", allowed.Allowed)

	exkit.Successf("a webhook is just a function: echo the UID, deny by default, allow explicitly")
}
