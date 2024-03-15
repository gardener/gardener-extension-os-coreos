// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package app

import (
	"context"
	"fmt"
	"os"

	"github.com/gardener/gardener/extensions/pkg/controller"
	controllercmd "github.com/gardener/gardener/extensions/pkg/controller/cmd"
	"github.com/gardener/gardener/extensions/pkg/controller/heartbeat"
	heartbeatcmd "github.com/gardener/gardener/extensions/pkg/controller/heartbeat/cmd"
	osccontroller "github.com/gardener/gardener/extensions/pkg/controller/operatingsystemconfig"
	"github.com/gardener/gardener/extensions/pkg/util"
	"github.com/spf13/cobra"
	componentbaseconfig "k8s.io/component-base/config"
	"sigs.k8s.io/controller-runtime/pkg/manager"

	"github.com/gardener/gardener-extension-os-coreos/pkg/controller/operatingsystemconfig"
)

// Name is the name of the CoreOS controller.
const Name = "os-coreos"

// NewControllerCommand creates a new CoreOS controller command.
func NewControllerCommand(ctx context.Context) *cobra.Command {
	var (
		generalOpts = &controllercmd.GeneralOptions{}
		restOpts    = &controllercmd.RESTOptions{}
		mgrOpts     = &controllercmd.ManagerOptions{
			LeaderElection:          true,
			LeaderElectionID:        controllercmd.LeaderElectionNameID(Name),
			LeaderElectionNamespace: os.Getenv("LEADER_ELECTION_NAMESPACE"),
		}
		ctrlOpts = &controllercmd.ControllerOptions{
			MaxConcurrentReconciles: 5,
		}
		reconcileOpts = &controllercmd.ReconcilerOptions{}

		heartbeatCtrlOpts = &heartbeatcmd.Options{
			ExtensionName:        Name,
			RenewIntervalSeconds: 30,
			Namespace:            os.Getenv("LEADER_ELECTION_NAMESPACE"),
		}

		controllerSwitches = controllercmd.NewSwitchOptions(
			controllercmd.Switch(osccontroller.ControllerName, operatingsystemconfig.AddToManager),
			controllercmd.Switch(heartbeat.ControllerName, heartbeat.AddToManager),
		)

		aggOption = controllercmd.NewOptionAggregator(
			generalOpts,
			restOpts,
			mgrOpts,
			ctrlOpts,
			controllercmd.PrefixOption("heartbeat-", heartbeatCtrlOpts),
			reconcileOpts,
			controllerSwitches,
		)
	)

	cmd := &cobra.Command{
		Use: "os-coreos-controller-manager",

		RunE: func(cmd *cobra.Command, args []string) error {
			if err := aggOption.Complete(); err != nil {
				return fmt.Errorf("error completing options: %w", err)
			}

			if err := heartbeatCtrlOpts.Validate(); err != nil {
				return err
			}

			// TODO: Make these flags configurable via command line parameters or component config file.
			util.ApplyClientConnectionConfigurationToRESTConfig(&componentbaseconfig.ClientConnectionConfiguration{
				QPS:   100.0,
				Burst: 130,
			}, restOpts.Completed().Config)

			mgr, err := manager.New(restOpts.Completed().Config, mgrOpts.Completed().Options())
			if err != nil {
				return fmt.Errorf("could not instantiate manager: %w", err)
			}

			if err := controller.AddToScheme(mgr.GetScheme()); err != nil {
				return fmt.Errorf("could not update manager scheme: %w", err)
			}

			ctrlOpts.Completed().Apply(&operatingsystemconfig.DefaultAddOptions.Controller)
			heartbeatCtrlOpts.Completed().Apply(&heartbeat.DefaultAddOptions)

			reconcileOpts.Completed().Apply(&operatingsystemconfig.DefaultAddOptions.IgnoreOperationAnnotation)
			// TODO(rfranzke): Remove the UseGardenerNodeAgent fields as soon as the general options no longer support
			//  the GardenletUsesGardenerNodeAgent field.
			operatingsystemconfig.DefaultAddOptions.UseGardenerNodeAgent = generalOpts.Completed().GardenletUsesGardenerNodeAgent

			if err := controllerSwitches.Completed().AddToManager(ctx, mgr); err != nil {
				return fmt.Errorf("could not add controller to manager: %w", err)
			}

			if err := mgr.Start(ctx); err != nil {
				return fmt.Errorf("error running manager: %w", err)
			}

			return nil
		},
	}

	aggOption.AddFlags(cmd.Flags())

	return cmd
}
