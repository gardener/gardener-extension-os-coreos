package app

import (
	"errors"
	"fmt"
	"os"

	extensionscmdcontroller "github.com/gardener/gardener/extensions/pkg/controller/cmd"
	"github.com/gardener/gardener/extensions/pkg/controller/heartbeat"
	heartbeatcmd "github.com/gardener/gardener/extensions/pkg/controller/heartbeat/cmd"
	osccontroller "github.com/gardener/gardener/extensions/pkg/controller/operatingsystemconfig"
	"github.com/spf13/pflag"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"

	configv1alpha1 "github.com/gardener/gardener-extension-os-coreos/pkg/controller/config/v1alpha1"
	"github.com/gardener/gardener-extension-os-coreos/pkg/controller/config/v1alpha1/validation"
	"github.com/gardener/gardener-extension-os-coreos/pkg/controller/operatingsystemconfig"
)

// Options holds configuration passed to the service controller.
type Options struct {
	generalOptions     *extensionscmdcontroller.GeneralOptions
	extensionOptions   *ExtensionOptions
	restOptions        *extensionscmdcontroller.RESTOptions
	managerOptions     *extensionscmdcontroller.ManagerOptions
	controllerOptions  *extensionscmdcontroller.ControllerOptions
	heartbeatOptions   *heartbeatcmd.Options
	controllerSwitches *extensionscmdcontroller.SwitchOptions
	reconcileOptions   *extensionscmdcontroller.ReconcilerOptions
	optionAggregator   extensionscmdcontroller.OptionAggregator
}

// ExtensionOptions holds options related to the extension (not the extension controller)
type ExtensionOptions struct {
	configFile string
	Config     *configv1alpha1.ExtensionConfig
}

var configDecoder runtime.Decoder

func init() {
	configScheme := runtime.NewScheme()
	schemeBuilder := runtime.NewSchemeBuilder(
		configv1alpha1.AddToScheme,
	)
	utilruntime.Must(schemeBuilder.AddToScheme(configScheme))
	configDecoder = serializer.NewCodecFactory(configScheme).UniversalDecoder()
}

// AddFlags implements Flagger.AddFlags.
func (o *ExtensionOptions) AddFlags(fs *pflag.FlagSet) {
	fs.StringVar(&o.configFile, "config", o.configFile, "Path to configuration file.")
}

// Complete implements Completer.Complete.
func (o *ExtensionOptions) Complete() error {
	if len(o.configFile) == 0 {
		return errors.New("missing config file")
	}

	data, err := os.ReadFile(o.configFile)
	if err != nil {
		return fmt.Errorf("error reading config file: %w", err)
	}

	o.Config = &configv1alpha1.ExtensionConfig{}
	if err = runtime.DecodeInto(configDecoder, data, o.Config); err != nil {
		return fmt.Errorf("error decoding config: %w", err)
	}

	return nil
}

func (o *ExtensionOptions) Completed() *ExtensionOptions {
	return o
}

// Apply applies the ExtensionOptions to the passed ControllerConfig instance.
func (o *ExtensionOptions) Apply(config *operatingsystemconfig.Config) {
	config.ExtensionConfig = o.Config
}

func (o *ExtensionOptions) Validate() error {
	if errs := validation.ValidateExtensionConfig(o.Config); len(errs) > 0 {
		return fmt.Errorf("invalid extension config: %w", errs.ToAggregate())
	}

	return nil
}

// NewOptions creates a new Options instance.
func NewOptions() *Options {
	options := &Options{
		generalOptions: &extensionscmdcontroller.GeneralOptions{},
		restOptions:    &extensionscmdcontroller.RESTOptions{},
		managerOptions: &extensionscmdcontroller.ManagerOptions{
			LeaderElection:          true,
			LeaderElectionID:        extensionscmdcontroller.LeaderElectionNameID(Name),
			LeaderElectionNamespace: os.Getenv("LEADER_ELECTION_NAMESPACE"),
		},
		controllerOptions: &extensionscmdcontroller.ControllerOptions{
			MaxConcurrentReconciles: 5,
		},
		reconcileOptions: &extensionscmdcontroller.ReconcilerOptions{},
		controllerSwitches: extensionscmdcontroller.NewSwitchOptions(
			extensionscmdcontroller.Switch(osccontroller.ControllerName, operatingsystemconfig.AddToManager),
			extensionscmdcontroller.Switch(heartbeat.ControllerName, heartbeat.AddToManager),
		),
		heartbeatOptions: &heartbeatcmd.Options{
			ExtensionName:        Name,
			RenewIntervalSeconds: 30,
			Namespace:            os.Getenv("LEADER_ELECTION_NAMESPACE"),
		},
		extensionOptions: &ExtensionOptions{},
	}

	options.optionAggregator = extensionscmdcontroller.NewOptionAggregator(
		options.generalOptions,
		options.restOptions,
		options.managerOptions,
		options.controllerOptions,
		options.extensionOptions,
		extensionscmdcontroller.PrefixOption("heartbeat-", options.heartbeatOptions),
		options.controllerSwitches,
		options.reconcileOptions,
	)

	return options
}

func (o *Options) Validate() error {
	if err := o.extensionOptions.Validate(); err != nil {
		return err
	}
	if err := o.heartbeatOptions.Validate(); err != nil {
		return err
	}
	return nil
}

func (o *Options) Complete() error {
	return o.optionAggregator.Complete()
}
