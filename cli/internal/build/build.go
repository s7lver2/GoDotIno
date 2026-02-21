// ─────────────────────────────────────────────────────────────────────────────
//  godotino :: build  —  transpile .go → .cpp  +  optional arduino-cli compile
// ─────────────────────────────────────────────────────────────────────────────

package build

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/godotino/cli/internal/core"
	"github.com/godotino/cli/internal/manifest"
	"github.com/godotino/cli/internal/ui"
)

// Options controls the build pipeline.
type Options struct {
	Board     string // override manifest board
	Compile   bool   // invoke arduino-cli after transpilation
	OutputDir string // output directory
	SourceMap bool   // emit #line pragmas
	Verbose   bool
	CoreBin   string // path to godotino-core
	ArduinoCLI string
}

// Result holds the outputs of a successful build.
type Result struct {
	CppFiles   []string
	FirmwareHex string // only set when Compile=true
	Warnings   []string
}

// Run executes the full build pipeline for the project at projectDir.
func Run(projectDir string, m *manifest.Manifest, opts Options) (*Result, error) {
	board := opts.Board
	if board == "" {
		board = m.Board
	}
	outDir := opts.OutputDir
	if outDir == "" {
		outDir = filepath.Join(projectDir, m.Build.OutputDir)
	}

	if err := os.MkdirAll(outDir, 0755); err != nil {
		return nil, fmt.Errorf("creating output dir: %w", err)
	}

	transpiler := core.New(opts.CoreBin, opts.Verbose)
	if !transpiler.Installed() {
		return nil, fmt.Errorf(
			"godotino-core not found — install it or set core_binary in your config\n" +
				"  run: godotino config set core_binary /path/to/godotino-core",
		)
	}

	srcDir := filepath.Join(projectDir, "src")
	goFiles, err := filepath.Glob(filepath.Join(srcDir, "*.go"))
	if err != nil || len(goFiles) == 0 {
		return nil, fmt.Errorf("no .go files found in %s", srcDir)
	}

	ui.SectionTitle(fmt.Sprintf("Transpiling  [board: %s]", board))

	result := &Result{}
	for _, goFile := range goFiles {
		base := strings.TrimSuffix(filepath.Base(goFile), ".go")
		cppFile := filepath.Join(outDir, base+".cpp")

		sp := ui.NewSpinner(fmt.Sprintf("%s → %s", filepath.Base(goFile), filepath.Base(cppFile)))
		sp.Start()

		tr, err := transpiler.Transpile(goFile, cppFile, board, opts.SourceMap || m.Build.SourceMap)
		if err != nil {
			sp.Stop(false, fmt.Sprintf("failed: %s", filepath.Base(goFile)))
			return nil, err
		}

		sp.Stop(true, fmt.Sprintf("%s  →  %s", filepath.Base(goFile), filepath.Base(cppFile)))
		result.CppFiles = append(result.CppFiles, tr.OutputFile)
		result.Warnings = append(result.Warnings, tr.Warnings...)
	}

	for _, w := range result.Warnings {
		ui.Warn(w)
	}

	if !opts.Compile {
		return result, nil
	}

	// ── arduino-cli compile ──────────────────────────────────────────────────
	ui.SectionTitle("Compiling")
	fqbn, err := boardFQBN(board)
	if err != nil {
		return result, fmt.Errorf("unknown board %q — run `godotino boards list` to see supported boards", board)
	}

	arduinoCLI := opts.ArduinoCLI
	if arduinoCLI == "" {
		arduinoCLI = "arduino-cli"
	}

	args := []string{
		"compile",
		"--fqbn", fqbn,
		"--build-path", outDir,
		"--warnings", "all",
	}
	if opts.Verbose {
		args = append(args, "--verbose")
	}
	args = append(args, projectDir)

	sp := ui.NewSpinner(fmt.Sprintf("arduino-cli compile --fqbn %s", fqbn))
	sp.Start()

	cmd := exec.Command(arduinoCLI, args...)
	cmd.Dir = projectDir
	out, err := cmd.CombinedOutput()
	if err != nil {
		sp.Stop(false, "compilation failed")
		// Render compilation errors in rich traceback style
		renderArduinoError(string(out))
		return result, fmt.Errorf("arduino-cli compile failed")
	}
	sp.Stop(true, fmt.Sprintf("firmware written to %s", outDir))

	// find .hex / .bin
	hexFiles, _ := filepath.Glob(filepath.Join(outDir, "*.hex"))
	if len(hexFiles) > 0 {
		result.FirmwareHex = hexFiles[0]
	}

	return result, nil
}

// renderArduinoError converts arduino-cli output into a rich traceback.
func renderArduinoError(output string) {
	lines := strings.Split(output, "\n")
	var frames []ui.Frame
	var errMsg string

	for _, line := range lines {
		// typical: "src/main.cpp:14:5: error: 'Delay' was not declared"
		if strings.Contains(line, ": error:") {
			parts := strings.SplitN(line, ": error:", 2)
			loc := parts[0]
			msg := ""
			if len(parts) > 1 {
				msg = strings.TrimSpace(parts[1])
			}
			locParts := strings.Split(loc, ":")
			frame := ui.Frame{Func: "compile"}
			if len(locParts) >= 1 {
				frame.File = locParts[0]
			}
			if len(locParts) >= 2 {
				fmt.Sscanf(locParts[1], "%d", &frame.Line)
			}
			frame.Code = []ui.CodeLine{{Number: frame.Line, Text: msg, IsPointer: true}}
			frames = append(frames, frame)
			if errMsg == "" {
				errMsg = msg
			}
		}
	}

	if len(frames) == 0 {
		frames = []ui.Frame{{
			File: "sketch",
			Func: "compile",
			Code: []ui.CodeLine{{Number: 0, Text: strings.TrimSpace(output), IsPointer: true}},
		}}
		errMsg = "compilation failed"
	}

	ui.Traceback("CompileError", errMsg, frames)
}

// boardFQBN maps a short board id to an FQBN string.
func boardFQBN(id string) (string, error) {
	table := map[string]string{
		"uno":       "arduino:avr:uno",
		"nano":      "arduino:avr:nano",
		"mega":      "arduino:avr:mega",
		"leonardo":  "arduino:avr:leonardo",
		"micro":     "arduino:avr:micro",
		"due":       "arduino:sam:arduino_due_x",
		"mkr1000":   "arduino:samd:mkr1000",
		"esp32":     "esp32:esp32:esp32",
		"esp8266":   "esp8266:esp8266:generic",
		"pico":      "rp2040:rp2040:rpipico",
		"teensy40":  "teensy:avr:teensy40",
	}
	fqbn, ok := table[strings.ToLower(id)]
	if !ok {
		return "", fmt.Errorf("unknown board")
	}
	return fqbn, nil
}
