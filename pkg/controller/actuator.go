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

package controller

import (
	"context"

	extensionsv1alpha1 "github.com/gardener/gardener/pkg/apis/extensions/v1alpha1"
)

// OperatingSystemConfigActuator acts upon OperatingSystemConfig resources.
type OperatingSystemConfigActuator interface {
	// Create the operating system config.
	Create(ctx context.Context, config *extensionsv1alpha1.OperatingSystemConfig) error
	// Delete the operating system config.
	Delete(ctx context.Context, config *extensionsv1alpha1.OperatingSystemConfig) error
	// Update the operating system config.
	Update(ctx context.Context, config *extensionsv1alpha1.OperatingSystemConfig) error
	// Exists checks whether the given config currently exists.
	Exists(ctx context.Context, config *extensionsv1alpha1.OperatingSystemConfig) (bool, error)
}
