package controller

import (
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	controllerv1 "saleh.dev/kubebuilder-crd/api/v1alpha1"
)

func CheckPort(port int32) int32 {
	if port == 0 {
		port = 3005
	}
	return port
}

func CheckImage(image string) string {
	if image == "" {
		image = controllerv1.DockerImage
	}
	return image
}

// newDeployment creates a new Deployment for a Uban resource. It also sets
// the appropriate OwnerReferences on the resource so handleObject can discover
// the Uban resource that 'owns' it.

func newDeployment(uban *controllerv1.Uban, deploymentName string) *appsv1.Deployment {
	return &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      deploymentName,
			Namespace: uban.Namespace,
			OwnerReferences: []metav1.OwnerReference{
				*metav1.NewControllerRef(uban, controllerv1.GroupVersion.WithKind(controllerv1.UBAN)),
			},
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: uban.Spec.Replicas,
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"app":  uban.Name,
					"kind": controllerv1.UBAN,
				},
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"app":  uban.Name,
						"kind": controllerv1.UBAN,
					},
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:  uban.Name,
							Image: CheckImage(uban.Spec.Container.Image),
							Ports: []corev1.ContainerPort{
								{
									Name:          controllerv1.HTTP,
									Protocol:      corev1.ProtocolTCP,
									ContainerPort: CheckPort(uban.Spec.Container.Port),
								},
							},
						},
					},
				},
			},
		},
	}
}

func newService(uban *controllerv1.Uban, serviceName string) *corev1.Service {
	labels := map[string]string{
		"app": uban.Name,
	}
	return &corev1.Service{
		TypeMeta: metav1.TypeMeta{
			Kind: controllerv1.KindService,
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      serviceName,
			Namespace: uban.Namespace,
			OwnerReferences: []metav1.OwnerReference{
				*metav1.NewControllerRef(uban, controllerv1.GroupVersion.WithKind(controllerv1.UBAN)),
			},
		},
		Spec: corev1.ServiceSpec{
			Selector: labels,
			Ports: []corev1.ServicePort{
				{
					Port:       CheckPort(uban.Spec.Container.Port),
					TargetPort: intstr.FromInt(int(CheckPort(uban.Spec.Container.Port))),
					Protocol:   controllerv1.TCP,
					NodePort:   uban.Spec.Service.ServiceNodePort,
				},
			},
			Type: func() corev1.ServiceType {
				if uban.Spec.Service.ServiceType == controllerv1.NodePort {
					return corev1.ServiceTypeNodePort
				} else {
					return corev1.ServiceTypeClusterIP
				}
			}(),
		},
	}
}
