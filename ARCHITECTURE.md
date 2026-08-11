# Architecture — `local-template-generator`

> **Status:** Design / planning  
> **Language:** Go 1.22+  
> **AI integration:** Anthropic API (claude-haiku-4-5) — Wizard mode only (v1)

---

## Overview

`localtemplate` is a Go command-line tool that generates customised Spring Boot projects from **bundled templates** shipped inside the binary. Users either pass flags directly for a one-shot invocation, or run the bare command for a guided interactive prompt sequence. An optional AI wizard (`--ai`) replaces the prompts with a free-text conversational interface.

The Spring Boot web server is removed entirely. There is no HTTP server, no REST API, no frontend. The CLI binary is the only artifact.

---

## System Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                     CLI entry point                         │
│         main.go · cobra · flag parsing                      │
│                                                             │
│  one-shot:    generate --postgres --s3 --artifact my-svc    │
│  interactive: generate          (no flags → prompt mode)    │
│  ai mode:     generate --ai     (free-text wizard)          │
└──────────────────────────┬──────────────────────────────────┘
                           │
┌──────────────────────────▼──────────────────────────────────┐
│                    Command layer                            │
│         generate │ list │ validate │ cache                  │
└──────┬────────────────────────┬────────────────────────────-┘
       │                        │
       ▼                        ▼
┌──────────────────┐   ┌────────────────────────────────────┐
│  Prompt engine   │   │           Core services            │
│  (huh / survey)  │   │                                    │
│                  │   │  ┌────────────────────────────┐    │
│  Interactive     │   │  │  Template resolver         │    │
│  multi-step      │   │  │  embed.FS (bundled)        │    │
│  form when flags │   │  │  optional --template URL   │    │
│  are missing     │   │  └────────────────────────────┘    │
│                  │   │                                    │
│  Skips answered  │   │  ┌────────────────────────────┐    │
│  fields when     │   │  │  Template processor        │    │
│  flags present   │   │  │  text/template engine      │    │
│                  │   │  │  File walk / copy          │    │
└────────┬─────────┘   │  │  Binary detection          │    │
         │             │  └────────────────────────────┘    │
         └──────────── │                                    │
                       │  ┌────────────────────────────┐    │
                       │  │  Config builder            │    │
                       │  │  flags + prompt → Config   │    │
                       │  │  org defaults baked in     │    │
                       │  │  dependency resolver       │    │
                       │  └────────────────────────────┘    │
                       └──────────────┬─────────────────────┘
                                      │
            ┌─────────────────────────▼───────────────────────┐
            │        Anthropic AI layer  (opt-in, --ai)       │
            │                                                  │
            │   Wizard mode — conversational flag inference    │
            │   claude-haiku-4-5 · streaming · JSON output    │
            │   Replaces prompt engine when --ai is passed    │
            └─────────────────────────┬───────────────────────┘
                                      │
            ┌─────────────────────────▼───────────────────────┐
            │              External integrations              │
            │  embed.FS (bundled templates, primary)          │
            │  GitHub API (optional --template override)      │
            │  Anthropic API (--ai only)                      │
            │  File system  (~/.cache/sb-cli, output dir)     │
            └─────────────────────────────────────────────────┘
```

---

## Template source — bundled vs remote

Templates live **inside this repository** under `templates/` and are compiled into the binary via Go's `embed.FS`. There is no external GitHub dependency for normal use.

```
templates/
├── base/                    ← always included
│   ├── build.gradle.kts
│   ├── settings.gradle.kts
│   ├── src/main/java/{{.PackagePath}}/
│   │   └── Application.java
│   └── src/main/resources/
│       └── application.yml
├── db/
│   ├── postgres/            ← merged when --postgres
│   │   ├── build.gradle.kts.patch
│   │   └── src/.../PostgresConfig.java
│   ├── dynamo/              ← merged when --dynamo
│   └── oracle/              ← merged when --oracle
└── aws/
    ├── s3/                  ← merged when --s3
    └── sqs/                 ← merged when --sqs (future)
