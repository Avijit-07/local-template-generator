# Handover — `localTemplate`

> Drop this file into a Code session and reference it alongside `ARCHITECTURE.md`.  
> It captures every decision made so far, what MVP1 scope is, and exactly where to start coding.

---

## What this project is

A Go CLI tool that generates customised Spring Boot projects. Single static binary,
no JVM required, works offline (except when `--ai` or `--template` remote override is used).

Replaces an earlier Spring Boot web application that did the same thing via REST API + GitHub
template fetch. That approach is scrapped entirely.

---

## Key decisions already locked in

| Decision | Choice | Rationale |
|---|---|---|
| Language | Go 1.22+ | Single binary distribution, fast startup, embed.FS |
| CLI framework | `cobra` | Standard Go CLI, shell completion built-in |
| Interactive prompts | `charmbracelet/huh` | Styled selects / multi-selects, same Config struct binding |
| Template storage | `embed.FS` (bundled in binary) | No external dependency for base templates; org standards enforced at compile time |
| Template engine | stdlib `text/template` | Replaces Freemarker; no extra dependency |
| AI feature (v1 only) | Anthropic Wizard mode (`claude-haiku-4-5`) | Conversational flag inference; replaces prompt engine when `--ai` passed |
| Build tool default | Gradle (Kotlin DSL) | Org standard, compiled into base template |
| Java version default | 25 | Org standard, baked into base template |
| Group ID default | `com.myorg` | Org standard, overridable via `--group` |

---

## MVP1 scope (build this, nothing else)

### Commands
- `generate` — the only command that matters for MVP1

### Flags for `generate`

```
--artifact  string   Artifact / project name  (required; prompted if missing)
--group     string   Maven group ID           (default: com.myorg)
--version   string   Project version          (default: 1.0.0)
--java      int      Java version             (default: 25)

--postgres           Add PostgreSQL layer
--dynamo             Add DynamoDB layer
--oracle             Add Oracle layer
(only one DB flag at a time)

--s3                 Add S3 integration layer
--ecs                Add ECS compute layer (Dockerfile + task def scaffold)

--ai                 Replace prompt engine with Claude wizard
--template  string   Remote template override (github.com/owner/repo)
                     Advanced escape hatch — skips embed.FS entirely

--output    string   Output directory (default: ./{artifact})
--dry-run            Print resolved config, write nothing
```

### Interaction modes (all three must work for MVP1)

1. **One-shot** — all required flags passed, zero prompts
2. **Interactive** — bare `generate` or partial flags; `huh` prompts fill the gaps
3. **AI wizard** — `generate --ai`; Claude conversation fills the gaps, then hands off to generate

### Template layers to ship in MVP1

```
templates/
├── base/                        ← always included
│   ├── build.gradle.kts
│   ├── settings.gradle.kts
│   ├── gradle/wrapper/
│   ├── src/main/java/{{.PackagePath}}/
│   │   └── Application.java
│   └── src/main/resources/
│       └── application.yml
├── db/
│   ├── postgres/
│   │   ├── build.gradle.kts.patch   ← appends JPA + postgres driver deps
│   │   └── src/.../PostgresConfig.java
│   ├── dynamo/
│   │   ├── build.gradle.kts.patch   ← appends AWS SDK + DynamoDB
│   │   └── src/.../DynamoConfig.java
│   └── oracle/
│       ├── build.gradle.kts.patch
│       └── src/.../OracleConfig.java
├── aws/
│   └── s3/
│       ├── build.gradle.kts.patch   ← appends AWS SDK S3
│       └── src/.../S3Config.java
└── compute/
    └── ecs/
        ├── Dockerfile
        ├── .dockerignore
        └── ecs-task-def.json.tmpl   ← Go template, uses {{.ArtifactId}} etc.
```

Lambda template is **explicitly out of scope for MVP1**. The `--template` remote override flag
is the escape hatch for it later.

---

## Config struct (source of truth)

```go
// internal/config/config.go

package config

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
```

---

## Package structure

```
localTemplate/
├── cmd/
│   ├── root.go          # cobra root, version flag
│   └── generate.go      # generate subcommand + all flag bindings
├── internal/
│   ├── config/
│   │   ├── config.go    # Config struct
│   │   └── defaults.go  # ApplyDefaults(cfg *Config), Validate(cfg *Config)
│   ├── prompt/
│   │   └── prompt.go    # CollectMissing(cfg *Config) error — huh forms
│   ├── template/
│   │   ├── resolver.go  # ResolveLayers(cfg *Config) []fs.FS
│   │   └── processor.go # Process(layers []fs.FS, cfg *Config, outDir string) error
│   └── ai/
│       ├── client.go    # Client struct, StreamMessage(), parseJSON()
│       ├── wizard.go    # RunWizard(cfg *Config) error
│       └── prompts/
│           └── wizard.txt  # Versioned system prompt (not Go, just text)
├── templates/           # Embedded template tree
│   ├── base/
│   ├── db/
│   ├── aws/
│   └── compute/
├── main.go
└── go.mod
```

