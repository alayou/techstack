# Techstack

[中文 README](README.zh-CN.md)

Techstack is a repository intelligence and AI-governance service for teams that do not want AI to silently drag projects into unfamiliar frameworks. It imports public repositories, extracts dependencies, syncs package metadata, and generates structured technical analysis. Its product intent is explicit: constrain agents to the developer's existing stack, force understanding over blind generation, reject black-box engineering, and let AI write code only within the team's current knowledge boundary.

## Product Intent

- Keep AI inside the stack a developer or team already understands.
- Turn dependencies, repo structure, and technical decisions into something inspectable before code generation.
- Use AI as an amplifier for learning and implementation, not as a shortcut around understanding.
- Make agent integrations boundary-aware: an agent should read the team's known stack before it writes code.

## Current Status

Techstack is not finished yet. The current repository is the minimum viable version of the product.

Today, the system is mainly focused on four things:

- importing public repositories into a unified system
- building a global package catalog across multiple ecosystems
- extracting real dependency and stack signals from codebases
- generating structured AI-readable repository analysis for later agent use

This means the current release already establishes the data layer and stack-boundary model, but it is not yet the full product vision for agent-native development workflows.

## What It Does

- Maintains a global package catalog for `npm`, `golang`, `pypi`, and `cargo`.
- Imports public GitHub repositories into the system.
- Clones repositories with `go-git` and scans dependencies with `osv-scalibr`.
- Tracks repo dependencies and links them to the internal package catalog.
- Runs asynchronous jobs for repository import and AI analysis.
- Generates structured repo reports through an OpenAI-compatible LLM endpoint.
- Serves an HTTP API and an embedded static web UI from the same binary.
- Supports login-based access and optional AK/SK signing for agent or tool integration.

## Roadmap

The long-term direction is to turn Techstack from a repository analysis service into a practical stack-governance layer for code agents.

Planned work includes:

- adding an MCP service layer so IDEs, agents, and automation tools can query stack knowledge through a standard interface
- letting agents discover high-quality packages and repositories automatically, while still keeping developers in control of adoption
- allowing developers to favorite packages and repositories as explicit preference signals
- classifying AI-related technical-stack repositories into clearer categories for search, recommendation, and governance
- connecting repository intelligence to real development workflows, including project-oriented skill generation for code agents
- syncing and updating technical-stack data and developer preferences in near real time
- making code agents prefer developer-approved or personally favored packages and repositories when generating code

## Development Task Checklist

The following checklist turns the roadmap into concrete product and engineering work items:

- [ ] Add an MCP service layer so code agents, IDEs, and automation tools can query package, repository, preference, and analysis data through a standard protocol.
- [ ] Support agent-driven discovery of high-quality packages and repositories, with reviewable recommendation results instead of silent adoption.
- [ ] Add developer favorites for repositories and packages, and use those favorites as explicit preference signals inside the system.
- [ ] Build classification for AI-related technical-stack repositories so they can be grouped by role, domain, framework, and usage pattern.
- [ ] Connect repository intelligence to real development workflows, including generating project-oriented skills for code agents from known repos and stack context.
- [ ] Keep technical-stack knowledge and developer preferences synced and updated in near real time as repos, dependencies, and favorites change.
- [ ] Make code agents prefer developer-approved or personally favored packages and repositories when generating code.
- [ ] Link package records with security data sources and produce security analysis so stack decisions include vulnerability and risk signals.

## Why This Matters For Agents

The service is designed to become a stack boundary for coding agents.

1. Import repositories that represent your existing engineering practice.
2. Extract the real dependencies and stack signals from source code.
3. Let the agent inspect those signals first.
4. Ask the agent to generate code only inside that already-known technical space.

The result is a more deliberate workflow: developers still learn the stack, while the agent accelerates implementation inside a readable and auditable boundary.

## Architecture

- `daemon/`: service bootstrap, HTTP server startup, worker pool lifecycle.
- `httpd/`: API handlers, middleware, auth, settings, and HTTP utilities.
- `httpd/taskmgr/`: async repo import and repo analysis jobs.
- `pkg/repofs/`: repository clone, in-memory file system conversion, and SCALIBR scanning.
- `pkg/pkgclient/`: package metadata clients for upstream ecosystems.
- `model/`: GORM models for users, packages, repos, settings, and tasks.
- `web/`: embedded static frontend assets served by the backend binary.

## Repo Analysis Pipeline

1. A public GitHub repository is imported into the system.
2. The backend clones the repo into an in-memory file system.
3. `osv-scalibr` scans the repo with plugins for `go`, `python`, `javascript`, and `rust`.
4. Dependency records are stored as repo dependencies and linked to internal packages.
5. An AI analysis task reads repository files through tool-based access and writes back a structured report.

The generated report is meant to answer questions such as:

- What is this project for?
- What problem does it solve?
- What stack and dependencies does it really use?
- What does the directory structure imply?
- What are the core APIs, strengths, weaknesses, and appropriate use cases?

## Supported Inputs

- Dependency ecosystems: `npm`, `golang`, `pypi`, `cargo`
- Dependency files parsed directly in the codebase: `go.mod`, `package.json`, `pyproject.toml`, `requirements.txt`, `Cargo.toml`
- Public repository import: currently GitHub repositories only
- LLM provider: OpenAI-compatible endpoint configured through `llm.baseUrl`, `llm.apiKey`, and `llm.model`

## Quick Start

### Requirements

- Go toolchain compatible with `go.mod` (`go 1.25.8`)
- PostgreSQL or another database config you have already validated for this project
- An OpenAI-compatible LLM endpoint

### Configure

Copy the example config and fill in the required values:

```bash
cp config.yml.tpl config.yml
```

