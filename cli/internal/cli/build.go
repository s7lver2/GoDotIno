// ─────────────────────────────────────────────────────────────────────────────
//  godotino :: build  (fixed)
//
//  THE BUG: arduino-cli compile requires a *sketch directory* — a folder
//  whose name matches the .ino file inside it.  The old code passed the
//  project root directly, which never contains a .ino file.
//
//  THE FIX: after transpiling, we:
//    1. Write .cpp files into  build/<project-name>/
//    2. Generate              build/<project-name>/<project-name>.ino
//    3. Pass the sketch dir   build/<project-name>/   to arduino-cli
//    4. Cache .hex/.elf into  build/.cache/
// ─────────────────────────────────────────────────────────────────────────────

package cli

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
	"github.com/godotino/cli/internal/core"
	"github.com/godotino/cli/internal/manifest"
	"github.com/godotino/cli/internal/pkgmgr"
	"github.com/godotino/cli/internal/ui"
)

// Options controls the build pipeline.
type Options struct {
	Board      string
	Compile    bool
	OutputDir  string
	SourceMap  bool
	Verbose    bool
	CoreBin    string
	ArduinoCLI string
}

// Result holds the outputs of a successful build.
type Result struct {
	CppFiles    []string
	SketchDir   string // path to the generated Arduino sketch dir
	FirmwareHex string
	Warnings    []string
}

// Run executes the full build pipeline.
func Run(projectDir string, m *manifest.Manifest, opts Options) (*Result, error) {
	board := opts.Board
	if board == "" {
		board = m.Board
	}

	// Base build directory: <project>/build/
	baseOutDir := opts.OutputDir
	if baseOutDir == "" {
		baseOutDir = filepath.Join(projectDir, m.Build.OutputDir)
	}

	// ── Arduino sketch directory ─────────────────────────────────────────────
	// arduino-cli compile requires a sketch directory whose name matches the
	// .ino file inside it:  build/<name>/<name>.ino
	sketchName := sanitizeSketchName(m.Name)
	if sketchName == "" {
		sketchName = "sketch"
	}
	sketchDir := filepath.Join(baseOutDir, sketchName)

	if err := os.MkdirAll(sketchDir, 0755); err != nil {
		return nil, fmt.Errorf("creating sketch dir: %w", err)
	}

	transpiler := core.New(opts.CoreBin, opts.Verbose)
	if !transpiler.Installed() {
		return nil, fmt.Errorf(
			"godotino-core not found — install it or set core_binary in config\n"+
				"  godotino config set core_binary /path/to/godotino-core",
		)
	}

	srcDir := filepath.Join(projectDir, "src")
	goFiles, err := filepath.Glob(filepath.Join(srcDir, "*.go"))
	if err != nil || len(goFiles) == 0 {
		return nil, fmt.Errorf("no .go files found in %s", srcDir)
	}

	// Resolve declared packages
	pkgNames := m.PackageNames()
	libsDir  := pkgmgr.LibsDir()

	if len(pkgNames) > 0 {
		ui.SectionTitle(fmt.Sprintf("Transpiling  [board: %s]  [packages: %s]",
			board, strings.Join(pkgNames, ", ")))
		for _, name := range pkgNames {
			if ok, _ := pkgmgr.IsInstalled(name); !ok {
				return nil, fmt.Errorf(
					"package %q declared in goduino.json is not installed\n"+
						"  Run: godotino pkg install %s", name, name,
				)
			}
		}
	} else {
		ui.SectionTitle(fmt.Sprintf("Transpiling  [board: %s]", board))
	}

	result := &Result{SketchDir: sketchDir}

	for _, goFile := range goFiles {
		base    := strings.TrimSuffix(filepath.Base(goFile), ".go")
		cppFile := filepath.Join(sketchDir, base+".cpp") // write INTO sketch dir

		sp := ui.NewSpinner(fmt.Sprintf("%s → %s", filepath.Base(goFile), filepath.Base(cppFile)))
		sp.Start()

		tr, err := transpiler.Transpile(core.TranspileRequest{
			InputFile:  goFile,
			OutputFile: cppFile,
			Board:      board,
			SourceMap:  opts.SourceMap || m.Build.SourceMap,
			LibsDir:    libsDir,
			PkgNames:   pkgNames,
		})
		if err != nil {
			sp.Stop(false, fmt.Sprintf("failed: %s", filepath.Base(goFile)))
			return nil, err
		}

		sp.Stop(true, fmt.Sprintf("%s  →  %s", filepath.Base(goFile), filepath.Base(cppFile)))
		result.CppFiles = append(result.CppFiles, tr.OutputFile)
		result.Warnings  = append(result.Warnings, tr.Warnings...)
	}

	for _, w := range result.Warnings {
		ui.Warn(w)
	}

	// ── Write the .ino stub ──────────────────────────────────────────────────
	// arduino-cli needs <sketchDir>/<sketchName>.ino to exist.
	if err := writeInoStub(sketchDir, sketchName, result.CppFiles); err != nil {
		return nil, fmt.Errorf("writing .ino stub: %w", err)
	}
	ui.Step("sketch", fmt.Sprintf("wrote %s/%s.ino", sketchName, sketchName))

	if !opts.Compile {
		return result, nil
	}

	// ── arduino-cli compile ──────────────────────────────────────────────────
	ui.SectionTitle("Compiling")
	fqbn, err := boardFQBN(board)
	if err != nil {
		return result, fmt.Errorf("unknown board %q — run `godotino boards list`", board)
	}

	arduinoCLI := opts.ArduinoCLI
	if arduinoCLI == "" {
		arduinoCLI = "arduino-cli"
	}

	// Compile artifacts go into build/.cache/ to keep the sketch dir clean.
	buildCacheDir := filepath.Join(baseOutDir, ".cache")
	_ = os.MkdirAll(buildCacheDir, 0755)

	args := []string{
		"compile",
		"--fqbn", fqbn,
		"--build-path", buildCacheDir,
		"--warnings", "all",
	}
	if opts.Verbose {
		args = append(args, "--verbose")
	}
	// *** KEY FIX: pass the SKETCH DIR, not the project root ***
	args = append(args, sketchDir)

	sp := ui.NewSpinner(fmt.Sprintf("arduino-cli compile --fqbn %s", fqbn))
	sp.Start()

	cmd := exec.Command(arduinoCLI, args...)
	cmd.Dir = sketchDir
	out, cmdErr := cmd.CombinedOutput()
	if cmdErr != nil {
		sp.Stop(false, "compilation failed")
		renderArduinoError(string(out))
		return result, fmt.Errorf("arduino-cli compile failed")
	}
	sp.Stop(true, fmt.Sprintf("firmware written to %s", buildCacheDir))

	hexFiles, _ := filepath.Glob(filepath.Join(buildCacheDir, "*.hex"))
	if len(hexFiles) > 0 {
		result.FirmwareHex = hexFiles[0]
	}

	return result, nil
}

