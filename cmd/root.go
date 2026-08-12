package cmd

import (
	"embed"
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

// templateFS is set once by Execute and read by generate.go. Cobra commands
// are wired up via package-level vars + init(), so there's no natural place
// to pass it through function arguments once we're inside RunE.
var templateFS embed.FS

var rootCmd = &cobra.Command{
	Use:   "localTemplate",
	Short: "Generate customised Spring Boot projects from bundled templates",
}

// Execute is the CLI entrypoint, called from main.go. templates is the
// embed.FS built at the module root (see /templates.go).
func Execute(templates embed.FS) {
	templateFS = templates

	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
