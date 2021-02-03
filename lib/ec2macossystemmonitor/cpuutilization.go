package ec2macossystemmonitor

import (
	"fmt"
	"github.com/shirou/gopsutil/cpu"
	"strconv"
)

// RunningCpuUsage gathers the value expected for CloudWatch but allows long running measurement. This is intended for
// usage where repeated calls will take place.
func RunningCpuUsage() (s string, err error) {
	percent, err := cpu.Percent(0, false)
	if err != nil {
		return "", fmt.Errorf("ec2macossystemmonitor: error while getting cpu stats: %s", err)
	}
	return strconv.FormatFloat(percent[0], 'f', -1, 64), nil
}
