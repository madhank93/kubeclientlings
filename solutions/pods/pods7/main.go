// pods7
//
// `kubectl exec` is not an API call you can make with the typed clientset —
// it is a protocol UPGRADE. You hand-build a request to the pod's exec
// subresource, then hand its URL to an executor that switches the connection
// to SPDY and multiplexes stdin/stdout/stderr over it.
//
// The trap: PodExecOptions and StreamOptions BOTH mention stdout, and they
// mean different things. StreamOptions says where to put the bytes locally;
// PodExecOptions asks the apiserver to send them at all. Attach a writer for a
// stream you never requested and you get silence, not an error.
//
// Capture the command's stdout.
package main

import (
	"bytes"
	"context"
	"strings"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/tools/remotecommand"

	"github.com/madhank93/kubeclientlings/internal/exkit"
)

const marker = "hello-from-inside"

func main() {
	ctx, cancel, cs, ns := exkit.Begin("pods7")
	defer cancel()

	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{Name: "shell", Namespace: ns},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{{
				Name:    "app",
				Image:   exkit.Image,
				Command: []string{"sh", "-c", "sleep 3600"},
			}},
		},
	}
	if _, err := cs.CoreV1().Pods(ns).Create(ctx, pod, metav1.CreateOptions{}); err != nil {
		exkit.Failf("creating pod: %v", err)
	}

	// Exec needs a container that is actually up, not merely scheduled.
	exkit.WaitFor(ctx, "the container to be ready", func(ctx context.Context) (bool, error) {
		got, err := cs.CoreV1().Pods(ns).Get(ctx, "shell", metav1.GetOptions{})
		if err != nil {
			return false, err
		}
		for _, st := range got.Status.ContainerStatuses {
			if st.Name == "app" && st.Ready {
				return true, nil
			}
		}
		return false, nil
	})

	// The request is built by hand: POST to .../pods/shell/exec, with the
	// PodExecOptions encoded into the QUERY STRING by VersionedParams (which
	// is what ParameterCodec is for — these options travel as ?stdout=true,
	// not as a body).
	req := cs.CoreV1().RESTClient().Post().
		Resource("pods").
		Name("shell").
		Namespace(ns).
		SubResource("exec").
		VersionedParams(&corev1.PodExecOptions{
			Container: "app",
			Command:   []string{"sh", "-c", "echo " + marker},
			// Ask the apiserver to actually send these streams. Setting a
			// writer in StreamOptions is not enough on its own — these two
			// flags are what put ?stdout=true&stderr=true on the URL.
			Stdout: true,
			Stderr: true,
		}, scheme.ParameterCodec)

	// SPDY upgrade. Note this takes the *rest.Config, not the clientset: the
	// executor needs the raw TLS/auth material to build its own connection.
	executor, err := remotecommand.NewSPDYExecutor(exkit.MustRESTConfig(), "POST", req.URL())
	if err != nil {
		exkit.Failf("building the executor: %v", err)
	}

	// StreamWithContext, not the older Stream: Stream is deprecated because
	// it has no way to abandon a session, so a wedged exec leaks the
	// connection and its goroutines.
	var stdout, stderr bytes.Buffer
	if err := executor.StreamWithContext(ctx, remotecommand.StreamOptions{
		Stdout: &stdout,
		Stderr: &stderr,
	}); err != nil {
		exkit.Failf("streaming the exec session: %v (stderr: %s)", err, stderr.String())
	}

	got := strings.TrimSpace(stdout.String())
	exkit.AssertEqual("stdout captured from the container", got, marker)
	exkit.Successf("exec'd into a container over SPDY and read its stdout back")
}
