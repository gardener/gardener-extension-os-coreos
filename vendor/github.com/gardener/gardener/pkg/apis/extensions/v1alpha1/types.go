// Copyright (c) 2019 SAP SE or an SAP affiliate company. All rights reserved. This file is licensed under the Apache Software License, v. 2 except as noted otherwise in the LICENSE file
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package v1alpha1

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// OperatingSystemConfig is a specification for a OperatingSystemConfig resource
type OperatingSystemConfig struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   OperatingSystemConfigSpec   `json:"spec"`
	Status OperatingSystemConfigStatus `json:"status"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// OperatingSystemConfigList is a list of OperatingSystemConfig resources
type OperatingSystemConfigList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata"`

	// Items is the list of OperatingSystemConfigs.
	Items []OperatingSystemConfig `json:"items"`
}

// OperatingSystemConfigSpec is the spec for a OperatingSystemConfig resource
type OperatingSystemConfigSpec struct {
	// DefaultSpec is a structure containing common fields used by all extension resources.
	DefaultSpec `json:",inline"`

	// Units is a list of unit for the operating system configuration (usually, a systemd unit).
	// +optional
	Units []Unit `json:"units,omitempty"`
	// Files is a list of files that should get written to the host's file system.
	// +optional
	Files []File `json:"files,omitempty"`
}

// Unit is a unit for the operating system configuration (usually, a systemd unit).
type Unit struct {
	// Name is the name of a unit.
	Name string `json:"name"`
	// Command is the unit's command.
	// +optional
	Command *string `json:"command,omitempty"`
	// Enable describes whether the unit is enabled or not.
	// +optional
	Enable *bool `json:"enable,omitempty"`
	// Content is the unit's content.
	// +optional
	Content *string `json:"content,omitempty"`
	// DropIns is a list of drop-ins for this unit.
	// +optional
	DropIns []DropIn `json:"dropIns,omitempty"`
}

// DropIn is a drop-in configuration for a systemd unit.
type DropIn struct {
	// Name is the name of the drop-in.
	Name string `json:"name"`
	// Content is the content of the drop-in.
	Content string `json:"content"`
}

// DefaultFilePermission is the default value for a permission of a file.
const DefaultFilePermission int32 = 0644

// File is a file that should get written to the host's file system. The content can either be inlined or
// referenced from a secret in the same namespace.
type File struct {
	// Path is the path of the file system where the file should get written to.
	Path string `json:"path"`
	// Permissions describes with which permissions the file should get written to the file system.
	// Should be defaulted to octal 0644.
	// +optional
	Permissions *int32 `json:"permissions,omitempty"`
	// Content describe the file's content.
	Content FileContent `json:"content"`
}

// FileContent can either reference a secret or contain inline configuration.
type FileContent struct {
	// SecretRef is a struct that contains information about the referenced secret.
	// +optional
	SecretRef *FileContentSecretRef `json:"secretRef,omitempty"`
	// Inline is a struct that contains information about the inlined data.
	// +optional
	Inline *FileContentInline `json:"inline,omitempty"`
}

// FileContentSecretRef contains keys for referencing a file content's data from a secret in the same namespace.
type FileContentSecretRef struct {
	// Name is the name of the secret.
	Name string `json:"name"`
	// DataKey is the key in the secret's `.data` field that should be read.
	DataKey string `json:"dataKey"`
}

// FileContentInline contains keys for inlining a file content's data and encoding.
type FileContentInline struct {
	// Encoding is the file's encoding (e.g. base64).
	Encoding string `json:"encoding"`
	// Data is the file's data.
	Data string `json:"data"`
}

// OperatingSystemConfigStatus is the status for a OperatingSystemConfig resource
type OperatingSystemConfigStatus struct {
	// DefaultStatus is a structure containing common fields used by all extension resources.
	DefaultStatus `json:",inline"`

	// CloudConfig is a structure for containing the generated output for the given operating system
	// config spec. It contains a reference to a secret as the result may contain confidential data.
	// +optional
	CloudConfig *CloudConfig `json:"cloudConfig,omitempty"`
}

// CloudConfig is a structure for containing the generated output for the given operating system
// config spec. It contains a reference to a secret as the result may contain confidential data.
type CloudConfig struct {
	// SecretRef is a reference to a secret that contains the actual result of the generated cloud config.
	SecretRef corev1.SecretReference `json:"secretRef"`
}