Minimal example:

```yaml
addr: :8775
debug: false
database:
  driver: postgres
  dsn: user=postgres password=changeme dbname=techstack host=127.0.0.1 port=5432 sslmode=disable TimeZone=Asia/Shanghai
session_hash_key: "replace-with-16-char-key"
session_cookie_key: "replace-with-16-char-key"
llm:
  apiKey: "YOUR_API_KEY"
  baseUrl: "https://api.minimaxi.com/v1"
  provider: "openai"
  model: "MiniMax-M2.7"
```

Important fields:

- `database.*`: database connection used by GORM.
- `session_hash_key` and `session_cookie_key`: required for session and cookie signing.
- `llm.*`: required for AI repo analysis.
- `debug`: in debug mode the service enables permissive CORS headers. Do not keep this on in production.

### Run

Build and start:

```bash
go build -o techstack .
./techstack daemon
```

Or run directly:

```bash
go run . daemon
```

The sample config listens on `:8775`.

### First Startup

On first startup the service:

- runs `AutoMigrate` for all models
- initializes default system settings
- creates a default admin account: `admin / techstack`
- generates `account_key` and `account_secret` for each created user

Change the default admin password immediately.

## API Overview

Public endpoints:

- `GET /api/v1/ping`
- `GET /api/v1/health`

Authentication and profile:

- `POST /api/v1/c/login`
- `POST /api/v1/c/signup`
- `POST /api/v1/c/logout`
- `GET /api/v1/c/profile`
- `PUT /api/v1/c/profile`
- `PUT /api/v1/c/user/password`

Global libraries:

- `GET /api/v1/c/libraries`
- `POST /api/v1/c/libraries`
- `POST /api/v1/c/libraries/batch`
- `GET /api/v1/c/libraries/{package_id}`
- `GET /api/v1/c/libraries/{package_id}/versions`
- `POST /api/v1/c/libraries/{package_id}/sync-versions`
- `GET /api/v1/c/libraries/get/purl`

Public repositories and tasks:

- `GET /api/v1/c/public-repos`
- `POST /api/v1/c/public-repos/import`
- `GET /api/v1/c/public-repos/{id}`
- `GET /api/v1/c/tasks`
- `GET /api/v1/c/tasks/{id}`
- `POST /api/v1/c/tasks/{id}/retry`

Settings:

- `GET /api/v1/c/setting`
- `PUT /api/v1/c/setting`
- `GET /api/v1/c/setting/basic`
- `PUT /api/v1/c/setting/basic`

Response conventions:

- List responses are returned as `{"list":[...],"total":N}`.
- Error responses are returned as `{"code":400,"msg":"...","error":"..."}`.
- `GET /api/v1/health` returns `health`, `revision`, `version`, and `build_at`.

## Basic Usage Example

Health check:

```bash
curl http://127.0.0.1:8775/api/v1/health
```

Login with the bootstrap admin:

```bash
curl -X POST http://127.0.0.1:8775/api/v1/c/login \
  -H 'Content-Type: application/json' \
  -d '{"username":"admin","password":"techstack"}'
```

Create a package entry:

```bash
curl -X POST http://127.0.0.1:8775/api/v1/c/libraries \
  -H 'Content-Type: application/json' \
  -H "Authorization: Bearer <token>" \
  -d '{"name":"gin","purl_type":"golang"}'
```

Import a GitHub repository:

```bash
curl -X POST http://127.0.0.1:8775/api/v1/c/public-repos/import \
  -H 'Content-Type: application/json' \
  -H "Authorization: Bearer <token>" \
  -d '{"repo_url":"https://github.com/gin-gonic/gin"}'
```

Repository import creates async work. Use the task endpoints to inspect progress, failures, and retries.

## Agent Integration

Techstack is intended to sit between the developer and the agent.

- A human developer defines the acceptable stack by importing repos and curating packages.
- The service turns that stack into structured data the agent can query.
- The agent can then inspect known dependencies, repo analysis, and technical patterns before generating code.

For non-browser clients, protected APIs also support AK/SK signing with:

- `x-aksk-version: 1`
- `x-access-key`
- `x-timestamp`
- `x-signature`

The signature is verified as HMAC-SHA256 over:

```text
{METHOD}&{PATH}&{sorted_query_params_without_signature}&{timestamp}
```

This is useful when integrating external agents or internal automation without relying on browser cookies.

## Service Mode

The binary also supports service management commands:

```bash
./techstack service install --config /absolute/path/to/config.yml
./techstack service start --config /absolute/path/to/config.yml
./techstack service stop --config /absolute/path/to/config.yml
./techstack service uninstall --config /absolute/path/to/config.yml
```

Passing `--config` explicitly is recommended.

## Docker

Build the binary expected by the Dockerfile, then build the image:

```bash
go build -o build/techstack .
docker build -t techstack:local .
docker run --rm -p 8775:8775 \
  -v "$(pwd)/config.yml:/opt/techstack/config.yml" \
  techstack:local
```

## Cross-Platform Build Script

The repository includes `build-binary.sh`, which builds:

- `build/techstack.exe`
- `build/techstack-linux`
- `build/techstack`
- `build/techstack-linux-arm64`

Example:

```bash
./build-binary.sh v0.1.0
```

## Production Notes

- Change the default admin password immediately after first login.
- Replace the example session keys with random values.
- Set `debug: false` in production.
- Use real TLS and CORS settings for non-local deployments.
- Do not treat the AI analysis as ground truth; use it as a structured starting point.
- The product intent is not "AI can write anything". The intended model is "AI writes inside a known stack, and the developer remains responsible for understanding it".

## License

See [LICENSE](LICENSE).
