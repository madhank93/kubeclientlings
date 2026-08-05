package cmd

import (
	"fmt"

	"github.com/fatih/color"
	"github.com/madhank93/kubeclientlings/kubeclientlings/exercises"
	"github.com/schollz/progressbar/v3"
	"github.com/spf13/cobra"
)

func VerifyCmd(infoFile string) *cobra.Command {
	return &cobra.Command{
		Use:           "verify",
		Short:         "Verify all exercises",
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := EnsureReady(); err != nil {
				color.Red(err.Error())
				return err
			}
			allExercises, err := exercises.List(infoFile)
			if err != nil {
				color.Red(err.Error())
				return err
			}

			bar := progressbar.NewOptions(
				len(allExercises),
				progressbar.OptionSetWidth(50),
				progressbar.OptionEnableColorCodes(true),
				progressbar.OptionSetPredictTime(false),
				progressbar.OptionSetElapsedTime(false),
				progressbar.OptionSetDescription("[cyan][reset] Running exercises"),
				progressbar.OptionSetTheme(progressbar.Theme{
					Saucer:        "[yellow]=[reset]",
					SaucerHead:    "[yellow]>[reset]",
					SaucerPadding: " ",
					BarStart:      "[",
					BarEnd:        "]",
				}),
			)
			if err := bar.RenderBlank(); err != nil {
				color.Red(err.Error())
				return err
			}

			for _, exercise := range allExercises {
				bar.Describe(fmt.Sprintf("Running %s", exercise.Name))
				result, runErr := exercise.Run()
				bar.Add(1) // nolint

				// The exit code decides, not stderr: every exercise talks to a
				// live cluster, and client-go logs warnings to stderr on runs
				// that succeed. Judging by stderr failed all of them.
				if runErr != nil {
					fmt.Print("\n\n")
					color.Cyan("Failed to verify the exercise %s\n\n", exercise.Path)
					color.White("Check the output below: \n\n")
					color.Red(result.Err)
					color.Red(result.Out)
					return fmt.Errorf("exercise %s failed: %w", exercise.Name, runErr)
				}
			}

			color.Green("Congratulations!!!")
			color.Green("You passed all the exercises")
			return nil
		},
	}
}
