// -*- Mode: Go; indent-tabs-mode: t -*-
//
// Copyright (C) 2018-2019 IOTech Ltd
//
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"github.com/edgexfoundry/disposable-device"
	"github.com/edgexfoundry/disposable-device/driver"
	"github.com/edgexfoundry/device-sdk-go/pkg/startup"
)

const (
	serviceName string = "disposable-device"
)

func main() {
	d := driver.NewProtocolDriver()
	startup.Bootstrap(serviceName, disposable_device.Version, d)
}
