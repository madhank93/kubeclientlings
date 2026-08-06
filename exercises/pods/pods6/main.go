// pods6
//
// Reading logs is not an object Get — there is no "Log" resource. GetLogs
// returns a *rest.Request you have to STREAM, because a log is an open pipe,
// not a value. And when a pod has more than one container the apiserver
// refuses to guess which one you meant: PodLogOptions.Container is mandatory
// the moment there is more than one.
//
// Read the marker line out of the writer container's log.
//
// I AM NOT DONE
package main

import (
	"context"
	"io"
	"strings"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/madhank93/kubeclientlings/internal/exkit"
)

const marker = "kubeclientlings-was-here"

func main() {
	ctx, cancel, cs, ns := exkit.Begin("pods6")
	defer cancel()

	// Two containers: one prints the marker and idles, one only idles.
	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{Name: "chatty", Namespace: ns},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{
				{
					Name:    "writer",
					Image:   exkit.Image,
					Command: []string{"sh", "-c", "echo " + marker + "; sleep 3600"},
				},
				{
					Name:    "idle",
					Image:   exkit.Image,
					Command: []string{"sh", "-c", "sleep 3600"},
				},
			},
		},
	}
	if _, err := cs.CoreV1().Pods(ns).Create(ctx, pod, metav1.CreateOptions{}); err != nil {
		exkit.Failf("creating pod: %v", err)
	}

	exkit.WaitFor(ctx, "the pod to be Running", func(ctx context.Context) (bool, error) {
		got, err := cs.CoreV1().Pods(ns).Get(ctx, "chatty", metav1.GetOptions{})
		if err != nil {
			return false, err
		}
		return got.Status.Phase == corev1.PodRunning, nil
	})

	var logs string
	exkit.WaitFor(ctx, "the marker to show up in the logs", func(ctx context.Context) (bool, error) {
		// This pod has TWO containers and these options name neither, so the
		// apiserver refuses the request rather than picking one for you —
		// read the error it returns, it even lists the choices. Name the
		// container that prints the marker.
		req := cs.CoreV1().Pods(ns).GetLogs("chatty", &corev1.PodLogOptions{})

		// GetLogs only built a request; Stream is what actually opens it.
		stream, err := req.Stream(ctx)
		if err != nil {
			return false, err
		}
		defer stream.Close()

		data, err := io.ReadAll(stream)
		if err != nil {
			return false, err
		}
		logs = string(data)
		return strings.Contains(logs, marker), nil
	})

	exkit.AssertTrue("the writer container's log carries the marker", strings.Contains(logs, marker))
	exkit.Successf("streamed a container's logs — GetLogs builds the request, Stream opens the pipe")
}
