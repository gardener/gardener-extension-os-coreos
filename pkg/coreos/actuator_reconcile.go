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
	"encoding/base64"
	"fmt"
	"strconv"

	actuatorUtil "github.com/gardener/gardener/extensions/pkg/controller/operatingsystemconfig/oscommon/actuator"
	extensionsv1alpha1 "github.com/gardener/gardener/pkg/apis/extensions/v1alpha1"
)

var coreOSCloudInitCommand = "/usr/bin/coreos-cloudinit --from-file="

func (c *actuator) reconcile(ctx context.Context, config *extensionsv1alpha1.OperatingSystemConfig) ([]byte, *string, []string, error) {
	cloudConfig, units, err := c.cloudConfigFromOperatingSystemConfig(ctx, config)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("could not generate cloud config: %v", err)
	}

	var command *string
	if path := config.Spec.ReloadConfigFilePath; path != nil {
		cmd := coreOSCloudInitCommand + *path
		command = &cmd
	}

	return []byte(cloudConfig), command, units, nil
}

func (c *actuator) cloudConfigFromOperatingSystemConfig(ctx context.Context, config *extensionsv1alpha1.OperatingSystemConfig) (string, []string, error) {
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

	for _, file := range config.Spec.Files {
		f := File{
			Path: file.Path,
		}

		permissions := extensionsv1alpha1.OperatingSystemConfigDefaultFilePermission
		if p := file.Permissions; p != nil {
			permissions = *p
		}
		f.RawFilePermissions = strconv.FormatInt(int64(permissions), 8)

		rawContent, err := actuatorUtil.DataForFileContent(ctx, c.client, config.Namespace, &file.Content)
		if err != nil {
			return "", nil, err
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

	data, err := cloudConfig.String()
	if err != nil {
		return "", nil, err
	}

	return data, unitNames, nil
}
