package template

import "github.com/myorg/localTemplate/internal/config"

// ResolveLayers returns, in merge order, the embed.FS paths that make up
// this project — e.g. []string{"templates/base", "templates/db/postgres",
// "templates/aws/s3"}. Layers are merged in the order returned, so later
// layers (db, aws, compute) can .patch files the base layer already wrote.
//
// TODO(you): implement. See HANDOVER.md "Layer resolution order" for the
// exact logic — it's a short one.
func ResolveLayers(cfg *config.Config) []string {
	return nil
}
