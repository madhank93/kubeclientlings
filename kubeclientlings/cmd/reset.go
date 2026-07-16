package cmd

import (
	"github.com/fatih/color"
	"github.com/madhank93/kubeclientlings/kubeclientlings/exercises"
	"github.com/spf13/cobra"
)

func ResetCmd(infoFile string) *cobra.Command {
	return &cobra.Command{
		Use:           "reset <exercise name>",
		Short:         "Restore an exercise to its original (broken) state via git",
		Args:          cobra.ExactArgs(1),
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			exercise, err := exercises.Find(args[0], infoFile)
			if err != nil {
				color.Red("No exercise found for '%s'", args[0])
				return err
			}
			if err := exercises.Reset(exercise); err != nil {
				color.Red(err.Error())
				return err
			}
			color.Green("Reset %s to its original state", exercise.Name)
			return nil
		},
	}
}
