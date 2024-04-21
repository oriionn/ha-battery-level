package main

import (
	"fmt"
	"github.com/getlantern/systray"
	"io/ioutil"
	"os/exec"
	"runtime"
	"strconv"
	"strings"
	"time"
)

type BatteryInfo struct {
	Level      float64
	IsCharging bool
}

func GetBatteryInfo() (BatteryInfo, error) {
	var info BatteryInfo
	var cmd *exec.Cmd
	var output []byte
	var err error

	switch os := runtime.GOOS; os {
	case "linux":
		cmd = exec.Command("acpi", "-b")
		output, err = cmd.Output()
		if err != nil {
			return info, err
		}
		outputStr := strings.TrimSpace(string(output))
		batteryInfo := strings.Fields(outputStr)
		if len(batteryInfo) < 4 {
			return info, fmt.Errorf("unable to retrieve battery info")
		}
		levelStr := strings.TrimSuffix(strings.TrimSuffix(batteryInfo[3], ","), "%")
		info.Level, err = strconv.ParseFloat(levelStr, 64)
		if err != nil {
			return info, err
		}
		info.IsCharging = strings.Contains(strings.ToLower(batteryInfo[2]), "charging")
	case "darwin":
		cmd = exec.Command("pmset", "-g", "batt")
		output, err = cmd.Output()
		if err != nil {
			return info, err
		}
		outputStr := strings.TrimSpace(string(output))
		lines := strings.Split(outputStr, "\n")
		if len(lines) < 2 {
			return info, fmt.Errorf("unable to retrieve battery info")
		}
		levelStr := strings.TrimSpace(strings.Split(lines[1], ";")[0])
		info.Level, err = strconv.ParseFloat(levelStr, 64)
		if err != nil {
			return info, err
		}
		info.IsCharging = strings.Contains(strings.ToLower(lines[0]), "charging")
	case "windows":
		cmd = exec.Command("powershell", "Get-WmiObject", "Win32_Battery")
		output, err = cmd.Output()
		if err != nil {
			return info, err
		}
		outputStr := strings.TrimSpace(string(output))
		lines := strings.Split(outputStr, "\n")
		if len(lines) < 2 {
			return info, fmt.Errorf("unable to retrieve battery info")
		}
		for _, line := range lines {
			if strings.Contains(line, "BatteryStatus") {
				fields := strings.Fields(line)
				if len(fields) >= 2 {
					statusStr := fields[2]
					status, err := strconv.Atoi(statusStr)
					if err != nil {
						return info, err
					}
					info.IsCharging = status == 2
				}
			} else if strings.Contains(line, "EstimatedChargeRemaining") {
				fields := strings.Fields(line)
				if len(fields) >= 2 {
					levelStr := fields[2]
					info.Level, err = strconv.ParseFloat(levelStr, 64)
					if err != nil {
						return info, err
					}
				}
			}
		}
	default:
		return info, fmt.Errorf("unsupported operating system")
	}

	return info, nil
}

func main() {
	go func() {
		for {
			info, err := GetBatteryInfo()
			if err != nil {
				fmt.Println("Error:", err)
				return
			}
			fmt.Printf("Battery level: %.2f%%\n", info.Level)
			if info.IsCharging {
				fmt.Println("Battery is charging.")
			} else {
				fmt.Println("Battery is not charging.")
			}
			time.Sleep(5 * time.Second)
		}
	}()

	if runtime.GOOS == "windows" || runtime.GOOS == "darwin" {
		systray.Run(onReady, onExit)
	} else {
		fmt.Println("Unsupported OS for Tray Icon")
	}
}

func onReady() {
	systray.SetIcon(getIcon("icons/icon.ico"))
	systray.SetTitle("Home Assistant - Battery Monitor")
	systray.SetTooltip("Have your battery on Home Assistant")
	mQuit := systray.AddMenuItem("Quit", "Quit the app")

	go func() {
		<-mQuit.ClickedCh
		systray.Quit()
	}()
}

func onExit() {
	// clean up here
}

func getIcon(s string) []byte {
	b, err := ioutil.ReadFile(s)
	if err != nil {
		fmt.Print(err)
	}
	return b
}
