package cli

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/godotino/cli/internal/build"
	"github.com/godotino/cli/internal/manifest"
	"github.com/godotino/cli/internal/ui"
)

func newBuildCmd() *cobra.Command {
	var (
		board     string
		compile   bool
		outputDir string
		sourceMap bool
	)

	cmd := &cobra.Command{
		Use:   "build",
		Short: "Transpile Go sources to C++ (and optionally compile)",
		Example: `  godotino build
  godotino build --board esp32
  godotino build --compile
  godotino build --compile --output build/`,
		RunE: func(cmd *cobra.Command, args []string) error {
			dir := projectDir()
			_, m, err := manifest.Find(dir)
			if err != nil {
				return err
			}

			opts := build.Options{
				Board:      board,
				Compile:    compile,
				OutputDir:  outputDir,
				SourceMap:  sourceMap,
				Verbose:    cfg.Verbose,
				CoreBin:    cfg.CoreBinary,
				ArduinoCLI: cfg.ArduinoCLI,
			}

			result, err := build.Run(dir, m, opts)
			if err != nil {
				return err
			}

			fmt.Println()
			ui.Success(fmt.Sprintf("Build complete — %d file(s) transpiled", len(result.CppFiles)))
			if result.FirmwareHex != "" {
				ui.Info(fmt.Sprintf("Firmware: %s", result.FirmwareHex))
			}
			return nil
		},
	}

	cmd.Flags().StringVarP(&board, "board", "b", "", "target board (overrides manifest)")
	cmd.Flags().BoolVar(&compile, "compile", false, "also compile with arduino-cli")
	cmd.Flags().StringVarP(&outputDir, "output", "o", "", "output directory (default: build/)")
	cmd.Flags().BoolVar(&sourceMap, "source-map", false, "emit #line pragmas")
	return cmd
}
