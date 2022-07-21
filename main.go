package main

import (
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
	"github.com/insomniacslk/xjson"
	"github.com/kirsle/configdir"
)

const progname = "bgchanger"

var supportedExtensions = []string{"png", "jpg"}

func main() {
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
	PicturesDir string         `json:"pictures_dir"`
	Interval    xjson.Duration `json:"interval"`
	Editor      string         `json:"editor"`
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
			return configFile, &cfg, nil
		}
		return configFile, nil, fmt.Errorf("failed to open '%s': %w", configFile, err)
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

func changeBG(cfg *Config) {
	filename, err := getRandomPicture(cfg.PicturesDir)
	if err != nil {
		log.Printf("Error: cannot get random picture: %v", err)
		return
	}
	cmd := exec.Command("gsettings", "set", "org.gnome.desktop.background", "picture-uri", "file://"+filename)
	cmd.Stdout, cmd.Stderr = os.Stdout, os.Stderr
	if err := cmd.Run(); err != nil {
		log.Printf("Error when changing background: %v", err)
	} else {
		log.Printf("Background changed to '%s'", filename)
	}
}

func onReady(configFile string, cfg *Config) {
	systray.SetIcon(Icon)
	systray.SetTitle("RandBG")
	systray.SetTooltip("Change background randomly")
	mChange := systray.AddMenuItem("Change background now", "Change background with a randomly picked one from your configured directory")
	var mInterval *systray.MenuItem
	if cfg.Interval == 0 {
	} else {
		mInterval = systray.AddMenuItem(fmt.Sprintf("Background will change every %s", cfg.Interval), "The background will automatically change at the configured interval")
	}
	mInterval.Disable()
	mEdit := systray.AddMenuItem("Edit config", "Open configuration file for editing")
	mQuit := systray.AddMenuItem("Quit", "Quit the whole app")

	// Sets the icon of a menu item. Only available on Mac and Windows.
	mQuit.SetIcon(Icon)

	// sets the editor
	editor := "xdg-open"
	if cfg.Editor != "" {
		editor = cfg.Editor
	}

	go func() {
		timer := time.NewTimer(time.Duration(cfg.Interval))
		log.Printf("Changing background picture every %s", cfg.Interval)
		for {
			select {
			case <-mQuit.ClickedCh:
				systray.Quit()
			case <-mEdit.ClickedCh:
				cmd := exec.Command(editor, configFile)
				log.Printf("Executing %v", cmd)
				cmd.Stdout, cmd.Stderr = os.Stdout, os.Stderr
				if err := cmd.Run(); err != nil {
					log.Printf("Error when opening editor: %v", err)
				}
			case <-mChange.ClickedCh:
				changeBG(cfg)
			case <-timer.C:
				changeBG(cfg)
			}
		}
	}()
}

func onExit() {
	// clean up here
}
