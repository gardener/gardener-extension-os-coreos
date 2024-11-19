// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package app

import (
	"context"
	"fmt"

	"github.com/gardener/gardener/extensions/pkg/controller"
	"github.com/gardener/gardener/extensions/pkg/controller/heartbeat"
	"github.com/gardener/gardener/extensions/pkg/util"
	extensionsv1alpha1 "github.com/gardener/gardener/pkg/apis/extensions/v1alpha1"
	"github.com/spf13/cobra"
	componentbaseconfig "k8s.io/component-base/config"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/manager"

	"github.com/gardener/gardener-extension-os-coreos/pkg/controller/operatingsystemconfig"
)

// Name is the name of the CoreOS controller.
const Name = "os-coreos"

// NewControllerCommand creates a new CoreOS controller command.
func NewControllerCommand() *cobra.Command {
	options := NewOptions()

	cmd := &cobra.Command{
		Use: "os-coreos-controller-manager",

		RunE: func(cmd *cobra.Command, args []string) error {
			if err := options.optionAggregator.Complete(); err != nil {
				return fmt.Errorf("error completing options: %s", err)
			}

			if err := options.Validate(); err != nil {
				return err
			}

			cmd.SilenceUsage = true
			return run(cmd.Context(), options)
		},
	}

	options.optionAggregator.AddFlags(cmd.Flags())

	return cmd
}

func run(ctx context.Context, o *Options) error {
	// TODO: Make these flags configurable via command line parameters or component config file.
	util.ApplyClientConnectionConfigurationToRESTConfig(&componentbaseconfig.ClientConnectionConfiguration{
		QPS:   100.0,
		Burst: 130,
	}, o.restOptions.Completed().Config)

	mgr, err := manager.New(o.restOptions.Completed().Config, o.managerOptions.Completed().Options())
	if err != nil {
		return fmt.Errorf("could not instantiate manager: %w", err)
	}

	if err := controller.AddToScheme(mgr.GetScheme()); err != nil {
		return fmt.Errorf("could not update manager scheme: %w", err)
	}

	ctrlConfig := o.extensionOptions.Completed()
	ctrlConfig.Apply(&operatingsystemconfig.DefaultAddOptions.ExtensionConfig)

	o.controllerOptions.Completed().Apply(&operatingsystemconfig.DefaultAddOptions.Controller)
	o.healthOptions.Completed().Apply(&heartbeat.DefaultAddOptions)

	o.reconcileOptions.Completed().Apply(&operatingsystemconfig.DefaultAddOptions.IgnoreOperationAnnotation, ptr.To(extensionsv1alpha1.ExtensionClassShoot))

	if err := o.controllerSwitches.Completed().AddToManager(ctx, mgr); err != nil {
		return fmt.Errorf("could not add controller to manager: %w", err)
	}

	if err := mgr.Start(ctx); err != nil {
		return fmt.Errorf("error running manager: %w", err)
	}

	return nil
}
