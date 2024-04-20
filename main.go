package main

import (
	"fmt"
	"github.com/distatus/battery"
	"github.com/getlantern/systray"
	"io/ioutil"
	"runtime"
	"time"
)

func main() {
	go func() {
		for {
			batteries, err := battery.GetAll()
			if err != nil {
				fmt.Println("Could not get battery info")
			} else {
				for i, battery := range batteries {
					fmt.Printf("Bat%d: ", i)
					fmt.Printf("state: %s, ", battery.State.String())
					fmt.Printf("current capacity: %f mWh, ", battery.Current)
					fmt.Printf("last full capacity: %f mWh, ", battery.Full)
					fmt.Printf("design capacity: %f mWh, ", battery.Design)
					fmt.Printf("charge rate: %f mW, ", battery.ChargeRate)
					fmt.Printf("voltage: %f V, ", battery.Voltage)
					fmt.Printf("design voltage: %f V\n", battery.DesignVoltage)
				}
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
