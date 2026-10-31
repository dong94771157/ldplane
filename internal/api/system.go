package api

import (
	"net/http"
	"runtime"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/shirou/gopsutil/v3/cpu"
	"github.com/shirou/gopsutil/v3/disk"
	"github.com/shirou/gopsutil/v3/host"
	"github.com/shirou/gopsutil/v3/load"
	"github.com/shirou/gopsutil/v3/mem"
	"github.com/shirou/gopsutil/v3/net"
)

// SystemOverview 系统概览（仪表盘数据）
func SystemOverview(c *gin.Context) {
	cpuPercent, _ := cpu.Percent(time.Second, false)
	vm, _ := mem.VirtualMemory()
	sm, _ := mem.SwapMemory()
	hostInfo, _ := host.Info()
	loadAvg, _ := load.Avg()
	partitions, _ := disk.Partitions(false)

	var disks []gin.H
	for _, p := range partitions {
		usage, err := disk.Usage(p.Mountpoint)
		if err != nil {
			continue
		}
		disks = append(disks, gin.H{
			"device":      p.Device,
			"mountpoint":  p.Mountpoint,
			"fstype":      p.Fstype,
			"total":       usage.Total,
			"used":        usage.Used,
			"free":        usage.Free,
			"used_percent": usage.UsedPercent,
		})
	}

	// 网络流量
	ioCounters, _ := net.IOCounters(false)
	var netInfo gin.H
	if len(ioCounters) > 0 {
		netInfo = gin.H{
			"bytes_sent":   ioCounters[0].BytesSent,
			"bytes_recv":   ioCounters[0].BytesRecv,
			"packets_sent": ioCounters[0].PacketsSent,
			"packets_recv": ioCounters[0].PacketsRecv,
		}
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "data": gin.H{
		"cpu": gin.H{
			"percent":   cpuPercent,
			"cores":     runtime.NumCPU(),
			"model":     getCPUModel(),
		},
		"memory": gin.H{
			"total":       vm.Total,
			"used":        vm.Used,
			"available":   vm.Available,
			"used_percent": vm.UsedPercent,
		},
		"swap": gin.H{
			"total":       sm.Total,
			"used":        sm.Used,
			"used_percent": sm.UsedPercent,
		},
		"load": gin.H{
			"load1":  loadAvg.Load1,
			"load5":  loadAvg.Load5,
			"load15": loadAvg.Load15,
		},
		"host": gin.H{
			"hostname":     hostInfo.Hostname,
			"os":           hostInfo.OS,
			"platform":     hostInfo.Platform,
			"kernel":       hostInfo.KernelVersion,
			"uptime":       hostInfo.Uptime,
			"boot_time":    hostInfo.BootTime,
		},
		"disks":    disks,
		"network":  netInfo,
		"time":     time.Now().Format("2006-01-02 15:04:05"),
	}})
}

// SystemInfo 系统详细信息
func SystemInfo(c *gin.Context) {
	info, _ := host.Info()
	c.JSON(http.StatusOK, gin.H{"code": 0, "data": gin.H{
		"hostname":        info.Hostname,
		"os":              info.OS,
		"platform":        info.Platform,
		"platform_family": info.PlatformFamily,
		"kernel":          info.KernelVersion,
		"kernel_arch":     info.KernelArch,
		"uptime":          info.Uptime,
		"boot_time":       info.BootTime,
		"procs":           info.Procs,
	}})
}

func getCPUModel() string {
	infos, err := cpu.Info()
	if err != nil || len(infos) == 0 {
		return "Unknown"
	}
	return infos[0].ModelName
}