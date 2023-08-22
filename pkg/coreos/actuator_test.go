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

package coreos_test

import (
	"context"
	"encoding/base64"

	"github.com/gardener/gardener/extensions/pkg/controller/operatingsystemconfig"
	extensionsv1alpha1 "github.com/gardener/gardener/pkg/apis/extensions/v1alpha1"
	"github.com/go-logr/logr"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"k8s.io/utils/pointer"

	"github.com/gardener/gardener-extension-os-coreos/pkg/coreos"
)

var logger = logr.Discard()

var _ = Describe("CloudConfig", func() {
	var (
		cloudConfig *coreos.CloudConfig
		actuator    operatingsystemconfig.Actuator
		osc         *extensionsv1alpha1.OperatingSystemConfig
	)

	BeforeEach(func() {
		cloudConfig = &coreos.CloudConfig{}
		actuator = coreos.NewActuator()

		osc = &extensionsv1alpha1.OperatingSystemConfig{
			Spec: extensionsv1alpha1.OperatingSystemConfigSpec{
				Files: []extensionsv1alpha1.File{{
					Path:        "fooPath",
					Permissions: pointer.Int32(0666),
					Content: extensionsv1alpha1.FileContent{
						Inline: &extensionsv1alpha1.FileContentInline{
							Encoding: "b64",
							Data:     "YmFy",
						},
					},
				}},
			},
		}
	})

	Describe("#Files", func() {

		It("should add files to userData", func() {
			userData, _, _, _, err := actuator.Reconcile(context.TODO(), logger, osc)
			Expect(err).To(BeNil())

			expectedFiles := `write_files:
- encoding: b64
  content: YmFy
  path: fooPath
  permissions: "666"`
			actual := string(userData)
			Expect(actual).To(ContainSubstring(expectedFiles))
		})

		It("should return files with flag TransmitUnencoded", func() {
			osc.Spec.Files = append(osc.Spec.Files, extensionsv1alpha1.File{
				Path: "fooPath",
				Content: extensionsv1alpha1.FileContent{
					TransmitUnencoded: pointer.Bool(true),
					Inline: &extensionsv1alpha1.FileContentInline{
						Encoding: "b64",
						Data:     base64.StdEncoding.EncodeToString([]byte("bar")),
					},
				}})
			userData, _, _, _, err := actuator.Reconcile(context.TODO(), logger, osc)
			Expect(err).To(BeNil())

			expectedFiles := `write_files:
- encoding: b64
  content: YmFy
  path: fooPath
  permissions: "666"
- content: bar
  path: fooPath`
			actual := string(userData)
			Expect(actual).To(ContainSubstring(expectedFiles))
		})

	})

	Describe("#Containerd", func() {
		BeforeEach(func() {
			osc.Spec.Purpose = extensionsv1alpha1.OperatingSystemConfigPurposeProvision
			osc.Spec.CRIConfig = &extensionsv1alpha1.CRIConfig{
				Name: extensionsv1alpha1.CRINameContainerD,
			}
		})

		It("should add containerd files", func() {
			osc.Spec.Files = []extensionsv1alpha1.File{}

			userData, _, _, _, err := actuator.Reconcile(context.TODO(), logger, osc)
			Expect(err).To(BeNil())

			expectedFiles := `write_files:
- content: |
    [Service]
    SyslogIdentifier=containerd
    ExecStart=
    ExecStart=/bin/bash -c 'PATH="/run/torcx/unpack/docker/bin:$PATH" /run/torcx/unpack/docker/bin/containerd --config /etc/containerd/config.toml'
  path: /etc/systemd/system/containerd.service.d/11-exec_config.conf
  permissions: "0644"
- content: |
    #!/bin/bash

    CONTAINERD_CONFIG=/etc/containerd/config.toml

    ALTERNATE_LOGROTATE_PATH="/usr/bin/logrotate"

    # initialize default containerd config if does not exist
    if [ ! -s "$CONTAINERD_CONFIG" ]; then
        mkdir -p /etc/containerd/
        /run/torcx/unpack/docker/bin/containerd config default > "$CONTAINERD_CONFIG"
        chmod 0644 "$CONTAINERD_CONFIG"
    fi

    # if cgroups v2 are used, patch containerd configuration to use systemd cgroup driver
    if [[ -e /sys/fs/cgroup/cgroup.controllers ]]; then
        sed -i "s/SystemdCgroup *= *false/SystemdCgroup = true/" "$CONTAINERD_CONFIG"
    fi

    # provide kubelet with access to the containerd binaries in /run/torcx/unpack/docker/bin
    if [ ! -s /etc/systemd/system/kubelet.service.d/environment.conf ]; then
        mkdir -p /etc/systemd/system/kubelet.service.d/
        cat <<EOF | tee /etc/systemd/system/kubelet.service.d/environment.conf
    [Service]
    Environment="PATH=/run/torcx/unpack/docker/bin:$PATH"
    EOF
        chmod 0644 /etc/systemd/system/kubelet.service.d/environment.conf
        systemctl daemon-reload
    fi

    # some flatcar versions have logrotate at /usr/bin instead of /usr/sbin
    if [ -f "$ALTERNATE_LOGROTATE_PATH" ]; then
        sed -i "s;/usr/sbin/logrotate;$ALTERNATE_LOGROTATE_PATH;" /etc/systemd/system/containerd-logrotate.service
        systemctl daemon-reload
    fi
  path: /opt/bin/run-command.sh
  permissions: "0755"`
			actual := string(userData)
			Expect(actual).To(ContainSubstring(expectedFiles))
		})

		It("should add run-command unit", func() {
			userData, _, unitNames, _, err := actuator.Reconcile(context.TODO(), logger, osc)
			Expect(err).To(BeNil())

			expectedUnit :=
				`- name: run-command.service
    enable: true
    content: |
      [Unit]
      Description=Oneshot unit used to run a script on node start-up.
      Before=containerd.service kubelet.service
      [Service]
      Type=oneshot
      EnvironmentFile=/etc/environment
      ExecStart=/opt/bin/run-command.sh
      [Install]
      WantedBy=containerd.service kubelet.service
    command: start`

			Expect(unitNames).To(ConsistOf("run-command.service", "enable-cgroupsv2.service"))
			Expect(string(userData)).To(ContainSubstring(expectedUnit))

		})
	})

	Describe("#CGroupsV2", func() {
		BeforeEach(func() {
			osc.Spec.Purpose = extensionsv1alpha1.OperatingSystemConfigPurposeProvision
		})

		It("should contain script to patch kubelet config for CGroupsV2", func() {
			osc.Spec.Files = []extensionsv1alpha1.File{}

			userData, _, _, _, err := actuator.Reconcile(context.TODO(), logger, osc)
			Expect(err).To(BeNil())

			expectedFiles :=
				`write_files:
- content: |
    #!/bin/bash

    KUBELET_CONFIG=/var/lib/kubelet/config/kubelet

    if [[ -e /sys/fs/cgroup/cgroup.controllers ]]; then
            echo "CGroups V2 are used!"
            echo "=> Patch kubelet to use systemd as cgroup driver"
            sed -i "s/cgroupDriver: cgroupfs/cgroupDriver: systemd/" "$KUBELET_CONFIG"
    else
            echo "No CGroups V2 used by system"
    fi
  path: /opt/bin/configure-cgroupsv2.sh
  permissions: "0755"`

			actual := string(userData)
			Expect(actual).To(ContainSubstring(expectedFiles))
		})

		It("should add unit to enable cgroupsv2", func() {
			userData, _, unitNames, _, err := actuator.Reconcile(context.TODO(), logger, osc)
			Expect(err).To(BeNil())

			expectedUnit :=
				`- name: enable-cgroupsv2.service
    enable: true
    content: |
      [Unit]
      Description=Oneshot unit used to patch the kubelet config for cgroupsv2.
      Before=containerd.service kubelet.service
      [Service]
      Type=oneshot
      EnvironmentFile=/etc/environment
      ExecStart=/opt/bin/configure-cgroupsv2.sh
      [Install]
      WantedBy=containerd.service kubelet.service
    command: start`

			Expect(unitNames).To(ConsistOf("enable-cgroupsv2.service"))
			Expect(string(userData)).To(ContainSubstring(expectedUnit))

		})

	})

	Describe("#String", func() {
		It("should return the string representation with correct header", func() {
			cloudConfig.CoreOS = coreos.Config{
				Update: coreos.Update{
					RebootStrategy: "off",
				},
			}

			expected := `#cloud-config

coreos:
  update:
    reboot_strategy: "off"
`
			Expect(cloudConfig.String()).To(Equal(expected))
		})
	})

	Describe("#Filepaths", func() {
		BeforeEach(func() {
			content := extensionsv1alpha1.FileContent{
				Inline: &extensionsv1alpha1.FileContentInline{
					Encoding: "",
					Data:     "test",
				},
			}
			osc.Spec.Files = []extensionsv1alpha1.File{
				{Path: "foo", Content: content},
				{Path: "bar", Content: content},
				{Path: "baz", Content: content},
			}
		})
		It("should return file paths", func() {
			_, _, _, filePaths, err := actuator.Reconcile(context.TODO(), logger, osc)
			Expect(err).To(BeNil())
			Expect(filePaths).To(ConsistOf("foo", "bar", "baz"))
		})
	})
})
