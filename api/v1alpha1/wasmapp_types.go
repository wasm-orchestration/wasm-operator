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

package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// WasmAppSpec defines the desired state of WasmApp
type WasmAppSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// +kubebuilder:validation:Required
	Replicas *int32 `json:"replicas,omitempty"`
	// +kubebuilder:validation:Required
	OciImage string `json:"ociImage"`
	// +kubebuilder:validation:Required
	OciImageTag       string `json:"ociImageTag"`
	OutboundHttp      *bool  `json:"outboundHttp,omitempty"`
	ImagePullSecret   string `json:"imagePullSecret,omitempty"`
	RuntimeClass      string `json:"runtimeClass,omitempty"`
	IngressEnabled    *bool  `json:"ingressEnabled,omitempty"`
	IngressClass      string `json:"ingressClass,omitempty"`
	IngressHost       string `json:"ingressHost,omitempty"`
	IngressTlsEnabled *bool  `json:"ingressTlsEnabled,omitempty"`
}

// WasmAppStatus defines the observed state of WasmApp
type WasmAppStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// WasmApp is the Schema for the wasmapps API
type WasmApp struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   WasmAppSpec   `json:"spec,omitempty"`
	Status WasmAppStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// WasmAppList contains a list of WasmApp
type WasmAppList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []WasmApp `json:"items"`
}

func init() {
	SchemeBuilder.Register(&WasmApp{}, &WasmAppList{})
}