```

**Org defaults baked in** — the base template enforces your standards by default:

| Setting | Default | Override flag |
|---|---|---|
| Build tool | Gradle (Kotlin DSL) | — (no Maven option in v1) |
| Java version | 25 | `--java <version>` |
| Spring Boot | latest stable | `--spring-boot <version>` |
| Group ID | `com.myorg` | `--group <id>` |
| Packaging | JAR | — |

The `--template github.com/owner/repo` flag remains available for teams with custom template repos, but it is an advanced escape hatch, not the primary path.

---

## Data flow

```
1. User invokes the CLI
   ↓
2. Flag parser (cobra) → partial or complete Config
   ↓
3a. [flags complete]   → skip to step 5
3b. [flags missing, no --ai] → Prompt engine fills gaps interactively
3c. [--ai flag]        → AI Wizard fills gaps conversationally
   ↓
4. Config is now complete and validated
   ↓
5. Template resolver selects layers from embed.FS
   base + db/{database} + aws/{resource...}
   ↓
6. Template processor walks and merges layers
   - text/template substitution on text files
   - Binary files copied unchanged
   - .patch files applied to base counterparts
   ↓
7. Write final project tree to ./{artifactId}/
   ↓
8. Print summary: files written, next steps
```

---

## Dual-mode UX — interactive vs one-shot

The same `generate` command supports both modes. The rule is simple: **any required field not supplied as a flag triggers a prompt for that field only**.

### Interactive mode (no flags)

```
$ localtemplate generate

  Project artifact ID: my-service
  Group ID [com.myorg]: ⏎

  Do you need database access?
  ❯ None
    PostgreSQL
    DynamoDB
    Oracle

  Do you need AWS resource integration?
  ❯ None
    S3
    SQS
    S3 + SQS

  Output directory [./my-service]: ⏎

  Generating...  ✓ Done
```

Questions are rendered as interactive selects / inputs using `charmbracelet/huh`. Arrow keys to navigate, Enter to confirm.

### One-shot mode (all flags)

```bash
localtemplate generate \
  --artifact my-service \
  --group com.acme \
  --postgres \
  --s3 \
  --output ./my-service
```

No prompts. Suitable for CI pipelines or scripting.

### Mixed mode (some flags)

```bash
localtemplate generate --postgres
# Prompts only for: artifact ID, group ID, AWS resources, output dir
# Skips the database question (already answered by --postgres)
```

### AI wizard mode

```bash
localtemplate generate --ai
```

The prompt engine is replaced entirely by a conversational loop powered by Claude. See the AI section below.

---

## Component details

### CLI entry / command layer

Built with [cobra](https://github.com/spf13/cobra).

```
localtemplate
  generate          Generate a project (interactive, one-shot, or --ai)
  list              List bundled templates and layers
  validate          Validate a custom template directory
  cache             Manage remote template cache (for --template overrides)
    clear
    list
