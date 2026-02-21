// ─────────────────────────────────────────────────────────────────────────────
//  godotino :: core  —  shell-out to goduino-core (Rust transpiler)
// ─────────────────────────────────────────────────────────────────────────────

package core

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/godotino/cli/internal/ui"
)

const defaultBinary = "godotino-core"

// Transpiler wraps the goduino-core binary.
type Transpiler struct {
	binary  string
	verbose bool
}

// New returns a Transpiler using the given binary path (empty = search PATH).
func New(binary string, verbose bool) *Transpiler {
	if binary == "" {
		binary = defaultBinary
	}
	return &Transpiler{binary: binary, verbose: verbose}
}

// TranspileResult holds the output of a transpilation run.
type TranspileResult struct {
	OutputFile string
	Warnings   []string
}

// Transpile transpiles a single .go source file to C++.
func (t *Transpiler) Transpile(inputFile, outputFile, board string, sourceMap bool) (*TranspileResult, error) {
	args := []string{inputFile, outputFile, "--board", board}
	if sourceMap {
		args = append(args, "--source-map")
	}

	cmd := exec.Command(t.binary, args...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if t.verbose {
		ui.Step("core", strings.Join(append([]string{t.binary}, args...), " "))
	}

	if err := cmd.Run(); err != nil {
		// Parse the stderr output and render it as a rich traceback
		errOutput := stderr.String()
		if errOutput != "" {
			renderCoreError(errOutput, inputFile)
		}
		return nil, fmt.Errorf("transpilation failed: %w", err)
	}

	return &TranspileResult{
		OutputFile: outputFile,
		Warnings:   parseWarnings(stderr.String()),
	}, nil
}

// Check validates a .go source file without producing output.
func (t *Transpiler) Check(inputFile, board string) ([]string, []string, error) {
	args := []string{inputFile, "--board", board, "--check"}

	cmd := exec.Command(t.binary, args...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	stdoutStr := stdout.String()
	stderrStr := stderr.String()

	warnings := parseWarnings(stdoutStr + stderrStr)
	errors := parseErrors(stderrStr)

	if err != nil {
		return warnings, errors, fmt.Errorf("check failed")
	}
	return warnings, errors, nil
}

// Version returns the version string of the core binary.
func (t *Transpiler) Version() (string, error) {
	out, err := exec.Command(t.binary, "--version").Output()
	if err != nil {
		return "", fmt.Errorf("cannot run %s: %w", t.binary, err)
	}
	return strings.TrimSpace(string(out)), nil
}

// Installed reports whether the core binary is available on PATH.
func (t *Transpiler) Installed() bool {
	_, err := exec.LookPath(t.binary)
	return err == nil
}

// ── Error parsing / rendering ─────────────────────────────────────────────────

// renderCoreError parses godotino-core stderr and renders a rich traceback.
func renderCoreError(raw, inputFile string) {
	// Try to parse structured error output.
	// godotino-core outputs lines like:
	//   error[E001]: undefined function `Delay`
	//     --> src/main.go:14:5
	//      |
	//   14 |     Delay(1000)
	//      |     ^^^^^ not found

	lines := strings.Split(raw, "\n")

	var errType, errMsg string
	var frames []ui.Frame

	var currentFrame *ui.Frame
	var codeLines []ui.CodeLine
	var errorLineNum int

	for _, line := range lines {
		line = strings.TrimRight(line, "\r")

		// "error[E001]: message"
		if strings.HasPrefix(line, "error") {
			parts := strings.SplitN(line, ": ", 2)
			errType = parts[0]
			if len(parts) > 1 {
				errMsg = parts[1]
			}
			continue
		}

		// "  --> file.go:14:5"
		if strings.Contains(line, "-->") {
			if currentFrame != nil {
				currentFrame.Code = codeLines
				frames = append(frames, *currentFrame)
			}
			loc := strings.TrimSpace(strings.TrimPrefix(line, "-->"))
			loc = strings.TrimSpace(strings.TrimPrefix(loc, "·"))
			parts := strings.Split(loc, ":")
			frame := ui.Frame{
				File: inputFile,
				Func: "main",
			}
			if len(parts) >= 1 {
				frame.File = parts[0]
			}
			if len(parts) >= 2 {
				fmt.Sscanf(parts[1], "%d", &errorLineNum)
				frame.Line = errorLineNum
			}
			codeLines = []ui.CodeLine{}
			currentFrame = &frame
			continue
		}

		// " 14 |   Delay(1000)"
		if currentFrame != nil {
			trimmed := strings.TrimSpace(line)
			if len(trimmed) > 0 && trimmed[0] != '|' && trimmed[0] != '^' {
				var lineNum int
				rest := line
				if _, err := fmt.Sscanf(trimmed, "%d |", &lineNum); err == nil {
					pipeIdx := strings.Index(line, "|")
					if pipeIdx >= 0 && pipeIdx+1 < len(line) {
						rest = line[pipeIdx+1:]
					}
					codeLines = append(codeLines, ui.CodeLine{
						Number:    lineNum,
						Text:      rest,
						IsPointer: lineNum == errorLineNum,
					})
				}
			}
		}
	}

	if currentFrame != nil {
		currentFrame.Code = codeLines
		frames = append(frames, *currentFrame)
	}

	if errType == "" {
		errType = "TranspileError"
		errMsg = strings.TrimSpace(raw)
	}

	if len(frames) == 0 {
		// fallback: show raw error
		frames = []ui.Frame{{
			File: inputFile,
			Line: 0,
			Func: "transpile",
			Code: []ui.CodeLine{{Number: 0, Text: errMsg, IsPointer: true}},
		}}
	}

	ui.Traceback(errType, errMsg, frames)
	_ = os.Stderr
}

func parseWarnings(output string) []string {
	var w []string
	for _, line := range strings.Split(output, "\n") {
		if strings.Contains(strings.ToLower(line), "warning") {
			w = append(w, strings.TrimSpace(line))
		}
	}
	return w
}

func parseErrors(output string) []string {
	var e []string
	for _, line := range strings.Split(output, "\n") {
		if strings.Contains(strings.ToLower(line), "error") {
			e = append(e, strings.TrimSpace(line))
		}
	}
	return e
}
