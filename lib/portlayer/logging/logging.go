// Copyright 2016 VMware, Inc. All Rights Reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//    http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package logging

import (
	"context"
	"fmt"
	"sync"

	log "github.com/Sirupsen/logrus"

	"github.com/vmware/govmomi/object"
	"github.com/vmware/govmomi/vim25/types"
	"github.com/vmware/vic/lib/portlayer/event/collector/vsphere"
	"github.com/vmware/vic/lib/portlayer/event/events"
	"github.com/vmware/vic/lib/portlayer/exec"
	"github.com/vmware/vic/pkg/retry"
	"github.com/vmware/vic/pkg/trace"
)

var once sync.Once

func Init(ctx context.Context) error {
	once.Do(func() {
		// Subscribe to vm events
		exec.Config.EventManager.Subscribe(
			events.NewEventType(vsphere.VMEvent{}).Topic(),
			"logging",
			func(ie events.Event) {
				eventCallback(ctx, ie)
			})
	})
	return nil
}

// listens migrated events and connects the file backed serial ports
func eventCallback(ctx context.Context, ie events.Event) {
	defer trace.End(trace.Begin(""))

	switch ie.String() {
	case events.ContainerMigrated,
		events.ContainerMigratedByDrs:
		log.Debugf("Received %s event", ie)

		// grab the container from the cache
		container := exec.Containers.Container(ie.Reference())
		if container == nil {
			log.Errorf("Container %s not found", ie.Reference())
			return
		}

		operation := func() error {
			var err error

			handle := container.NewHandle(ctx)
			if handle == nil {
				err = fmt.Errorf("Handle for %s cannot be created", ie.Reference())
				log.Error(err)
				return err
			}
			defer handle.Close()

			// set them to true
			if handle, err = toggle(handle, true); err != nil {
				log.Errorf("Failed to toggle logging after %s event for container %s: %s", ie, ie.Reference(), err)
				return err
			}

			if err = handle.Commit(ctx, nil, nil); err != nil {
				log.Errorf("Failed to commit handle after getting %s event for container %s: %s", ie, ie.Reference(), err)
				return err
			}
			return nil
		}

		if err := retry.Do(operation, exec.IsConcurrentAccessError); err != nil {
			log.Errorf("Multiple attempts failed to commit handle after getting %s event for container %s: %s", ie, ie.Reference(), err)
		}
	}

}

func toggle(handle *exec.Handle, connected bool) (*exec.Handle, error) {
	defer trace.End(trace.Begin(""))

	// get the virtual device list
	devices := object.VirtualDeviceList(handle.Config.Hardware.Device)

	// select the virtual serial ports
	serials := devices.SelectByBackingInfo((*types.VirtualSerialPortFileBackingInfo)(nil))
	if len(serials) == 0 {
		return nil, fmt.Errorf("Unable to find a device with desired backing")
	}

	for i := range serials {
		serial := serials[i]

		log.Debugf("Found a device with desired backing: %#v", serial)

		c := serial.GetVirtualDevice().Connectable
		if c.Connected == connected {
			log.Debugf("Already in the desired state (connected: %t)", connected)
			continue
		}
		log.Debugf("Setting Connected to %t", connected)
		c.Connected = connected

		config := &types.VirtualDeviceConfigSpec{
			Device:    serial,
			Operation: types.VirtualDeviceConfigSpecOperationEdit,
		}
		handle.Spec.DeviceChange = append(handle.Spec.DeviceChange, config)
	}
	return handle, nil
}

// Join adds two file backed serial port and configures them
func Join(h interface{}) (interface{}, error) {
	defer trace.End(trace.Begin(""))

	handle, ok := h.(*exec.Handle)
	if !ok {
		return nil, fmt.Errorf("Type assertion failed for %#+v", handle)
	}

	VMPathName := handle.Spec.VMPathName()
	VMName := handle.Spec.Spec().Name

	for _, logFile := range []string{"tether.debug", "output.log"} {
		filename := fmt.Sprintf("%s/%s/%s", VMPathName, VMName, logFile)

		// Debug and log serial ports - backed by datastore file
		serial := &types.VirtualSerialPort{
			VirtualDevice: types.VirtualDevice{
				Backing: &types.VirtualSerialPortFileBackingInfo{
					VirtualDeviceFileBackingInfo: types.VirtualDeviceFileBackingInfo{
						FileName: filename,
					},
				},
				Connectable: &types.VirtualDeviceConnectInfo{
					Connected:         true,
					StartConnected:    true,
					AllowGuestControl: true,
				},
			},
			YieldOnPoll: true,
		}
		config := &types.VirtualDeviceConfigSpec{
			Device:    serial,
			Operation: types.VirtualDeviceConfigSpecOperationAdd,
		}
		handle.Spec.DeviceChange = append(handle.Spec.DeviceChange, config)
	}

	return handle, nil
}

// Bind sets the *Connected fields of the VirtualSerialPort
func Bind(h interface{}) (interface{}, error) {
	defer trace.End(trace.Begin(""))

	handle, ok := h.(*exec.Handle)
	if !ok {
		return nil, fmt.Errorf("Type assertion failed for %#+v", handle)
	}
	return toggle(handle, true)
}

// Unbind unsets the *Connected fields of the VirtualSerialPort
func Unbind(h interface{}) (interface{}, error) {
	defer trace.End(trace.Begin(""))

	handle, ok := h.(*exec.Handle)
	if !ok {
		return nil, fmt.Errorf("Type assertion failed for %#+v", handle)
	}
	return toggle(handle, false)
}