//go:build (windows)

package main

import (
    "os"
    "os/exec"
    "time"
    "strconv"
    "syscall"
    "unsafe"
    "github.com/chen-keinan/go-simple-config/simple"
    "github.com/getlantern/systray"
    "github.com/go-toast/toast"
)

const (
    COLOR_DESKTOP  = 1
    CONFIG_FILE    = "config.json"
    ICON_FILE      = "icon.ico"
    INTERVAL       = 5 * time.Second
    MSG_SUCCESS    = "Hintergrund gesetzt!"
    MSG_FAILURE    = "Fehler beim Setzen des Hintergrunds!"
    MSG_ERR_CONFIG = "Fehler beim Laden der Konfiguration!"
    MSG_ERR_ICON   = "Fehler beim Laden des Icons!"
    MSG_ERR_COLOR  = "Ungültiger Farbcode in Konfigurationsdatei!"
)

var (
    user32           = syscall.NewLazyDLL("user32.dll")
    procSetSysColors = user32.NewProc("SetSysColors")
    procGetSysColor  = user32.NewProc("GetSysColor")
    config           = simple.New()
)

func main() {
    err := config.Load(CONFIG_FILE)
    
    if err != nil {
        showToast(MSG_ERR_CONFIG)
        os.Exit(1)
    }
    systray.Run(onReady, onExit)
}

func onReady() {
    systray.SetIcon(getIcon(ICON_FILE))
    systray.SetTooltip("Wallpaper Watch")

    mConfig := systray.AddMenuItem("Konfiguration", "Konfigurationsdatei bearbeiten")
    mQuit := systray.AddMenuItem("Beenden", "Wallpaper Watch Beenden")

    ticker := time.NewTicker(INTERVAL)
    
    configuredColor := convertHexColorToSysColor(config.GetStringValue("color"))

    // Launch goroutines to handle menu item clicks or to perform ticker for main operation
    go func() {
        for {
            select {
            case <-ticker.C:
                checkAndChange(configuredColor)
            case <-mConfig.ClickedCh:
                cmd := exec.Command("cmd.exe", "/C", "start", "notepad.exe", CONFIG_FILE)
                cmd.Run()
            case <-mQuit.ClickedCh:
                systray.Quit()
            }
        }
    }()
}

func checkAndChange(configuredColor uint32) {
    currentColor := getSysColor(COLOR_DESKTOP)

    if (currentColor != configuredColor) {
        if setSysColors(COLOR_DESKTOP, configuredColor) {
            showToast(MSG_SUCCESS)
        } else {
            showToast(MSG_FAILURE)
        }
    }
}

func onExit() {
    systray.Quit()
    os.Exit(0)
}

func showToast(message string) {
    notification := toast.Notification{
        AppID:   "Wallpaper Watch",
        Message: message,
    }

    notification.Push()
}

func convertHexColorToSysColor(hex string) uint32 {
    if len(hex) != 7 || hex[0] != '#' {
        showToast(MSG_ERR_COLOR)
        os.Exit(1)
    }

    // Parse the RGB components
    r, err := strconv.ParseUint(hex[1:3], 16, 8)
    if err != nil {
        return 0
    }
    g, err := strconv.ParseUint(hex[3:5], 16, 8)
    if err != nil {
        return 0
    }
    b, err := strconv.ParseUint(hex[5:7], 16, 8)
    if err != nil {
        return 0
    }

    // Combine the RGB values into a uint32 (little endian, see SetSysColors)
    return uint32(b)<<16 | uint32(g)<<8 | uint32(r)
}

func setSysColors(colorIndex int32, color uint32) bool {
    colorIndices := []int32{colorIndex}
    colors := []uint32{color}

    r1, _, _ := procSetSysColors.Call(uintptr(len(colorIndices)),
        uintptr(unsafe.Pointer(&colorIndices[0])),
        uintptr(unsafe.Pointer(&colors[0])))

    return r1 != 0
}

func getSysColor(colorIndex int32) uint32 {
    r1, _, _ := procGetSysColor.Call(uintptr(colorIndex))
    return uint32(r1)
}

func getIcon(path string) []byte {
    file, err := os.ReadFile(path)
    if err != nil {
        showToast(MSG_ERR_ICON)
    }
    return file
}