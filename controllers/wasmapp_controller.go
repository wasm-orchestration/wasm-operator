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

package controllers

import (
	"context"
	"fmt"
	operatorv1alpha1 "github.com/korvoj/kube-spin/api/v1alpha1"
	"github.com/korvoj/kube-spin/assets"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

// WasmAppReconciler reconciles a WasmApp object
type WasmAppReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

//+kubebuilder:rbac:groups=operator.kube-spin.mrezhi.net,resources=wasmapps,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=operator.kube-spin.mrezhi.net,resources=wasmapps/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=operator.kube-spin.mrezhi.net,resources=wasmapps/finalizers,verbs=update
//+kubebuilder:rbac:groups=apps,resources=deployments,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=networking.k8s.io,resources=ingresses,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups="",resources=services,verbs=get;list;watch;create;update;patch;delete

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the WasmApp object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.13.0/pkg/reconcile
func (r *WasmAppReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)
	// Try to obtain the WasmApp resource
	operatorCR := &operatorv1alpha1.WasmApp{}
	err := r.Get(ctx, req.NamespacedName, operatorCR)
	// The resource has been deleted
	if err != nil && errors.IsNotFound(err) {
		//TODO: The resource has been deleted, delete other dependant objects as well
		logger.Info("Operator resource object not found.")
		return ctrl.Result{}, nil
	} else if err != nil {
		logger.Error(err, "Error getting operator resource object.")
		return ctrl.Result{}, err
	}

	// Get the relevant Deployment for the CRD, so we can update its fields
	deployment := &appsv1.Deployment{}
	service := &corev1.Service{}
	ingress := &networkingv1.Ingress{}

	createDeployment := false
	createService := false
	createIngress := false

	// Check existence of deployment
	err = r.Get(ctx, req.NamespacedName, deployment)
	// The deployment does not exist, so it needs to be created
	if err != nil && errors.IsNotFound(err) {
		createDeployment = true
		deployment = assets.GetDeploymentFromFile("manifests/deployment.yaml")
	} else if err != nil {
		logger.Error(err, "Error getting existing deployment.")
		return ctrl.Result{}, err
	}

	// Check existence of service
	err = r.Get(ctx, req.NamespacedName, service)
	// The service does not exist, so it needs to be created
	if err != nil && errors.IsNotFound(err) {
		createService = true
		service = assets.GetServiceFromFile("manifests/service.yaml")
	} else if err != nil {
		logger.Error(err, "Error getting existing service.")
		return ctrl.Result{}, err
	}

	// Check existence of ingress
	if *operatorCR.Spec.IngressEnabled {
		err = r.Get(ctx, req.NamespacedName, ingress)
		// The ingress does not exist, so it needs to be created
		if err != nil && errors.IsNotFound(err) {
			createIngress = true
			ingress = assets.GetIngressFromFile("manifests/ingress.yaml")
		} else if err != nil {
			logger.Error(err, "Error getting existing ingress.")
			return ctrl.Result{}, err
		}
	}

	// WasmApp should be listed as the OwnerReference of the Deployment
	r.prepareDeployment(deployment, operatorCR, &req)
	r.prepareService(service, operatorCR, &req)
	if *operatorCR.Spec.IngressEnabled {
		r.prepareIngress(ingress, operatorCR, &req)
	}

	_ = ctrl.SetControllerReference(operatorCR, deployment, r.Scheme)
	_ = ctrl.SetControllerReference(operatorCR, service, r.Scheme)
	if *operatorCR.Spec.IngressEnabled {
		_ = ctrl.SetControllerReference(operatorCR, ingress, r.Scheme)
	}

	if createService {
		err = r.Create(ctx, service)
	} else {
		err = r.Update(ctx, service)
	}
	if *operatorCR.Spec.IngressEnabled && createIngress {
		err = r.Create(ctx, ingress)
	} else if *operatorCR.Spec.IngressEnabled {
		err = r.Update(ctx, ingress)
	}
	if createDeployment {
		err = r.Create(ctx, deployment)
	} else {
		err = r.Update(ctx, deployment)
	}

	if errors.IsConflict(err) {
		err = nil
	}

	return ctrl.Result{}, err
}

