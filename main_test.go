package main

import (
	"path/filepath"
	"runtime"
	"testing"
)

// Test_firstSerialDevice tests that the first character device in the list is
// selected and that missing paths and regular files are skipped.
func Test_firstSerialDevice(t *testing.T) {
	_, testFile, _, _ := runtime.Caller(0)
	missing := filepath.Join(t.TempDir(), "cu.missing")
	tests := []struct {
		name     string
		devPaths []string
		want     string
	}{
		{"Empty List", nil, ""},
		{"No Devices", []string{missing}, ""},
		{"Regular File Skipped", []string{testFile}, ""},
		{"Missing Skipped", []string{missing, "/dev/null"}, "/dev/null"},
		{"First Device Preferred", []string{"/dev/null", "/dev/zero"}, "/dev/null"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := firstSerialDevice(tt.devPaths); got != tt.want {
				t.Errorf("firstSerialDevice() = %q, want %q", got, tt.want)
			}
		})
	}
}

// Test_defaultSerialDevices tests that each instance generation's monitor
// serial device is in the default list, with the Apple Silicon devices ahead of
// the mac1.metal device.
func Test_defaultSerialDevices(t *testing.T) {
	index := make(map[string]int, len(defaultSerialDevices))
	for i, devPath := range defaultSerialDevices {
		index[devPath] = i
	}

	const mac1Device = "/dev/cu.pci-serial0"
	for _, devPath := range []string{
		"/dev/cu.pci-0000:4c:00.0,@00", // mac2 and later Apple Silicon
		"/dev/cu.pci-0000:49:00.0,@00", // mac-m4.metal
		mac1Device,
	} {
		if _, ok := index[devPath]; !ok {
			t.Errorf("defaultSerialDevices is missing %q", devPath)
		}
	}

	for devPath, i := range index {
		if devPath != mac1Device && i > index[mac1Device] {
			t.Errorf("%q is listed after the mac1.metal device %q", devPath, mac1Device)
		}
	}
}