```

### Prompt engine (`internal/prompt/`)

Wraps `charmbracelet/huh`. Receives the partially-populated `Config` and renders only the questions whose fields are zero-valued.

```go
func CollectMissing(cfg *config.Config) error {
    var fields []huh.Field

    if cfg.ArtifactId == "" {
        fields = append(fields, huh.NewInput().
            Title("Project artifact ID").
            Value(&cfg.ArtifactId))
    }
    if cfg.GroupId == "" {
        fields = append(fields, huh.NewInput().
            Title("Group ID").
            Placeholder("com.myorg").
            Value(&cfg.GroupId))
    }
    if cfg.Database == "" {
        fields = append(fields, huh.NewSelect[string]().
            Title("Do you need database access?").
            Options(
                huh.NewOption("None", ""),
                huh.NewOption("PostgreSQL", "postgres"),
                huh.NewOption("DynamoDB", "dynamo"),
                huh.NewOption("Oracle", "oracle"),
            ).
            Value(&cfg.Database))
    }
    if len(cfg.AWSResources) == 0 {
        fields = append(fields, huh.NewMultiSelect[string]().
            Title("Do you need AWS resource integration?").
            Options(
                huh.NewOption("None / skip", ""),
                huh.NewOption("S3", "s3"),
                huh.NewOption("SQS", "sqs"),
            ).
            Value(&cfg.AWSResources))
    }

    if len(fields) == 0 {
        return nil  // everything already set by flags
    }
    return huh.NewForm(huh.NewGroup(fields...)).Run()
}
```

### Template resolver (`internal/template/resolver.go`)

Selects which layers from `embed.FS` to merge based on `Config`:

```go
layers := []string{"base"}
if cfg.Database != "" {
    layers = append(layers, "db/"+cfg.Database)
}
for _, r := range cfg.AWSResources {
    layers = append(layers, "aws/"+r)
}
```

Layers are merged in order. Later layers can patch files from earlier ones using `.patch` files (applied with `go-patch` or simple append semantics for `build.gradle.kts`).

### Template processor (`internal/template/processor.go`)

| Responsibility | Detail |
|---|---|
| Engine | stdlib `text/template` |
| File walk | `fs.WalkDir` over the resolved layer set |
| Filename templating | `{{.ArtifactId}}` in directory and file names is expanded |
| Text file types | `.java .kt .xml .yml .yaml .properties .gradle .kts .md .sql .json .sh` |
| Binary passthrough | Everything else copied unchanged |

### Config builder (`internal/config/`)

```go
type Config struct {
    // Project identity
    ArtifactId  string
    GroupId     string   // default: "com.myorg"
    Version     string   // default: "1.0.0"
    PackageName string   // derived: GroupId + "." + ArtifactId (sanitised)

    // Build
    JavaVersion      int    // default: 25
    SpringBootVersion string // default: resolved from catalog

    // Integrations
    Database     string   // "postgres" | "dynamo" | "oracle" | ""
    AWSResources []string // ["s3"] | ["s3","sqs"] | []

    // Output
    OutputDir string // default: "./{ArtifactId}"

    // Modes
    AIEnabled    bool
    TemplateOverride string // github URL, empty = use bundled
}
```

**Org defaults** are applied as zero-value fallbacks in the builder before prompts run, so users only see a prompt when the default is genuinely not appropriate.

---

## Project structure

```
localtemplate/
├── cmd/
│   ├── root.go          # cobra root, persistent flags
│   └── generate.go      # generate subcommand
├── internal/
│   ├── config/          # Config struct, flag binding, defaults, validation
│   ├── prompt/          # huh-based interactive prompt engine
│   ├── template/
│   │   ├── resolver.go  # layer selection from embed.FS
│   │   └── processor.go # walk, substitute, copy
│   └── ai/
│       ├── client.go    # thin Anthropic API wrapper (streaming)
│       └── wizard.go    # conversational wizard (v1 AI feature)
├── templates/           # bundled project templates (embed.FS)
│   ├── base/
│   ├── db/
│   │   ├── postgres/
│   │   ├── dynamo/
│   │   └── oracle/
│   └── aws/
│       ├── s3/
│       └── sqs/
├── main.go
└── go.mod
```

---

## AI integration — v1 (Wizard mode only)

The AI layer is **opt-in** (`--ai` flag). The tool works fully offline without it.

When `--ai` is passed, the wizard replaces the `huh` prompt sequence entirely. The user describes their project in plain English; Claude extracts the flag values and asks follow-up questions until the `Config` is complete.

```
$ localtemplate generate --ai

  Describe the service you want to build:
  > A REST API that stores user events in DynamoDB and sends
    notification emails via SES. Use com.acme as the group.

  Got it. I'll configure:
    Database  →  DynamoDB
    AWS       →  SES  (I'll add this as a custom layer)
    Group     →  com.acme
    Java      →  25  (org default)

  What should the artifact ID be?
  > event-tracker

  Generating event-tracker...  ✓ Done
```

### Implementation

```
User utterance
    ↓
System prompt (versioned, stored in internal/ai/prompts/wizard.txt):
  "You are a Spring Boot project setup assistant for <org>.
   Extract: database (postgres|dynamo|oracle|none),
   awsResources (list of: s3|sqs|ses|none),
   groupId, artifactId, javaVersion.
   Org defaults: groupId=com.myorg, javaVersion=25, build=gradle.
   Respond ONLY with JSON: { config: {...}, followUp: '...' | null }
   followUp is null when all required fields are populated."
    ↓
Parse JSON → merge into Config
    ↓
If followUp != null → print it, read next user input, loop
If followUp == null → Config complete → proceed to generate
```

**Conversation history** is maintained across turns so corrections work naturally ("actually use postgres not dynamo").

| Parameter | Value |
|---|---|
| Model | `claude-haiku-4-5` |
| Max turns | 5 |
| Max output tokens | 512 per turn |
| Streaming | Yes — follow-up questions appear immediately |
| Fallback | If API call fails → fall back to `huh` prompt engine transparently |

### Prompt hygiene

- System prompt lives in `internal/ai/prompts/wizard.txt` — versioned in git, never concatenated at runtime.
- The model only ever sees: the org defaults, the user's text, and the JSON schema. It never sees file contents or template internals.
- JSON parse errors retry once with a nudge (`"Respond only with valid JSON"`), then fall back to the prompt engine.

---

## Technology stack

| Layer | Technology | Replaces |
|---|---|---|
| Language | Go 1.22+ | Java 17 |
| CLI framework | cobra | Spring MVC controllers |
| Interactive prompts | charmbracelet/huh | — (new) |
| Templating | stdlib `text/template` | Freemarker |
| Template storage | stdlib `embed.FS` | GitHub API / Kohsuke |
| AI (wizard) | Anthropic API — haiku-4-5 | — (new) |
| Build | `go build` → single binary | Maven JAR |
| Distribution | GitHub Releases / Homebrew tap | Docker / Beanstalk |

---

## Security

| Concern | Approach |
|---|---|
| Anthropic key | `ANTHROPIC_API_KEY` env var only — never a flag |
| GitHub token | `GITHUB_TOKEN` env var — only needed for `--template` override |
| Path traversal | `filepath.Clean` + output-dir prefix assertion on every write |
| Input validation | Regex `^[a-z][a-z0-9-]+$` on artifact/group segments |
| embed.FS | Templates are read-only at compile time; no runtime writes to the binary |

---

## Configuration

```bash
# Required only for --ai
export ANTHROPIC_API_KEY=sk-ant-...

# Required only for --template github.com/... override with private repos
export GITHUB_TOKEN=ghp_...

# Optional overrides
export SB_CLI_OUTPUT_DIR=./output   # default: ./{artifactId}
export SB_CLI_CACHE_DIR=~/.cache/sb-cli
```

No config file is required. Org defaults (Java 25, Gradle, `com.myorg`) are compiled in and overridable via flags.

---

## Error handling

```
CLIError
  Code     TEMPLATE_NOT_FOUND | RENDER_FAILED | AI_TIMEOUT |
           VALIDATION_FAILED  | UNKNOWN_LAYER
  Message  human-readable description
  Hint     one-line recovery suggestion
```

Errors print to stderr. Exit code 1 = error, 2 = bad usage. AI failures silently fall back to the prompt engine rather than hard-failing — the user sees a brief notice and continues.

---

## Future enhancements (v2+)

| Priority | Feature |
|---|---|
| High | Shell completion (cobra built-in) |
| High | `--dry-run` — print resolved variable map, write nothing |
| Medium | **AI: scaffold advisor** — post-generation review streamed to stdout |
| Medium | **AI: template selector** — rank layers when ambiguity exists |
| Medium | SQS, SES, SNS layers in `templates/aws/` |
| Medium | `--template gitlab.com/...` source support |
| Low | TUI mode (bubbletea) for richer interactive experience |
| Low | `--explain` — AI narrates what each generated file does |
