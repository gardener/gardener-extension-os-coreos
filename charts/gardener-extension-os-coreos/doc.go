// SPDX-FileCopyrightText: Contributors to the Gardener project
//
// SPDX-License-Identifier: Apache-2.0

//go:generate sh -c "bash $GARDENER_HACK_DIR/generate-controller-registration.sh os-coreos . $(cat ../../VERSION) ../../example/controller-registration.yaml OperatingSystemConfig:coreos OperatingSystemConfig:flatcar OperatingSystemConfig:flatcar-alpha OperatingSystemConfig:flatcar-beta OperatingSystemConfig:flatcar-stable OperatingSystemConfig:flatcar-lts"

// Package chart enables go:generate support for generating the correct controller registration.
package chart
