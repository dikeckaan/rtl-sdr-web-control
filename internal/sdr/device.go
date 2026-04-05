package sdr

import (
	"fmt"
	"os/exec"
	"strings"
	"sync"
)

// DeviceLock ensures only one service uses the RTL-SDR device at a time.
type DeviceLock struct {
	mu    sync.Mutex
	owner string
}

func NewDeviceLock() *DeviceLock {
	return &DeviceLock{}
}

func (d *DeviceLock) Acquire(serviceID string) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	if d.owner != "" && d.owner != serviceID {
		return fmt.Errorf("device locked by %s", d.owner)
	}
	d.owner = serviceID
	return nil
}

func (d *DeviceLock) Release(serviceID string) {
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.owner == serviceID {
		d.owner = ""
	}
}

func (d *DeviceLock) Owner() string {
	d.mu.Lock()
	defer d.mu.Unlock()
	return d.owner
}

// DetectDevice checks if an RTL-SDR device is available.
func DetectDevice() (bool, string) {
	cmd := exec.Command("rtl_test", "-t")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return false, ""
	}
	out := string(output)
	if strings.Contains(out, "Found") {
		return true, strings.TrimSpace(out)
	}
	return false, ""
}
