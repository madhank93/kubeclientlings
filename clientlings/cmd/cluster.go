package cmd

import (
	"fmt"
	"os"

	"github.com/fatih/color"
	"github.com/madhank93/clientlings/clientlings/cluster"
	"github.com/madhank93/clientlings/clientlings/preflight"
	"github.com/spf13/cobra"
)

func UpCmd() *cobra.Command {
	return &cobra.Command{
		Use:           "up",
		Short:         "Create the local kind cluster exercises run against",
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			return cluster.Up()
		},
	}
}

func DownCmd() *cobra.Command {
	return &cobra.Command{
		Use:           "down",
		Short:         "Delete the local kind cluster",
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			return cluster.Down()
		},
	}
}

func DoctorCmd() *cobra.Command {
	return &cobra.Command{
		Use:           "doctor",
		Short:         "Check tooling and cluster health",
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			issues := preflight.Check()
			if len(issues) == 0 {
				color.Green("✅ all good — docker, kind, kubectl, go and the %q cluster are ready", cluster.Name)
				return nil
			}
			for _, issue := range issues {
				color.Red("✗ %s", issue.Msg)
				color.Yellow("  fix: %s", issue.Fix)
			}
			return fmt.Errorf("%d issue(s) found", len(issues))
		},
	}
}

// EnsureReady runs preflight and prints actionable issues. Commands that hit
// the cluster call this first so the learner sees "cluster not up" instead of
// a cryptic connection-refused from deep inside client-go.
func EnsureReady() error {
	if os.Getenv("CLIENTLINGS_SKIP_PREFLIGHT") != "" {
		return nil
	}
	issues := preflight.Check()
	if len(issues) == 0 {
		return nil
	}
	for _, issue := range issues {
		color.Red("✗ %s", issue.Msg)
		color.Yellow("  fix: %s", issue.Fix)
	}
	return fmt.Errorf("environment not ready — fix the issue(s) above and retry")
}
