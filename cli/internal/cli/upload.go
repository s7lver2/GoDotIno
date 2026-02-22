package cli

import (
	"github.com/spf13/cobra"
	"github.com/tsuki/cli/internal/flash"
	"github.com/tsuki/cli/internal/manifest"
)

func newUploadCmd() *cobra.Command {
	var (
		port     string
		board    string
		buildDir string
	)

	cmd := &cobra.Command{
		Use:   "upload",
		Short: "Upload compiled firmware to a connected board",
		Example: `  tsuki upload
  tsuki upload --port /dev/ttyUSB0
  tsuki upload --port COM3 --board uno`,
		RunE: func(cmd *cobra.Command, args []string) error {
			dir := projectDir()
			_, m, err := manifest.Find(dir)
			if err != nil {
				return err
			}

			return flash.Run(dir, m, flash.Options{
				Port:       port,
				Board:      board,
				BuildDir:   buildDir,
				ArduinoCLI: cfg.ArduinoCLI,
				Verbose:    cfg.Verbose,
			})
		},
	}

	cmd.Flags().StringVarP(&port, "port", "p", "", "serial port (auto-detect if omitted)")
	cmd.Flags().StringVarP(&board, "board", "b", "", "target board (overrides manifest)")
	cmd.Flags().StringVar(&buildDir, "build-dir", "", "directory with compiled firmware")
	return cmd
}
