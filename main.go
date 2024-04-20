package main

import (
	"bufio"
	"fmt"
	"github.com/getlantern/systray"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"runtime"
	"strconv"
	"strings"
	"time"
)

type batteryStatus struct {
	connected  bool
	percentage int
	color      *string
}

func batteryCharge(battStat *batteryStatus) {
	switch runtime.GOOS {
	case "darwin": // MacOS
		if opts.GeneralOption.PmsetOn {
			acCmd := "pmset -g batt | grep -o 'AC Power'"
			cmd := exec.Command("sh", "-c", acCmd)
			cmd.Run()
			// Battery Connection
			if cmd.ProcessState.ExitCode() == 0 {
				battStat.connected = true
			} else {
				battStat.connected = false
			}
			// Battery Percentage
			battPrcCmd := "pmset -g batt | grep -o '[0-9]*%' | tr -d %"
			out, err := exec.Command("sh", "-c", battPrcCmd).Output()
			if err != nil {
				log.Fatal(err)
			}
			// Convert from byte to int
			battStat.percentage, err = strconv.Atoi(strings.TrimRight(string(out), "\n"))
			if err != nil {
				log.Fatal(err)
			}
		} else {
			// Battery Info by ioreg
			acCmd := "ioreg -n AppleSmartBattery -r | grep -o '\"[^\"]*\" = [^ ]*' | sed -e 's/= //g' -e 's/\"//g'"
			out, err := exec.Command("sh", "-c", acCmd).Output()
			if err != nil {
				log.Fatal(err)
			}
			ioregInfo := make(map[string]string, 50)
			scanner := bufio.NewScanner(strings.NewReader(string(out)))
			for scanner.Scan() {
				words := strings.Fields(scanner.Text())
				ioregInfo[words[0]] = words[1]
			}
			// Battery Connection
			if ioregInfo["ExternalConnected"] == "No" {
				battStat.connected = false
			} else {
				battStat.connected = true
			}

			// Battery Percentage
			maxCapacity, hasMaxCapacity := ioregInfo["MaxCapacity"]
			currentCapacity, hasCurrentCapacity := ioregInfo["CurrentCapacity"]
			if hasMaxCapacity && hasCurrentCapacity {
				currentCapacityInt, err := strconv.Atoi(currentCapacity)
				if err != nil {
					log.Fatal(err)
				}
				maxCapacityInt, err := strconv.Atoi(maxCapacity)
				if err != nil {
					log.Fatal(err)
				}
				battStat.percentage = 100 * currentCapacityInt / maxCapacityInt
			} else {
				log.Fatalf("failed to get battery capacity from ioreg")
				os.Exit(-1)
			}
		}
	case "linux":
		f, err := os.Open(opts.GeneralOption.BatteryPath + "/uevent")
		if err != nil {
			log.Fatal(err)
		}
		defer f.Close()

		uevent := make(map[string]string, 20)
		scanner := bufio.NewScanner(f)
		for scanner.Scan() {
			words := strings.SplitN(scanner.Text(), "=", 2)
			uevent[words[0]] = words[1]
		}

		// Battery Connection
		if uevent["POWER_SUPPLY_STATUS"] == "Discharging" {
			battStat.connected = false
		} else {
			battStat.connected = true
		}

		// Battery Percentage
		maxCapacity := uevent["POWER_SUPPLY_ENERGY_FULL"]
		currentCapacity := uevent["POWER_SUPPLY_ENERGY_NOW"]

		currentCapacityInt, err := strconv.Atoi(currentCapacity)
		if err != nil {
			log.Fatal(err)
		}
		maxCapacityInt, err := strconv.Atoi(maxCapacity)
		if err != nil {
			log.Fatal(err)
		}
		battStat.percentage = 100 * currentCapacityInt / maxCapacityInt
	default:
		log.Fatalf("this version does not yet support your OS")
		os.Exit(-1)
	}
}

func main() {
	go func() {
		for {
			battStat := batteryStatus{}
			batteryCharge(&battStat)
			fmt.Println("Battery Status: ", battStat.percentage)
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