---

## Critical implementation details

### embed.FS wiring

```go
// templates.go (at package root or in internal/template/)
package template

import "embed"

//go:embed templates
var TemplateFS embed.FS
```

### Layer resolution order

```go
func ResolveLayers(cfg *config.Config) []string {
    layers := []string{"templates/base"}
    if cfg.Database != "" {
        layers = append(layers, "templates/db/"+cfg.Database)
    }
    for _, r := range cfg.AWSResources {
        layers = append(layers, "templates/aws/"+r)
    }
    if cfg.ECS {
        layers = append(layers, "templates/compute/ecs")
    }
    return layers
}
```

### .patch file semantics (keep it simple for MVP1)

For MVP1, `.patch` files are **append-only**. The processor finds the matching base file
(by stripping `.patch`), appends the patch content after a blank line, then writes the result.
No diff/apply library needed. This works fine for `build.gradle.kts` where deps are just
added to the `dependencies {}` block.

If the base file doesn't exist for a patch, write it as a standalone file (drop the `.patch`
extension).

### Filename template expansion

Before writing any file, expand the filename itself through `text/template`:

```go
// "{{.PackagePath}}/Application.java" → "com/myorg/myservice/Application.java"
func expandPath(tmplPath string, cfg *config.Config) (string, error) {
    t, err := template.New("").Parse(tmplPath)
    // ...
    return buf.String(), nil
}
```

### Prompt engine — only ask what's missing

```go
func CollectMissing(cfg *config.Config) error {
    var fields []huh.Field
    // append a field only if the value is the zero value
    // database select: options are "", "postgres", "dynamo", "oracle"
    // aws multi-select: options are "s3" (sqs is future)
    // ...
    if len(fields) == 0 {
        return nil
    }
    return huh.NewForm(huh.NewGroup(fields...)).Run()
}
```

### AI wizard — contract

The wizard calls `CollectMissing` logic, but instead of `huh` forms it uses a Claude
conversation. The system prompt is in `internal/ai/prompts/wizard.txt`. Expected JSON
schema from Claude per turn:

```json
{
  "config": {
    "artifactId":   "my-service",
    "groupId":      "com.myorg",
    "database":     "postgres",
    "awsResources": ["s3"],
    "ecs":          false
  },
  "followUp": "What should the artifact ID be?"
}
```

`followUp` is `null` when all required fields are populated. Max 5 turns. On any API error
or JSON parse failure (after one retry), fall back to `CollectMissing` with the partially
populated `Config`.

Streaming: print the `followUp` text incrementally as it arrives from the SSE stream,
then read the next line of user input.

### --template remote override

When `cfg.TemplateOverride != ""`, skip `embed.FS` entirely and pull the ZIP from GitHub
using `google/go-github`. Cache to `~/.cache/sb-cli/{owner}/{repo}/{branch}/`. The processor
then walks the cached directory instead of `embed.FS` layers. Same template engine applies.

---

## go.mod dependencies

```
github.com/spf13/cobra               v1.8+
github.com/charmbracelet/huh         v0.6+
github.com/google/go-github/v62      v62+   (for --template remote only)
golang.org/x/oauth2                         (for github auth token)
```

No external dependency for the Anthropic API — use stdlib `net/http` with a thin wrapper.
`text/template` and `embed` are stdlib. Keep the dependency count low.

---

## Deferred to v2 (do not implement in MVP1)

- `list` command
- `validate` command
- `cache` subcommand (just let the OS clear `~/.cache/sb-cli` manually for now)
- Shell completion
- AI scaffold advisor (post-generation review)
- AI template selector
- `--sqs`, `--ses` AWS flags
- Lambda compute layer (`--lambda` + `--template` remote)
- GitLab / Bitbucket source support
- `--explain` flag

---

## Where to start in Code

1. `go mod init github.com/myorg/localTemplate`
2. Wire up `cobra` root + `generate` subcommand with all flags bound to a `Config` struct
3. Implement `config.ApplyDefaults` and `config.Validate`
4. Stub out `templates/base/` with a minimal but runnable Spring Boot skeleton
5. Implement `template/resolver.go` and `template/processor.go` with just the base layer
6. Wire `prompt.CollectMissing` — confirm interactive mode works end-to-end
7. Add `db/postgres` layer — confirm layer merging and `.patch` appending works
8. Add remaining db layers, then `aws/s3`, then `compute/ecs`
9. Implement `ai/client.go` + `ai/wizard.go` — add `--ai` last, it depends on everything above working

Test the happy path at each step before moving to the next.
