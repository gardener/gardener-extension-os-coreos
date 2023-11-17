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

package operatingsystemconfig

import (
	"context"

	"github.com/gardener/gardener/extensions/pkg/controller/operatingsystemconfig"
	extensionsv1alpha1 "github.com/gardener/gardener/pkg/apis/extensions/v1alpha1"
	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/manager"
)

type actuator struct {
	client client.Client
	scheme *runtime.Scheme
	logger logr.Logger
}

// NewActuator creates a new Actuator that updates the status of the handled OperatingSystemConfigs.
func NewActuator(mgr manager.Manager) operatingsystemconfig.Actuator {
	return &actuator{
		client: mgr.GetClient(),
		scheme: mgr.GetScheme(),
		logger: log.Log.WithName("coreos-operatingsystemconfig-actuator"),
	}
}

func (c *actuator) Reconcile(ctx context.Context, _ logr.Logger, config *extensionsv1alpha1.OperatingSystemConfig) ([]byte, *string, []string, []string, error) {
	return c.reconcile(ctx, config)
}

func (c *actuator) Delete(ctx context.Context, _ logr.Logger, config *extensionsv1alpha1.OperatingSystemConfig) error {
	return c.delete(ctx, config)
}

func (c *actuator) ForceDelete(ctx context.Context, _ logr.Logger, config *extensionsv1alpha1.OperatingSystemConfig) error {
	return c.delete(ctx, config)
}

func (c *actuator) Restore(ctx context.Context, logger logr.Logger, config *extensionsv1alpha1.OperatingSystemConfig) ([]byte, *string, []string, []string, error) {
	return c.Reconcile(ctx, logger, config)
}

func (c *actuator) Migrate(ctx context.Context, _ logr.Logger, config *extensionsv1alpha1.OperatingSystemConfig) error {
	return nil
}
