// ─────────────────────────────────────────────────────────────────────────────
//  godotino :: config  —  persistent CLI configuration
//
//  Stored at:
//    Linux/macOS: ~/.config/godotino/config.json
//    Windows:     %APPDATA%\godotino\config.json
// ─────────────────────────────────────────────────────────────────────────────

package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"
)

// Config holds all persistent user-level settings.
type Config struct {
	// Core binary path (empty = search PATH)
	CoreBinary string `json:"core_binary" comment:"path to godotino-core binary"`

	// arduino-cli path (empty = search PATH)
	ArduinoCLI string `json:"arduino_cli" comment:"path to arduino-cli binary"`

	// Default board id
	DefaultBoard string `json:"default_board" comment:"default target board"`

	// Default baud rate for serial monitor
	DefaultBaud int `json:"default_baud" comment:"default serial baud rate"`

	// Color output
	Color bool `json:"color" comment:"enable colored output"`

	// Verbose build output
	Verbose bool `json:"verbose" comment:"verbose command output"`

	// Auto-detect board on flash/monitor
	AutoDetect bool `json:"auto_detect" comment:"auto-detect connected boards"`
}

// Default returns a Config with sensible defaults.
func Default() *Config {
	return &Config{
		CoreBinary:   "",
		ArduinoCLI:   "arduino-cli",
		DefaultBoard: "uno",
		DefaultBaud:  9600,
		Color:        true,
		Verbose:      false,
		AutoDetect:   true,
	}
}

// configPath returns the OS-appropriate config file path.
func configPath() (string, error) {
	var base string
	if xdg := os.Getenv("XDG_CONFIG_HOME"); xdg != "" {
		base = xdg
	} else {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		base = filepath.Join(home, ".config")
	}
	return filepath.Join(base, "godotino", "config.json"), nil
}

// Load reads the config from disk.  Returns defaults if the file doesn't exist.
func Load() (*Config, error) {
	path, err := configPath()
	if err != nil {
		return Default(), nil
	}
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return Default(), nil
	}
	if err != nil {
		return nil, fmt.Errorf("reading config: %w", err)
	}
	c := Default()
	if err := json.Unmarshal(data, c); err != nil {
		return nil, fmt.Errorf("parsing config: %w", err)
	}
	return c, nil
}

// Save writes the config to disk.
func (c *Config) Save() error {
	path, err := configPath()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, append(data, '\n'), 0644)
}

// Get returns the value of a config key by its JSON name.
func (c *Config) Get(key string) (interface{}, error) {
	rv := reflect.ValueOf(c).Elem()
	rt := rv.Type()
	for i := 0; i < rt.NumField(); i++ {
		field := rt.Field(i)
		tag := field.Tag.Get("json")
		if tag == key || strings.ToLower(field.Name) == strings.ToLower(key) {
			return rv.Field(i).Interface(), nil
		}
	}
	return nil, fmt.Errorf("unknown config key %q", key)
}

// Set updates a config key by its JSON name.
func (c *Config) Set(key, value string) error {
	rv := reflect.ValueOf(c).Elem()
	rt := rv.Type()
	for i := 0; i < rt.NumField(); i++ {
		field := rt.Field(i)
		tag := field.Tag.Get("json")
		if tag == key || strings.ToLower(field.Name) == strings.ToLower(key) {
			fv := rv.Field(i)
			switch fv.Kind() {
			case reflect.String:
				fv.SetString(value)
			case reflect.Bool:
				b, err := strconv.ParseBool(value)
				if err != nil {
					return fmt.Errorf("invalid bool value %q for key %q", value, key)
				}
				fv.SetBool(b)
			case reflect.Int:
				n, err := strconv.ParseInt(value, 10, 64)
				if err != nil {
					return fmt.Errorf("invalid int value %q for key %q", value, key)
				}
				fv.SetInt(n)
			default:
				return fmt.Errorf("unsupported type for key %q", key)
			}
			return nil
		}
	}
	return fmt.Errorf("unknown config key %q", key)
}

// AllEntries returns all config keys with metadata (for display).
type Entry struct {
	Key     string
	Value   interface{}
	Comment string
}

func (c *Config) AllEntries() []Entry {
	rv := reflect.ValueOf(c).Elem()
	rt := rv.Type()
	entries := make([]Entry, 0, rt.NumField())
	for i := 0; i < rt.NumField(); i++ {
		field := rt.Field(i)
		tag := field.Tag.Get("json")
		comment := field.Tag.Get("comment")
		entries = append(entries, Entry{
			Key:     tag,
			Value:   rv.Field(i).Interface(),
			Comment: comment,
		})
	}
	return entries
}

// Path returns the path of the config file on disk.
func Path() (string, error) {
	return configPath()
}
