package main

import (
	"bytes"
	"fmt"
	"github.com/getlantern/systray"
	"github.com/pelletier/go-toml"
	"io/ioutil"
	"math"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"syscall"
	"time"
)

type BatteryInfo struct {
	Level      float64
	IsCharging bool
}

func getParentConfigPath() string {
	switch oss := runtime.GOOS; oss {
	case "windows":
		return filepath.Join(os.Getenv("APPDATA"), "ha-battery-level")
	case "darwin":
		return filepath.Join(os.Getenv("HOME"), "Library", "Application Support", "ha-battery-level")
	case "linux":
		return filepath.Join(os.Getenv("HOME"), ".config", "ha-battery-level")
	default:
		return ""
	}
}

func getConfigPath() string {
	switch oss := runtime.GOOS; oss {
	case "windows":
		return filepath.Join(os.Getenv("APPDATA"), "ha-battery-level", "settings.toml")
	case "darwin":
		return filepath.Join(os.Getenv("HOME"), "Library", "Application Support", "ha-battery-level", "settings.toml")
	case "linux":
		return filepath.Join(os.Getenv("HOME"), ".config", "ha-battery-level", "settings.toml")
	default:
		return ""
	}
}

func getUserConfig() (map[string]interface{}, error) {
	configPath := getConfigPath()
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		parentConfigPath := getParentConfigPath()
		err := os.MkdirAll(parentConfigPath, 0755)
		if err != nil {
			return nil, err
		}
		err = ioutil.WriteFile(configPath, []byte("baseUrl=\ntoken=\nfriendlyName=\nsensor="), 0644)
		if err != nil {
			return nil, err
		}
		panic("Please configure the settings in the settings.toml file\n" + configPath)
	}
	configData, err := ioutil.ReadFile(configPath)
	if err != nil {
		return nil, err
	}
	config := make(map[string]interface{})
	if err := toml.Unmarshal(configData, &config); err != nil {
		return nil, err
	}
	return config, nil
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
		cmd := exec.Command("powershell", "Get-WmiObject", "Win32_Battery")
		cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}

		var outb, errb bytes.Buffer
		cmd.Stdout = &outb
		cmd.Stderr = &errb

		err := cmd.Run()
		if err != nil {
			return info, err
		}
		output := outb.Bytes()
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
		userConfig, err := getUserConfig()
		if err != nil {
			panic(err)
		}
		for {
			info, err := GetBatteryInfo()
			if err != nil {
				fmt.Println("Error:", err)
				return
			}
			if userConfig["baseUrl"] != nil && userConfig["token"] != nil && userConfig["friendlyName"] != nil && userConfig["sensor"] != nil {
				baseUrl := userConfig["baseUrl"].(string)
				token := userConfig["token"].(string)
				friendlyName := userConfig["friendlyName"].(string)
				sensor := userConfig["sensor"].(string)
				url := fmt.Sprintf("%s/api/states/%s", baseUrl, sensor)
				icon := "mdi:battery"
				if info.IsCharging {
					icon = "mdi:battery-charging"
				}
				icon = fmt.Sprintf("%s-%s", icon, strconv.Itoa(int(math.Round(info.Level/10)*10)))
				payload := fmt.Sprintf("{\"state\": \"%f\", \"attributes\": {\"unit_of_measurement\": \"%%\", \"friendly_name\": \"%s\", \"icon\": \"%s\"}}", info.Level, friendlyName, icon)
				req, err := http.NewRequest("POST", url, strings.NewReader(payload))
				if err != nil {
					fmt.Println("Error:", err)
					return
				}
				req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))
				req.Header.Set("Content-Type", "application/json")
				client := &http.Client{}
				resp, err := client.Do(req)
				if err != nil {
					fmt.Println("Error:", err)
					return
				}
				err = resp.Body.Close()
				if err != nil {
					return
				}
			} else {
				fmt.Println("Error: Config not found")
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
