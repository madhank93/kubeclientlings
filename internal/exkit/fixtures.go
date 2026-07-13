package exkit

import (
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

// Image is the container image all fixtures run. nginx is tiny, quick to
// pull once, and cached on the kind nodes after the first exercise.
const Image = "nginx:1.27-alpine"

// NginxPod returns a minimal runnable pod spec so exercises stay focused on
// the client-go call being taught, not on pod YAML.
func NginxPod(ns, name string) *corev1.Pod {
	return &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: ns,
			Labels:    map[string]string{"app": name},
		},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{{
				Name:  "web",
				Image: Image,
			}},
		},
	}
}

// WebService returns a ClusterIP service on port 80 selecting app=<name>,
// matching the pods NginxDeployment(ns, name, …) creates.
func WebService(ns, name string) *corev1.Service {
	return &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: ns},
		Spec: corev1.ServiceSpec{
			Selector: map[string]string{"app": name},
			Ports: []corev1.ServicePort{{
				Port:       80,
				TargetPort: intstr.FromInt32(80),
			}},
		},
	}
}

// NginxDeployment returns a minimal deployment whose selector matches its
// template labels.
func NginxDeployment(ns, name string, replicas int32) *appsv1.Deployment {
	labels := map[string]string{"app": name}
	return &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: ns,
			Labels:    labels,
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: &replicas,
			Selector: &metav1.LabelSelector{MatchLabels: labels},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{Labels: labels},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{{
						Name:  "web",
						Image: Image,
					}},
				},
			},
		},
	}
}
