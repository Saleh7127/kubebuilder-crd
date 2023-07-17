/*
Copyright 2023.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package controller

import (
	"context"
	"fmt"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	salehdevv1alpha1 "saleh.dev/kubebuilder-crd/api/v1alpha1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"time"
)

// UbanReconciler reconciles a Uban object
type UbanReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

//+kubebuilder:rbac:groups=saleh.dev.saleh.dev,resources=ubans,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=saleh.dev.saleh.dev,resources=ubans/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=saleh.dev.saleh.dev,resources=ubans/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the Uban object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.15.0/pkg/reconcile

func (r *UbanReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logs := log.FromContext(ctx)
	logs.WithValues("RequestName", req.Name, "RequestNamespace", req.Namespace)
	/*
		### 1: Load the Uban by name

		We'll fetch the Uban using our client.  All client methods take a
		context (to allow for cancellation) as their first argument, and the object
		in question as their last.  Get is a bit special, in that it takes a
		[`NamespacedName`](https://pkg.go.dev/sigs.k8s.io/controller-runtime/pkg/client?tab=doc#ObjectKey)
		as the middle argument (most don't have a middle argument, as we'll see
		below).
		Many client methods also take variadic options at the end.
	*/

	var uban salehdevv1alpha1.Uban

	if err := r.Get(ctx, req.NamespacedName, &uban); err != nil {
		fmt.Println(err, "Unable to fetch Uban")
		// we'll ignore not-found errors, since they can't be fixed by an immediate
		// requeue (we'll need to wait for a new notification), and we can get them
		// on deleted requests.
		return ctrl.Result{}, nil
	}

	fmt.Println("Name: ", uban.Name)

	// deploymentObject carry the all data of deployment in specific namespace and name
	var deploymentObject appsv1.Deployment
	//Naming of deployment
	deploymentName := func() string {
		var name string
		if uban.Spec.DeploymentName == "" {
			name = fmt.Sprintf("%s-missing-deployment-name", uban.Name)
		} else {
			name = fmt.Sprintf("%s-%s-depl", uban.Name, uban.Spec.DeploymentName)
		}
		return name
	}()

	//Creating NamespacedName for deploymentObject

	deploymentObjectKey := client.ObjectKey{
		Namespace: req.Namespace,
		Name:      deploymentName,
	}

	if err := r.Get(ctx, deploymentObjectKey, &deploymentObject); err != nil {
		if errors.IsNotFound(err) {
			fmt.Println("Could not find existing Deployment for ", uban.Name, ", creating new one ....")
			err := r.Client.Create(ctx, newDeployment(&uban, deploymentName))
			if err != nil {
				fmt.Errorf("Error while creating deployment %s\n", err)
				return ctrl.Result{}, err
			} else {
				fmt.Printf("%s Deployments Created...\n", uban.Name)
			}
		} else {
			fmt.Errorf("Error Fetching Deployments %s\n", err)
		}
	} else {
		if uban.Spec.Replicas != nil && *uban.Spec.Replicas != *deploymentObject.Spec.Replicas {
			fmt.Println(*uban.Spec.Replicas, *deploymentObject.Spec.Replicas)
			fmt.Println("Deployment replicas miss match .... updating")
			//As the replica count didn't match, we need to update it
			deploymentObject.Spec.Replicas = uban.Spec.Replicas
			if err := r.Update(ctx, &deploymentObject); err != nil {
				fmt.Errorf("Error Updating Deployment %s\n", err)
				return ctrl.Result{}, err
			}
			fmt.Println("Deployment Updated")
		}
	}

	var serviceObject corev1.Service
	//making service name

	serviceName := func() string {
		var name string
		// name must contain (lowercase . - )
		if uban.Spec.Service.ServiceName == "" {
			name = fmt.Sprintf("%s-missing-service-name", uban.Name)
		} else {
			name = fmt.Sprintf("%s-%s-depl", uban.Name, uban.Spec.Service.ServiceName)
		}
		return name
	}()

	serviceObjectKey := client.ObjectKey{
		Namespace: req.Namespace,
		Name:      serviceName,
	}

	if err := r.Get(ctx, serviceObjectKey, &serviceObject); err != nil {
		if errors.IsNotFound(err) {
			fmt.Println("Could not find existing Service for ", uban.Name, ", creating new one ....")
			err := r.Client.Create(ctx, newService(&uban, serviceName))
			if err != nil {
				fmt.Errorf("Error while creating service %s\n", err)
				return ctrl.Result{}, err
			} else {
				fmt.Printf("%s Service Created...\n", uban.Name)
			}
		} else {
			fmt.Errorf("Error Fetching Deployments %s\n", err)
		}
	} else {
		if uban.Spec.Replicas != nil && *uban.Spec.Replicas != uban.Status.AvailableReplicas {
			fmt.Println("Service replicas miss match .... updating")
			//As the replica count didn't match, we need to update it
			uban.Status.AvailableReplicas = *uban.Spec.Replicas
			if err := r.Status().Update(ctx, &serviceObject); err != nil {
				fmt.Errorf("Error Updating Service %s\n", err)
				return ctrl.Result{}, err
			}
			fmt.Println("Service Updated")
		}
	}
	fmt.Println("Reconciliation Done")
	return ctrl.Result{
		Requeue:      true,
		RequeueAfter: 1 * time.Minute,
	}, nil
}

