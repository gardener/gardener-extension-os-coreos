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

package coreos

import (
	"context"
	_ "embed"
	"encoding/base64"
	"fmt"
	"strconv"

	actuatorutil "github.com/gardener/gardener/extensions/pkg/controller/operatingsystemconfig/oscommon/actuator"
	extensionsv1alpha1 "github.com/gardener/gardener/pkg/apis/extensions/v1alpha1"
)

var (
	coreOSCloudInitCommand = "/usr/bin/coreos-cloudinit --from-file="

	//go:embed templates/containerd/run-command.sh.tpl
	containerdTemplateContent string

	//go:embed templates/configure-cgroupsv2.sh.tpl
	cgroupsv2TemplateContent string
)

func (c *actuator) reconcile(ctx context.Context, config *extensionsv1alpha1.OperatingSystemConfig) ([]byte, *string, []string, []string, error) {
	cloudConfig, units, files, err := c.cloudConfigFromOperatingSystemConfig(ctx, config)
	if err != nil {
		return nil, nil, nil, nil, fmt.Errorf("could not generate cloud config: %v", err)
	}

	var command *string
	if path := config.Spec.ReloadConfigFilePath; path != nil {
		cmd := coreOSCloudInitCommand + *path
		command = &cmd
	}

	return []byte(cloudConfig), command, units, files, nil
}

func (c *actuator) cloudConfigFromOperatingSystemConfig(ctx context.Context, config *extensionsv1alpha1.OperatingSystemConfig) (string, []string, []string, error) {
	cloudConfig := &CloudConfig{
		CoreOS: Config{
			Update: Update{
				RebootStrategy: "off",
			},
			Units: []Unit{
				{
					Name:    "update-engine.service",
					Mask:    true,
					Command: "stop",
				},
				{
					Name:    "locksmithd.service",
					Mask:    true,
					Command: "stop",
				},
			},
		},
	}

	// blacklist sctp kernel module
	if config.Spec.Purpose == extensionsv1alpha1.OperatingSystemConfigPurposeReconcile {
		cloudConfig.WriteFiles = []File{
			{
				Encoding:           "b64",
				Content:            base64.StdEncoding.EncodeToString([]byte("install sctp /bin/true")),
				Owner:              "root",
				Path:               "/etc/modprobe.d/sctp.conf",
				RawFilePermissions: "0644",
			},
		}
	}

	unitNames := make([]string, 0, len(config.Spec.Units))
	for _, unit := range config.Spec.Units {
		unitNames = append(unitNames, unit.Name)

		u := Unit{Name: unit.Name}

		if unit.Command != nil {
			u.Command = *unit.Command
		}
		if unit.Enable != nil {
			u.Enable = *unit.Enable
		}
		if unit.Content != nil {
			u.Content = *unit.Content
		}

		for _, dropIn := range unit.DropIns {
			u.DropIns = append(u.DropIns, UnitDropIn{
				Name:    dropIn.Name,
				Content: dropIn.Content,
			})
		}

		cloudConfig.CoreOS.Units = append(cloudConfig.CoreOS.Units, u)
	}

	filePaths := make([]string, 0, len(config.Spec.Files))
	for _, file := range config.Spec.Files {
		filePaths = append(filePaths, file.Path)
		f := File{
			Path: file.Path,
		}

		permissions := extensionsv1alpha1.OperatingSystemConfigDefaultFilePermission
		if p := file.Permissions; p != nil {
			permissions = *p
		}
		f.RawFilePermissions = strconv.FormatInt(int64(permissions), 8)

		rawContent, err := actuatorutil.DataForFileContent(ctx, c.client, config.Namespace, &file.Content)
		if err != nil {
			return "", nil, nil, err
		}

		if file.Content.TransmitUnencoded != nil && *file.Content.TransmitUnencoded {
			f.Content = string(rawContent)
			f.Encoding = ""
		} else {
			f.Encoding = "b64"
			f.Content = base64.StdEncoding.EncodeToString(rawContent)
		}

		cloudConfig.WriteFiles = append(cloudConfig.WriteFiles, f)
	}

	if isContainerdEnabled(config.Spec.CRIConfig) && config.Spec.Purpose == extensionsv1alpha1.OperatingSystemConfigPurposeProvision {
		cloudConfig.CoreOS.Units = append(
			cloudConfig.CoreOS.Units,
			Unit{
				Name:    "run-command.service",
				Command: "start",
				Enable:  true,
				Content: `[Unit]
Description=Oneshot unit used to run a script on node start-up.
Before=containerd.service kubelet.service
[Service]
Type=oneshot
EnvironmentFile=/etc/environment
ExecStart=/opt/bin/run-command.sh
[Install]
WantedBy=containerd.service kubelet.service
`,
			})

		unitNames = append(unitNames, "run-command.service")

		cloudConfig.WriteFiles = append(
			cloudConfig.WriteFiles,
			File{
				Path:               "/etc/systemd/system/containerd.service.d/11-exec_config.conf",
				RawFilePermissions: "0644",
				Content: `[Service]
SyslogIdentifier=containerd
ExecStart=
ExecStart=/bin/bash -c 'PATH="/run/torcx/unpack/docker/bin:$PATH" /run/torcx/unpack/docker/bin/containerd --config /etc/containerd/config.toml'
`,
			},
			File{
				Path:               "/opt/bin/run-command.sh",
				RawFilePermissions: "0755",
				Content:            containerdTemplateContent,
			})

	}

	names, err := enableCGroupsV2(cloudConfig)
	if err != nil {
		return "", nil, nil, err
	}
	unitNames = append(unitNames, names...)

	data, err := cloudConfig.String()
	if err != nil {
		return "", nil, nil, err
	}

	return data, unitNames, filePaths, nil
}

func isContainerdEnabled(criConfig *extensionsv1alpha1.CRIConfig) bool {
	if criConfig == nil {
		return false
	}

	return criConfig.Name == extensionsv1alpha1.CRINameContainerD
}

func enableCGroupsV2(cloudConfig *CloudConfig) ([]string, error) {
	var additionalUnitNames []string

	cloudConfig.CoreOS.Units = append(
		cloudConfig.CoreOS.Units,
		Unit{
			Name:    "enable-cgroupsv2.service",
			Command: "start",
			Enable:  true,
			Content: `[Unit]
Description=Oneshot unit used to patch the kubelet config for cgroupsv2.
Before=containerd.service kubelet.service
[Service]
Type=oneshot
EnvironmentFile=/etc/environment
ExecStart=/opt/bin/configure-cgroupsv2.sh
[Install]
WantedBy=containerd.service kubelet.service
`,
		})
	additionalUnitNames = append(additionalUnitNames, "enable-cgroupsv2.service")

	cloudConfig.WriteFiles = append(
		cloudConfig.WriteFiles,
		File{
			Path:               "/opt/bin/configure-cgroupsv2.sh",
			RawFilePermissions: "0755",
			Content:            cgroupsv2TemplateContent,
		})

	return additionalUnitNames, nil
}
