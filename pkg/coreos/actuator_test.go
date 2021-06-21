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

	"github.com/gardener/gardener-extension-os-coreos/pkg/coreos"

	"github.com/gardener/gardener/extensions/pkg/controller/operatingsystemconfig"
	extensionsv1alpha1 "github.com/gardener/gardener/pkg/apis/extensions/v1alpha1"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"k8s.io/utils/pointer"
)

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
					Permissions: pointer.Int32Ptr(0666),
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
			userData, _, _, err := actuator.Reconcile(context.TODO(), osc)
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
					TransmitUnencoded: pointer.BoolPtr(true),
					Inline: &extensionsv1alpha1.FileContentInline{
						Encoding: "b64",
						Data:     base64.StdEncoding.EncodeToString([]byte("bar")),
					},
				}})
			userData, _, _, err := actuator.Reconcile(context.TODO(), osc)
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
})
