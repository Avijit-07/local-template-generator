package config

// ApplyDefaults fills in any zero-valued field with the org default, and
// derives PackageName / PackagePath from GroupId + ArtifactId.
//
// TODO(you): implement. See HANDOVER.md "Key decisions already locked in"
// for the default values, and ARCHITECTURE.md's Config builder section for
// the derivation rules. Mutates cfg in place — no return value needed.
func ApplyDefaults(cfg *Config) {
}

// Validate checks cfg for correctness once defaults have been applied and
// returns a descriptive error if something is wrong.
//
// TODO(you): implement. See ARCHITECTURE.md "Security" for the validation
// regex expected on ArtifactId / GroupId segments. Also worth checking here:
// Database is one of "", "postgres", "dynamo", "oracle".
func Validate(cfg *Config) error {
	return nil
}