// writeInoStub creates <sketchDir>/<sketchName>.ino — the required entry
// point for arduino-cli.  The file name MUST match the directory name.
func writeInoStub(sketchDir, sketchName string, _ []string) error {
	const stub = "// Auto-generated by godotino — do not edit.\n" +
		"// arduino-cli compiles the .cpp files in this directory automatically.\n"
	return os.WriteFile(filepath.Join(sketchDir, sketchName+".ino"), []byte(stub), 0644)
}

// sanitizeSketchName converts a project name to a valid Arduino sketch name:
// only letters, digits, underscores; cannot start with a digit.
func sanitizeSketchName(name string) string {
	var sb strings.Builder
	for i, r := range name {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r == '_':
			sb.WriteRune(r)
		case r >= '0' && r <= '9':
			if i > 0 {
				sb.WriteRune(r)
			}
		default:
			if sb.Len() > 0 {
				sb.WriteRune('_')
			}
		}
	}
	return sb.String()
}

func newBuildCmd() *cobra.Command {
	var board string
	var output string
	var compile bool
	var verbose bool

	cmd := &cobra.Command{
		Use:   "build",
		Short: "Transpile and optionally compile the project",
		Example: `  godotino build
  godotino build --board esp32
  godotino build --compile`,
		RunE: func(cmd *cobra.Command, args []string) error {
			dir := projectDir()
			m, err := manifest.Load(dir)
			if err != nil {
				return err
			}

			opts := Options{
				Board:      board,
				Compile:    compile,
				OutputDir:  output,
				Verbose:    verbose,
				CoreBin:    cfg.CoreBinary,
				ArduinoCLI: cfg.ArduinoCLI,
				SourceMap:  m.Build.SourceMap,
			}

			res, err := Run(dir, m, opts)
			if err != nil {
				return err
			}
			if res.SketchDir != "" {
				ui.Info(fmt.Sprintf("Sketch: %s", res.SketchDir))
			}
			ui.Success("Build finished!")
			return nil
		},
	}

	cmd.Flags().StringVarP(&board, "board", "b", "", "target board (default from manifest)")
	cmd.Flags().StringVarP(&output, "out", "o", "", "output directory")
	cmd.Flags().BoolVarP(&compile, "compile", "c", false, "compile to firmware after transpile")
	cmd.Flags().BoolVarP(&verbose, "verbose", "v", false, "verbose output")
	return cmd
}

func renderArduinoError(output string) {
	lines := strings.Split(output, "\n")
	var frames []ui.Frame
	var errMsg string

	for _, line := range lines {
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
			File: "sketch", Func: "compile",
			Code: []ui.CodeLine{{Number: 0, Text: strings.TrimSpace(output), IsPointer: true}},
		}}
		errMsg = "compilation failed"
	}
	ui.Traceback("CompileError", errMsg, frames)
}

func boardFQBN(id string) (string, error) {
	table := map[string]string{
		"uno":      "arduino:avr:uno",
		"nano":     "arduino:avr:nano",
		"mega":     "arduino:avr:mega",
		"leonardo": "arduino:avr:leonardo",
		"micro":    "arduino:avr:micro",
		"due":      "arduino:sam:arduino_due_x",
		"mkr1000":  "arduino:samd:mkr1000",
		"esp32":    "esp32:esp32:esp32",
		"esp8266":  "esp8266:esp8266:generic",
		"pico":     "rp2040:rp2040:rpipico",
		"teensy40": "teensy:avr:teensy40",
	}
	fqbn, ok := table[strings.ToLower(id)]
	if !ok {
		return "", fmt.Errorf("unknown board")
	}
	return fqbn, nil
}