var (
	deploymentOwmnerKey = ".metadata.controller"
	serviceOwnerKey     = ".metadata.controller"
	ourApiGVStr         = salehdevv1alpha1.GroupVersion.String()
	ourKind             = "Uban"
)

// SetupWithManager sets up the controller with the Manager.
func (r *UbanReconciler) SetupWithManager(mgr ctrl.Manager) error {
	// Indexing our Owns resource.This will allow for quickly answer the question:
	// If Owns Resource x is updated, which CustomR are affected?

	if err := mgr.GetFieldIndexer().IndexField(context.Background(), &appsv1.Deployment{}, deploymentOwmnerKey, func(object client.Object) []string {
		//grab the deployment object, extract the owner.
		deployment := object.(*appsv1.Deployment)
		owner := metav1.GetControllerOf(deployment)
		if owner == nil {
			return nil
		}
		//make sure it's a Uban Resource
		if owner.APIVersion != ourApiGVStr || owner.Kind != ourKind {
			return nil
		}
		// ...and if so, return it
		return []string{owner.Name}
	}); err != nil {
		return err
	}

	if err := mgr.GetFieldIndexer().IndexField(context.Background(), &corev1.Service{}, serviceOwnerKey, func(object client.Object) []string {

		service := object.(*corev1.Service)
		owner := metav1.GetControllerOf(service)
		if owner == nil {
			return nil
		}
		//make sure it's a Uban Resource
		if owner.APIVersion != ourApiGVStr || owner.Kind != ourKind {
			return nil
		}
		// ...and if so, return it
		return []string{owner.Name}

	}); err != nil {
		return err
	}

	// Extra part
	// Implementation with watches and custom eventHandler
	// if someone edit the resources(here example given for deployment resource) by kubectl

	handlerForDeployment := handler.EnqueueRequestsFromMapFunc(func(ctx context.Context, obj client.Object) []reconcile.Request {
		// List all the Custom Resource
		ubans := &salehdevv1alpha1.UbanList{}
		if err := r.List(context.Background(), ubans); err != nil {
			return nil
		}

		//This func return a Reconcile request array
		var request []reconcile.Request
		for _, deploy := range ubans.Items {
			deploymentName := func() string {
				var name string
				if deploy.Spec.DeploymentName == "" {
					name = fmt.Sprintf("%s-missing-deployment-name", deploy.Name)
				} else {
					name = fmt.Sprintf("%s-%s-depl", deploy.Name, deploy.Spec.DeploymentName)
				}
				return name
			}()

			//Find the deployment owned by Custom Resource
			if deploymentName == obj.GetName() && deploy.Namespace == obj.GetNamespace() {
				currentDeploy := &appsv1.Deployment{}
				if err := r.Get(context.Background(), types.NamespacedName{
					Namespace: obj.GetNamespace(),
					Name:      obj.GetName(),
				}, currentDeploy); err != nil {
					// This case can happen if somehow deployment gets deleted by
					// Kubectl command. We need to append new reconcile request to array
					// to create desired number of deployment again.

					if errors.IsNotFound(err) {
						request = append(request, reconcile.Request{
							NamespacedName: types.NamespacedName{
								Namespace: deploy.Namespace,
								Name:      deploy.Name,
							},
						})
						continue
					} else {
						return nil
					}
				}
				// Only append to the reconcile request array if replica count miss match.
				if currentDeploy.Spec.Replicas != deploy.Spec.Replicas {
					request = append(request, reconcile.Request{
						NamespacedName: types.NamespacedName{
							Namespace: deploy.Namespace,
							Name:      deploy.Name,
						},
					})
				}
			}
		}
		return request
	})

	return ctrl.NewControllerManagedBy(mgr).
		For(&salehdevv1alpha1.Uban{}).
		Watches(&appsv1.Deployment{}, handlerForDeployment).
		Owns(&corev1.Service{}).
		Complete(r)
}

// SetupWithManager sets up the controller with the Manager.
/*func (r *UbanReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&salehdevv1alpha1.Uban{}).
		Owns(&appsv1.Deployment{}).
		Owns(&corev1.Service{}).
		Complete(r)
}
*/
