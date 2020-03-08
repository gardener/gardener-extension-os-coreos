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
	"github.com/gardener/gardener-extension-os-coreos/pkg/coreos"
	extensionsv1alpha1 "github.com/gardener/gardener/pkg/apis/extensions/v1alpha1"
	"github.com/gobuffalo/packr/v2"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("CloudConfig", func() {
	var cloudConfig *coreos.CloudConfig
	var box = packr.NewBox("./testfiles")

	BeforeEach(func() {
		cloudConfig = &coreos.CloudConfig{}
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

		It("should return the correct provision cloud-config", func() {
			configProvision := &extensionsv1alpha1.OperatingSystemConfig{
				Spec: extensionsv1alpha1.OperatingSystemConfigSpec{
					Purpose: "reconcile",
					CRIConfig: &extensionsv1alpha1.CRIConfig{ Name: extensionsv1alpha1.CRINameContainerD,},
					Units: []extensionsv1alpha1.Unit{{
						Name: "containerd.service",
					}},
				},

			}
			userData, _, _, err := coreos.NewActuator().Reconcile(nil, configProvision)
			Expect(err).NotTo(HaveOccurred())
			expectedCloudInit, err2 := box.Find("cloud-init-reconcile")
			Expect(err2).NotTo(HaveOccurred())
			Expect(userData).To(Equal(expectedCloudInit))
		})
	})
})
