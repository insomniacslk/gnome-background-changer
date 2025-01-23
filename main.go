package main

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"math/rand"
	"os"
	"os/exec"
	"path"
	"strings"
	"time"

	"github.com/getlantern/systray"
	"github.com/insomniacslk/editor"
	"github.com/insomniacslk/xjson"
	"github.com/kirsle/configdir"
)

const progname = "bgchanger"

var supportedExtensions = []string{"png", "jpg"}

//go:embed config.json.example
var exampleConfig []byte

func main() {
	rand.Seed(time.Now().UnixNano())
	configFile, cfg, err := loadConfig()
	if err != nil {
		log.Fatalf("Failed to open config file: %v", err)
	}
	systray.Run(
		func() { onReady(configFile, cfg) },
		onExit,
	)
}

// Config contains the program's configuration.
type Config struct {
	PicturesDir   string         `json:"pictures_dir"`
	Interval      xjson.Duration `json:"interval"`
	Editor        string         `json:"editor"`
	ChangeOnStart bool           `json:"change_on_start"`
}

func getRandomPicture(dirname string) (string, error) {
	files, err := ioutil.ReadDir(dirname)
	if err != nil {
		return "", fmt.Errorf("failed to read directory '%s': %w", dirname, err)
	}
	var pictures []string
	for _, f := range files {
		for _, ext := range supportedExtensions {
			if strings.HasSuffix(strings.ToLower(f.Name()), ext) {
				pictures = append(pictures, f.Name())
				break
			}
		}
	}
	if len(pictures) == 0 {
		return "", fmt.Errorf("no pictures found")
	}
	rand.Shuffle(len(pictures), func(i, j int) { pictures[i], pictures[j] = pictures[j], pictures[i] })
	return path.Join(dirname, pictures[0]), nil
}

func loadConfig() (string, *Config, error) {
	cfg := Config{}

	configPath := configdir.LocalConfig(progname)
	configFile := path.Join(configPath, "config.json")
	log.Printf("Trying to load config file %s", configFile)
	if err := configdir.MakePath(configPath); err != nil {
		if os.IsNotExist(err) {
			return configFile, &cfg, nil
		}
		return configFile, nil, fmt.Errorf("failed to create config path '%s': %w", configPath, err)
	}
	data, err := os.ReadFile(configFile)
	if err != nil {
		if os.IsNotExist(err) {
			// create a template file and open it with the default editor
			if err := editor.Open(configFile); err != nil {
				return configFile, &cfg, fmt.Errorf("failed to create config file: %w", err)
			}
			// re-read the config file
			data, err = os.ReadFile(configFile)
			if err != nil {
				return configFile, &cfg, fmt.Errorf("failed to read config file: %w", err)
			}
			// after this point, the newly created config file will be parsed by
			// the rest of this function.
		} else {
			return configFile, nil, fmt.Errorf("failed to read config file: %w", err)
		}
	}
	if err := json.Unmarshal(data, &cfg); err != nil {
		return configFile, nil, fmt.Errorf("failed to unmarshal config file: %w", err)
	}

	// sanity checks
	if cfg.PicturesDir == "" {
		return configFile, nil, fmt.Errorf("pictures_dir cannot be empty")
	}

	return configFile, &cfg, nil
}

func isDarkTheme() bool {
	out, err := exec.Command("gsettings", "get", "org.gnome.desktop.interface", "gtk-theme").Output()
	if err != nil {
		log.Printf("Warning: failed to get `gtk-theme` property: %v", err)
		return false
	}
	themeName := strings.Trim(
		strings.TrimSpace(string(out)),
		"'",
	)
	return strings.HasSuffix(themeName, "-dark")

}

func changeBG(cfg *Config) {
	filename, err := getRandomPicture(cfg.PicturesDir)
	if err != nil {
		log.Printf("Error: cannot get random picture: %v", err)
		return
	}
	gCmd := "picture-uri"
	if isDarkTheme() {
		gCmd = "picture-uri-dark"
	}
	cmd := exec.Command("gsettings", "set", "org.gnome.desktop.background", gCmd, "file://"+filename)
	log.Printf("Executing command: %s", cmd)
	cmd.Stdout, cmd.Stderr = os.Stdout, os.Stderr
	if err := cmd.Run(); err != nil {
		log.Printf("Error when changing background: %v", err)
	} else {
		log.Printf("Background changed to '%s'", filename)
	}
}

func onReady(configFile string, cfg *Config) {
	systray.SetIcon(Icon)
	//systray.SetTitle("RandBG")
	systray.SetTooltip("Change background randomly")
	mChange := systray.AddMenuItem("Change background now", "Change background with a randomly picked one from your configured directory")
	var mInterval *systray.MenuItem
	if cfg.Interval == 0 {
	} else {
		mInterval = systray.AddMenuItem(fmt.Sprintf("Background will change every %s", cfg.Interval), "The background will automatically change at the configured interval")
		mInterval.Disable()
	}
	mEdit := systray.AddMenuItem("Edit config", "Open configuration file for editing")
	mOpen := systray.AddMenuItem("Show backgrounds directory", "Show the directory containing the background images")
	mQuit := systray.AddMenuItem("Quit", "Quit the whole app")

	// Sets the icon of a menu item. Only available on Mac and Windows.
	mQuit.SetIcon(Icon)

	// sets the editor
	if cfg.Editor != "" {
		editor.Set(cfg.Editor)
	}
	if cfg.ChangeOnStart {
		changeBG(cfg)
	}

	go func() {
		var (
			timer       *time.Ticker
			ignoreTimer = false
		)
		if cfg.Interval > 0 {
			timer = time.NewTicker(time.Duration(cfg.Interval))
			log.Printf("Changing background picture every %s", cfg.Interval)
		} else {
			// a non-positive interval means "don't change background". This
			// creates a ticker with a valid time, but it will be ignored
			timer = time.NewTicker(time.Hour)
			ignoreTimer = true
		}
		for {
			select {
			case <-mQuit.ClickedCh:
				systray.Quit()
			case <-mEdit.ClickedCh:
				if err := editor.Open(configFile); err != nil {
					log.Printf("Error opening config file: %v", err)
				}
			case <-mOpen.ClickedCh:
				cmd := exec.Command("xdg-open", cfg.PicturesDir)
				cmd.Stdout, cmd.Stderr = os.Stdout, os.Stderr
				if err := cmd.Run(); err != nil {
					log.Printf("Error opening background directory: %v", err)
				}
			case <-mChange.ClickedCh:
				changeBG(cfg)
			case <-timer.C:
				if !ignoreTimer {
					changeBG(cfg)
				}
			}
		}
	}()
}

func onExit() {
	// clean up here
}
