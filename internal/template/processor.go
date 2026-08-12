package template

import (
	"embed"

	"github.com/myorg/localTemplate/internal/config"
)

// Process walks each resolved layer (in order) and writes the merged,
// templated project tree to outDir.
//
// TODO(you): implement. See HANDOVER.md "Critical implementation details"
// for the pieces this needs to wire together:
//   - fs.WalkDir over each layer in templates
//   - expand every file's path through expandPath before writing it
//   - render text files (.java .kt .xml .yml .yaml .properties .gradle .kts
//     .md .sql .json .sh) through text/template with cfg as the data
//   - copy everything else byte-for-byte (binary passthrough)
//   - .patch files: find the base file by stripping ".patch", append the
//     patch content after a blank line; if there's no base file, write it
//     standalone with ".patch" dropped from the name
func Process(templates embed.FS, layers []string, cfg *config.Config, outDir string) error {
	return nil
}

// expandPath expands a template-tree path — e.g. "{{.PackagePath}}/Application.java"
// — through text/template using cfg as the data, so it becomes something like
// "com/myorg/myservice/Application.java".
//
// TODO(you): implement.
func expandPath(tmplPath string, cfg *config.Config) (string, error) {
	return "", nil
}
