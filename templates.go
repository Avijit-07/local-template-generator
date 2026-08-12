package main

import "embed"

// TemplateFS holds the bundled project templates (see templates/). It has to be
// declared here at the module root: go:embed patterns can't use ".." to reach
// out of the declaring file's own directory, so this can't live in internal/template
// even though that's the package that actually uses it. We thread it down through
// cmd.Execute instead.
//
//go:embed templates
var TemplateFS embed.FS
