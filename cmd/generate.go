package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/myorg/localTemplate/internal/config"
	"github.com/myorg/localTemplate/internal/prompt"
	"github.com/myorg/localTemplate/internal/template"
)

// genCfg is populated directly by flag bindings below, then filled in by
// prompt.CollectMissing and config.ApplyDefaults before use.
var genCfg config.Config

// The three DB flags and --s3 aren't Config fields themselves (Database is a
// single string, AWSResources is a slice) — they get folded into genCfg at
// the top of runGenerate.
var (
	genPostgres bool
	genDynamo   bool
	genOracle   bool
	genS3       bool
)

var generateCmd = &cobra.Command{
	Use:   "generate",
	Short: "Generate a new Spring Boot project",
	RunE:  runGenerate,
}

func init() {
	rootCmd.AddCommand(generateCmd)

	flags := generateCmd.Flags()

	flags.StringVar(&genCfg.ArtifactId, "artifact", "", "Artifact / project name (required; prompted if missing)")
	flags.StringVar(&genCfg.GroupId, "group", "", "Maven group ID (default: com.myorg)")
	flags.StringVar(&genCfg.Version, "version", "", "Project version (default: 1.0.0)")
	flags.IntVar(&genCfg.JavaVersion, "java", 0, "Java version (default: 25)")

	flags.BoolVar(&genPostgres, "postgres", false, "Add PostgreSQL layer")
	flags.BoolVar(&genDynamo, "dynamo", false, "Add DynamoDB layer")
	flags.BoolVar(&genOracle, "oracle", false, "Add Oracle layer")
	generateCmd.MarkFlagsMutuallyExclusive("postgres", "dynamo", "oracle")

	flags.BoolVar(&genS3, "s3", false, "Add S3 integration layer")
	flags.BoolVar(&genCfg.ECS, "ecs", false, "Add ECS compute layer (Dockerfile + task def scaffold)")

	flags.BoolVar(&genCfg.AIEnabled, "ai", false, "Replace prompt engine with Claude wizard")
	flags.StringVar(&genCfg.TemplateOverride, "template", "", "Remote template override (github.com/owner/repo)")

	flags.StringVar(&genCfg.OutputDir, "output", "", "Output directory (default: ./{artifact})")
	flags.BoolVar(&genCfg.DryRun, "dry-run", false, "Print resolved config, write nothing")
}

func runGenerate(cmd *cobra.Command, args []string) error {
	switch {
	case genPostgres:
		genCfg.Database = "postgres"
	case genDynamo:
		genCfg.Database = "dynamo"
	case genOracle:
		genCfg.Database = "oracle"
	}
	if genS3 {
		genCfg.AWSResources = append(genCfg.AWSResources, "s3")
	}

	if genCfg.AIEnabled {
		return fmt.Errorf("--ai is out of scope for MVP1")
	}

	if err := prompt.CollectMissing(&genCfg); err != nil {
		return err
	}

	config.ApplyDefaults(&genCfg)

	if err := config.Validate(&genCfg); err != nil {
		return err
	}

	layers := template.ResolveLayers(&genCfg)

	if genCfg.DryRun {
		fmt.Printf("%+v\n", genCfg)
		return nil
	}

	outDir := genCfg.OutputDir
	if outDir == "" {
		outDir = "./" + genCfg.ArtifactId
	}

	return template.Process(templateFS, layers, &genCfg, outDir)
}
