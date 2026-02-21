// ─────────────────────────────────────────────────────────────────────────────
//  godotino :: manifest  —  load / save goduino.json
// ─────────────────────────────────────────────────────────────────────────────

package manifest

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

const FileName = "goduino.json"

type Manifest struct {
	Name         string       `json:"name"`
	Version      string       `json:"version"`
	Board        string       `json:"board"`
	GoVersion    string       `json:"go_version"`
	Description  string       `json:"description,omitempty"`
	Dependencies []Dependency `json:"dependencies"`
	Build        BuildConfig  `json:"build"`
}

type Dependency struct {
	Name    string `json:"name"`
	Version string `json:"version"`
	Kind    string `json:"kind"` // "go" | "arduino" | "local"
}

type BuildConfig struct {
	OutputDir  string   `json:"output_dir"`
	CppStd     string   `json:"cpp_std"`
	Optimize   string   `json:"optimize"`
	ExtraFlags []string `json:"extra_flags"`
	SourceMap  bool     `json:"source_map"`
}

// Default returns a new manifest with sensible defaults.
func Default(name, board string) *Manifest {
	return &Manifest{
		Name:         name,
		Version:      "0.1.0",
		Board:        board,
		GoVersion:    "1.21",
		Dependencies: []Dependency{},
		Build: BuildConfig{
			OutputDir:  "build",
			CppStd:     "c++11",
			Optimize:   "Os",
			ExtraFlags: []string{},
			SourceMap:  false,
		},
	}
}

// Load reads goduino.json from the given directory.
func Load(dir string) (*Manifest, error) {
	path := filepath.Join(dir, FileName)
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("no %s found in %s — run `godotino init` first", FileName, dir)
		}
		return nil, fmt.Errorf("reading %s: %w", FileName, err)
	}
	var m Manifest
	if err := json.Unmarshal(data, &m); err != nil {
		return nil, fmt.Errorf("parsing %s: %w", FileName, err)
	}
	return &m, nil
}

// Save writes the manifest to goduino.json in the given directory.
func (m *Manifest) Save(dir string) error {
	data, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(dir, FileName), append(data, '\n'), 0644)
}

// Find searches upward from dir for a goduino.json file.
func Find(startDir string) (string, *Manifest, error) {
	dir := startDir
	for {
		path := filepath.Join(dir, FileName)
		if _, err := os.Stat(path); err == nil {
			m, err := Load(dir)
			return dir, m, err
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	return "", nil, fmt.Errorf("no %s found (searched from %s upward)", FileName, startDir)
}