// SetupWithManager sets up the controller with the Manager, and defines what events will trigger the reconciliation loop.
func (r *WasmAppReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&operatorv1alpha1.WasmApp{}).
		Owns(&appsv1.Deployment{}). // recreate the relevant deployment if it has been deleted
		Complete(r)
}

func (r *WasmAppReconciler) prepareDeployment(deployment *appsv1.Deployment, operatorCR *operatorv1alpha1.WasmApp, req *ctrl.Request) {
	// Replicas is an optional field
	if operatorCR.Spec.Replicas != nil {
		deployment.Spec.Replicas = operatorCR.Spec.Replicas
	}

	// RuntimeClass is an optional field
	if operatorCR.Spec.RuntimeClass != "" {
		deployment.Spec.Template.Spec.RuntimeClassName = &operatorCR.Spec.RuntimeClass
	}

	if operatorCR.Spec.ImagePullSecret != "" {
		deployment.Spec.Template.Spec.ImagePullSecrets[0] = corev1.LocalObjectReference{Name: operatorCR.Spec.ImagePullSecret}
	}

	// Image name and tag are mandatory fields
	deployment.Spec.Template.Spec.Containers[0].Image = operatorCR.Spec.OciImage + ":" + operatorCR.Spec.OciImageTag
	deployment.Name = req.Name
	deployment.Namespace = req.Namespace

	// set annotations for the managed Pods
	if deployment.Spec.Template.Annotations == nil {
		annotations := map[string]string{}
		deployment.Spec.Template.Annotations = annotations
	}
	deployment.Spec.Template.Annotations["kube-spin.mrezhi.net/wasm"] = "true"
	deployment.Spec.Template.Annotations["kube-spin.mrezhi.net/identifier"] = req.Name

	// set labels for the managed Pods
	labels := map[string]string{}
	deployment.Spec.Template.Labels = labels
	deployment.Spec.Template.Labels["app"] = req.Name

	// match labels for the Pods
	deployment.Spec.Selector.MatchLabels = deployment.Spec.Template.Labels

	// set container name
	deployment.Spec.Template.Spec.Containers[0].Name = req.Name
}

func (r *WasmAppReconciler) prepareService(service *corev1.Service, operatorCR *operatorv1alpha1.WasmApp, req *ctrl.Request) {
	service.Name = req.Name
	service.Namespace = req.Namespace

	selector := map[string]string{}
	service.Spec.Selector = selector
	service.Spec.Selector["app"] = req.Name
}

func (r *WasmAppReconciler) prepareIngress(ingress *networkingv1.Ingress, operatorCR *operatorv1alpha1.WasmApp, req *ctrl.Request) {
	defaultIngressClass := "nginx"
	ingress.Name = req.Name
	ingress.Namespace = req.Namespace
	ingress.Spec.Rules[0].HTTP.Paths[0].Backend.Service.Name = req.Name

	if operatorCR.Spec.IngressClass != "" {
		ingress.Spec.IngressClassName = &operatorCR.Spec.IngressClass
	} else {
		ingress.Spec.IngressClassName = &defaultIngressClass
	}

	if operatorCR.Spec.IngressHost != "" {
		ingress.Spec.Rules[0].Host = operatorCR.Spec.IngressHost
	} else {
		ingress.Spec.Rules[0].Host = fmt.Sprintf("http://%s.example.local", req.Name)
	}

	if *operatorCR.Spec.IngressTlsEnabled {
		ingress.Spec.TLS[0].Hosts[0] = operatorCR.Spec.IngressHost
	} else {
		ingress.Spec.TLS = nil
	}
}
