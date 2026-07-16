package cmd

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/madhank93/kubeclientlings/kubeclientlings/tui"
	"github.com/spf13/cobra"
)

func WatchCmd(infoFile string) *cobra.Command {
	return &cobra.Command{
		Use:           "watch",
		Short:         "Interactive TUI: verify exercises as you edit",
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := EnsureReady(); err != nil {
				return err
			}
			m, err := tui.New(infoFile)
			if err != nil {
				return err
			}
			p := tea.NewProgram(m, tea.WithAltScreen())
			_, err = p.Run()
			return err
		},
	}
}
