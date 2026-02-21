package cli

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
	"github.com/godotino/cli/internal/manifest"
	"github.com/godotino/cli/internal/ui"
)

var initTemplate = `package main

import "arduino"

func setup() {
	arduino.PinMode(arduino.LED_BUILTIN, arduino.OUTPUT)
}

func loop() {
	arduino.DigitalWrite(arduino.LED_BUILTIN, arduino.HIGH)
	arduino.Delay(1000)
	arduino.DigitalWrite(arduino.LED_BUILTIN, arduino.LOW)
	arduino.Delay(1000)
}
`

func newInitCmd() *cobra.Command {
	var board string

	cmd := &cobra.Command{
		Use:   "init [project-name]",
		Short: "Initialize a new godotino project",
		Args:  cobra.MaximumNArgs(1),
		Example: `  godotino init
  godotino init my-robot
  godotino init my-robot --board esp32`,
		RunE: func(cmd *cobra.Command, args []string) error {
			dir := projectDir()
			name := filepath.Base(dir)
			if len(args) > 0 {
				name = args[0]
				dir = filepath.Join(dir, name)
			}

			if board == "" {
				board = cfg.DefaultBoard
			}

			ui.SectionTitle("Initializing project")
			ui.Info(fmt.Sprintf("Name:  %s", name))
			ui.Info(fmt.Sprintf("Board: %s", board))
			ui.Info(fmt.Sprintf("Dir:   %s", dir))
			fmt.Println()

			// Create directories
			srcDir := filepath.Join(dir, "src")
			for _, d := range []string{dir, srcDir} {
				if err := os.MkdirAll(d, 0755); err != nil {
					return fmt.Errorf("creating directory %s: %w", d, err)
				}
			}

			// Write manifest
			m := manifest.Default(name, board)
			if err := m.Save(dir); err != nil {
				return fmt.Errorf("writing manifest: %w", err)
			}
			ui.Success("Created goduino.json")

			// Write main.go skeleton
			mainGo := filepath.Join(srcDir, "main.go")
			if _, err := os.Stat(mainGo); os.IsNotExist(err) {
				if err := os.WriteFile(mainGo, []byte(initTemplate), 0644); err != nil {
					return fmt.Errorf("writing main.go: %w", err)
				}
				ui.Success("Created src/main.go")
			} else {
				ui.Warn("src/main.go already exists — skipping")
			}

			// .gitignore
			gitignore := filepath.Join(dir, ".gitignore")
			if _, err := os.Stat(gitignore); os.IsNotExist(err) {
				content := "build/\n*.hex\n*.bin\n*.uf2\n"
				_ = os.WriteFile(gitignore, []byte(content), 0644)
				ui.Success("Created .gitignore")
			}

			fmt.Println()
			ui.Success(fmt.Sprintf("Project '%s' initialized for board '%s'!", name, board))
			fmt.Println()
			ui.Info("Next steps:")
			ui.Step("  1", fmt.Sprintf("cd %s", name))
			ui.Step("  2", "edit src/main.go")
			ui.Step("  3", "godotino build")
			ui.Step("  4", "godotino upload")
			return nil
		},
	}

	cmd.Flags().StringVarP(&board, "board", "b", "", "target board (default from config)")
	return cmd
}
