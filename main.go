package main

import (
	"fmt"
	"github.com/getlantern/systray"
	"io/ioutil"
	"os/exec"
	"runtime"
	"strings"
)

func main() {
	if runtime.GOOS == "windows" || runtime.GOOS == "darwin" {
		systray.Run(onReady, onExit)
	} else if runtime.GOOS == "linux" {
		// Exécute la commande pour vérifier les processus du gestionnaire de fenêtres
		out, err := exec.Command("ps", "-e").Output()
		if err != nil {
			fmt.Println("Erreur lors de l'exécution de la commande :", err)
			return
		}

		compatibility := false
		windowManagers := []string{"gnome-session", "kdeinit", "xfce4-session", "lxsession", "mate-session", "cinnamon", "unity", "peppermint", "lxqt-session", "fluxbox", "blackbox", "openbox"}
		for _, wm := range windowManagers {
			if strings.Contains(string(out), wm) {
				compatibility = true
			}
		}

		if compatibility {
			systray.Run(onReady, onExit)
		} else {
			fmt.Println("Unsupported OS for Tray Icon")
		}
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
