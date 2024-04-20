package main

import (
	"fmt"
	"github.com/getlantern/systray"
	"io/ioutil"
	"runtime"
)

func main() {
	if runtime.GOOS == "windows" || runtime.GOOS == "darwin" {
		systray.Run(onReady, onExit)
	} else if runtime.GOOS == "linux" {
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
