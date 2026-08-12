// Package prompt fills in whatever Config fields the user didn't already
// supply as flags, via interactive charmbracelet/huh forms.
package prompt

import "github.com/myorg/localTemplate/internal/config"

// CollectMissing inspects cfg and prompts only for fields still at their
// zero value — anything already set by a flag is left untouched and skipped.
//
// TODO(you): implement using charmbracelet/huh. See ARCHITECTURE.md's
// "Prompt engine" section for a worked example of the pattern (build a
// []huh.Field, append conditionally, huh.NewForm(...).Run() at the end —
// and do nothing if the field slice ends up empty).
func CollectMissing(cfg *config.Config) error {
	return nil
}
