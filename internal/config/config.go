// Package config holds the Config struct that every other package reads
// from: flags and prompts populate it, the template resolver and processor
// consume it. See HANDOVER.md for the full field-by-field rationale.
package config

// Config is the fully-resolved set of choices for a single `generate` run.
type Config struct {
	// Project identity
	ArtifactId  string
	GroupId     string // default: "com.myorg"
	Version     string // default: "1.0.0"
	PackageName string // derived: sanitise(GroupId + "." + ArtifactId)
	PackagePath string // derived: strings.ReplaceAll(PackageName, ".", "/")

	// Build
	JavaVersion int // default: 25

	// Integrations — DB (mutually exclusive)
	Database string // "postgres" | "dynamo" | "oracle" | ""

	// Integrations — AWS (additive)
	AWSResources []string // e.g. ["s3"] — extensible for sqs, ses later

	// Compute
	ECS bool

	// Output
	OutputDir string // default: "./{ArtifactId}"

	// Modes
	AIEnabled        bool
	DryRun           bool
	TemplateOverride string // remote github URL; empty = use embed.FS
}
