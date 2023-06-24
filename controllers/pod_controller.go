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
	"github.com/korvoj/kube-spin/util"
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"strconv"
	"strings"
)

// PodReconciler reconciles a Pod object
type PodReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

//+kubebuilder:rbac:groups=core,resources=pods,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=core,resources=pods/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=core,resources=pods/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.13.0/pkg/reconcile
func (r *PodReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	pod := &corev1.Pod{}
	err := r.Get(ctx, req.NamespacedName, pod)

	if err != nil && errors.IsNotFound(err) {
		logger.Info("Pod %s in [%s] deleted", req.Name, req.Namespace)
	} else if err != nil {
		logger.Error(err, "Error getting existing pod.")
		return ctrl.Result{}, err
	}

	_, ok := pod.Annotations["kube-spin.mrezhi.net/wasm"]
	if ok {
		if len(pod.Status.ContainerStatuses) > 0 && pod.Status.ContainerStatuses[0].State.Running != nil {
			containerId := pod.Status.ContainerStatuses[0].ContainerID
			containerId = strings.ReplaceAll(containerId, "containerd://", "")
			logger.Info("The container id is ", "id", fmt.Sprintf("%s", containerId))
			port, err := util.Rdc.Get(ctx, containerId).Result()

			if err != nil {
				logger.Info("Cannot get port from Redis for containerId", "containerId", containerId)
				return ctrl.Result{}, nil
			}
			service := &corev1.Service{}
			ingress := &networkingv1.Ingress{}
			identifier, ok := pod.Annotations["kube-spin.mrezhi.net/identifier"]
			if !ok {
				logger.Error(nil, "Identifier annotation not present")
				return ctrl.Result{}, nil
			}
			namespacedName := types.NamespacedName{Namespace: req.Namespace, Name: identifier}
			err = r.Get(ctx, namespacedName, service)
			if err != nil {
				logger.Error(err, "[Pod Watcher] Error getting existing service.")
				return ctrl.Result{}, err
			}
			err = r.Get(ctx, namespacedName, ingress)
			if err != nil {
				logger.Error(err, "[Pod Watcher] Error getting existing ingress.")
				return ctrl.Result{}, err
			}

			servicePort := corev1.ServicePort{}
			appProtocol := "http"
			servicePort.AppProtocol = &appProtocol
			servicePort.Name = "web"
			portInt, err := strconv.Atoi(port)
			if err != nil || portInt < 1 || portInt > 65535 {
				logger.Error(err, "Acquired port from Redis is not an integer or is too large")
				return ctrl.Result{}, err
			}
			servicePort.Port = int32(portInt)
			servicePort.Protocol = "TCP"
			servicePort.TargetPort = intstr.IntOrString{IntVal: int32(portInt)}
			service.Spec.Ports = []corev1.ServicePort{servicePort}

			if ingress.Spec.Rules != nil && len(ingress.Spec.Rules) > 0 && ingress.Spec.Rules[0].HTTP.Paths != nil && len(ingress.Spec.Rules[0].HTTP.Paths) > 0 {
				ingress.Spec.Rules[0].HTTP.Paths[0].Backend.Service.Port.Number = int32(portInt)
			}

			err = r.Update(ctx, service)
			if err != nil {
				return ctrl.Result{}, err
			}

			err = r.Update(ctx, ingress)
			if err != nil {
				return ctrl.Result{}, err
			}
		}
	}

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *PodReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&corev1.Pod{}).
		Complete(r)
}
