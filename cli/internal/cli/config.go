package cli

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/godotino/cli/internal/config"
	"github.com/godotino/cli/internal/ui"
)

func newConfigCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "config",
		Short: "Get or set CLI configuration",
		Long: `Manage godotino CLI configuration.

The config file is stored at ~/.config/godotino/config.json.

Use 'godotino config set <key> <value>' to set a value.
Use 'godotino config get <key>' to read a specific key.
Use 'godotino config show' to display all settings.`,
	}

	cmd.AddCommand(
		newConfigSetCmd(),
		newConfigGetCmd(),
		newConfigShowCmd(),
		newConfigPathCmd(),
	)
	return cmd
}

// ── config set ────────────────────────────────────────────────────────────────

func newConfigSetCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "set <key> <value>",
		Short: "Set a configuration key",
		Example: `  godotino config set default_board esp32
  godotino config set arduino_cli /usr/local/bin/arduino-cli
  godotino config set verbose true
  godotino config set default_baud 115200`,
		Args: cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			key, value := args[0], args[1]

			c, err := config.Load()
			if err != nil {
				return err
			}

			if err := c.Set(key, value); err != nil {
				ui.Fail(err.Error())
				fmt.Fprintln(os.Stderr, "")
				ui.Info("Available keys:")
				for _, e := range config.Default().AllEntries() {
					ui.Step("  "+e.Key, e.Comment)
				}
				return fmt.Errorf("unknown key")
			}

			if err := c.Save(); err != nil {
				return fmt.Errorf("saving config: %w", err)
			}

			ui.Success(fmt.Sprintf("Set %s = %s", key, value))

			// Show updated entry in styled box
			ui.PrintConfig("godotino config", []ui.ConfigEntry{
				{Key: key, Value: value},
			}, false)

			return nil
		},
	}
	return cmd
}

// ── config get ────────────────────────────────────────────────────────────────

func newConfigGetCmd() *cobra.Command {
	var (
		rawFlag   bool
		paramFlag string
	)

	cmd := &cobra.Command{
		Use:   "get <key>",
		Short: "Get a configuration key",
		Example: `  godotino config get default_board
  godotino config get default_board --raw
  godotino config get default_board --param`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			key := args[0]
			_ = paramFlag

			c, err := config.Load()
			if err != nil {
				return err
			}

			val, err := c.Get(key)
			if err != nil {
				return err
			}

			if rawFlag {
				fmt.Println(val)
				return nil
			}

			// Find comment
			comment := ""
			for _, e := range c.AllEntries() {
				if e.Key == key {
					comment = e.Comment
					break
				}
			}

			ui.PrintConfig("godotino config", []ui.ConfigEntry{
				{Key: key, Value: val, Comment: comment},
			}, false)

			return nil
		},
	}

	cmd.Flags().BoolVar(&rawFlag, "raw", false, "print raw value only (no styling)")
	cmd.Flags().StringVar(&paramFlag, "param", "", "filter by param (same as key)")
	return cmd
}

// ── config show ───────────────────────────────────────────────────────────────

func newConfigShowCmd() *cobra.Command {
	var rawFlag bool

	cmd := &cobra.Command{
		Use:     "show",
		Short:   "Show all configuration values",
		Aliases: []string{"list", "ls"},
		Example: `  godotino config show
  godotino config show --raw`,
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := config.Load()
			if err != nil {
				return err
			}

			entries := c.AllEntries()
			uiEntries := make([]ui.ConfigEntry, len(entries))
			for i, e := range entries {
				uiEntries[i] = ui.ConfigEntry{
					Key:     e.Key,
					Value:   e.Value,
					Comment: e.Comment,
				}
			}

			ui.PrintConfig("godotino config", uiEntries, rawFlag)
			return nil
		},
	}

	cmd.Flags().BoolVar(&rawFlag, "raw", false, "print raw key=value pairs")
	return cmd
}

// ── config path ───────────────────────────────────────────────────────────────

func newConfigPathCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "path",
		Short: "Print the path to the config file",
		RunE: func(cmd *cobra.Command, args []string) error {
			p, err := config.Path()
			if err != nil {
				return err
			}
			ui.Info(fmt.Sprintf("Config file: %s", p))
			if _, err := os.Stat(p); os.IsNotExist(err) {
				ui.Warn("File does not exist yet — it will be created on first `godotino config set`")
			}
			return nil
		},
	}
}
