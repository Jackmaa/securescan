# 🔐 SecureScan

[![License](https://img.shields.io/badge/license-MIT-blue.svg)](LICENSE)
[![Go](https://img.shields.io/badge/Go-1.23%2B-00ADD8.svg)](https://go.dev/)
[![Fiber](https://img.shields.io/badge/Fiber-v3-00ACD7.svg)](https://gofiber.io/)
[![SvelteKit](https://img.shields.io/badge/SvelteKit-2-orange.svg)](https://kit.svelte.dev/)
[![PostgreSQL](https://img.shields.io/badge/PostgreSQL-18-336791.svg)](https://www.postgresql.org/)

> **A web platform that orchestrates open-source security tools against submitted codebases, aggregates results, maps them to OWASP Top 10:2025, and proposes fixes — with GitHub PR automation.**

Stop juggling CLIs and copy/pasting findings. Submit a repo, watch tools run concurrently, review an OWASP-mapped dashboard, accept fixes, and ship a PR.

---

## 🎯 Why SecureScan?

Security scanning is easy. **Security scanning that’s actionable** is not.

SecureScan is built to:

- Run multiple tools concurrently (SAST, secrets, dependency audits)
- Normalize + persist findings in one schema
- Map everything to **OWASP Top 10:2025** (with confidence tiers)
- Generate **template fixes** (and optional AI fixes)
- Apply accepted fixes into a new branch and open a PR
- Produce an HTML/PDF report for sharing

---

## ✨ Features

- ✅ **Project submission** — scan by Git URL (ZIP upload planned)
- ✅ **Concurrent multi-tool scanning** — Semgrep, TruffleHog, npm audit, ESLint Security
- ✅ **OWASP Top 10:2025 mapping** — direct metadata → CWE lookup → heuristic fallback
- ✅ **Dashboard-ready stats** — score/grade, severity breakdown, OWASP breakdown, tool breakdown
- ✅ **Real-time progress** — Server-Sent Events (SSE) stream during scans
- ✅ **Fix engine** — template-based fixes + optional Claude-powered fixes
- ✅ **Fix review workflow** — accept/reject per fix, bulk actions planned
- ✅ **Git integration** — branch → apply fixes (reverse line order) → commit → push → PR
- ✅ **Report generation** — HTML template → PDF (chromedp)

---

## 🧰 Security Tools

SecureScan currently orchestrates:

| Tool | Purpose | How it runs |
|---|---|---|
| **Semgrep** | SAST (injection/XSS/auth/crypto/access control, etc.) | `semgrep --config p/owasp-top-ten --json` |
| **TruffleHog** | secrets detection (keys/tokens/credentials) | `trufflehog filesystem ... --json` |
| **npm audit** | vulnerable Node/JS dependencies | `npm audit --json` |
| **ESLint Security** | risky JS patterns (eval, unsafe regex, object injection, etc.) | `eslint --plugin security --format json` |

---

## 📦 Requirements

### Core

- Go (for `api/`)
- Bun (for `web/`)
- Docker (for the included Postgres dev DB via `make db`)
- PostgreSQL client tools optional (not required)

### Toolchain (required for scanning)

SecureScan shells out to local CLIs:

- `semgrep`
- `trufflehog`
- `npm`
- `eslint` + `eslint-plugin-security`

If a tool is missing or fails, the orchestrator continues and you’ll see partial results.

---

## 🚀 Quickstart (Dev)

### 1) Configure environment

Copy and edit `.env.example` into `.env`:

```bash
cp .env.example .env
```

The defaults assume a local Docker Postgres container:

```env
DB_HOST=localhost
DB_PORT=5432
DB_USER=securescan
DB_PASSWORD=securescan
DB_NAME=securescan
DB_SSLMODE=disable

API_PORT=3000
FRONTEND_URL=http://localhost:5173

SCAN_WORKSPACE=/tmp/securescan

GITHUB_TOKEN=
ANTHROPIC_API_KEY=
```

### 2) Install frontend deps

```bash
cd web && bun install
```

### 3) Start everything

From repo root:

```bash
make dev
```

This runs:

- Postgres (Docker): `make db` (binds `5432`)
- API (hot reload): `make api` (Fiber on `:3000`)
- Frontend: `make web` (SvelteKit on `:5173`)

### 4) Run migrations

In a separate terminal:

```bash
make migrate
```

### 5) Trigger a test scan (Juice Shop)

```bash
make test-scan
```

---

## ⚙️ Configuration

### GitHub token (PR creation)

Set `GITHUB_TOKEN` in `.env` to enable PR creation during “apply fixes”.

### Claude API key (AI fixes)

Set `ANTHROPIC_API_KEY` in `.env` to enable AI-generated fixes. SecureScan is designed to:

- try template fixes first (fast, deterministic, free)
- offer AI as an optional “smarter fix” per finding

---

## 🧠 How OWASP Mapping Works

SecureScan maps findings using a priority chain:

1. **Direct tool metadata** — e.g. Semgrep OWASP tags (highest confidence)
2. **CWE → OWASP lookup** — static mapping table (when CWE is present)
3. **Heuristic fallback** — rule name / tool-specific logic (best-effort)

---

## 🧮 Scoring

The score is designed to be simple and explainable:

\[
\text{penalty}=\sum (\text{critical}\cdot10+\text{high}\cdot5+\text{medium}\cdot2+\text{low}\cdot0.5)
\]
\[
\text{score}=\text{clamp}(100-\text{penalty}, 0, 100)
\]

Grade mapping:

- A (≥ 90)
- B (≥ 75)
- C (≥ 55)
- D (≥ 35)
- F (< 35)

---

## 🔌 API Overview

Base URL (dev): `http://localhost:3000`

### Projects

- `POST /api/projects` — submit a project (git URL)
- `GET /api/projects/:id` — project details

### Scans

- `POST /api/projects/:id/scan` — start a scan
- `GET /api/scans/:id` — scan status + summary
- `GET /api/scans/:id/progress` — SSE scan progress stream
- `GET /api/scans/:id/stats` — aggregated dashboard stats
- `GET /api/scans/:id/findings?severity=&owasp=&tool=&sort=&page=&limit=` — findings (filtered)
- `GET /api/scans/:id/fixes` — generated fixes

### Fix workflow

- `POST /api/fixes/:id/accept` — accept a fix
- `POST /api/fixes/:id/reject` — reject a fix
- `POST /api/fixes/bulk` — bulk accept/reject

### Git integration

- `POST /api/scans/:id/apply-fixes` — branch → apply → commit → push → PR
- `GET /api/scans/:id/apply-status` — SSE progress stream for git operations

### Reports

- `POST /api/scans/:id/report` — generate report (HTML/PDF)
- `GET /api/scans/:id/report/download` — download report

### AI (optional)

- `POST /api/scans/:id/ai-fix/:findingId` — generate AI fix for a finding

---

## 🗂️ Repo Layout

```text
securescan/
├── api/        # Go + Fiber backend (scan orchestration, mapping, fixes, git, reports)
├── web/        # SvelteKit frontend (dashboard, progress, fix review, report)
├── testdata/   # sample vulnerable repos (optional)
├── Makefile
├── .env.example
└── README.md
```

---

## 🛠️ Common Commands

```bash
make dev         # start db + api + web
make db          # run Postgres in Docker
make db-stop     # stop Postgres container
make migrate     # run SQL migrations
make test-scan   # create a project (juice-shop) via API
make clean       # remove /tmp/securescan + pgdata/
```

---

## 🛣️ Roadmap

### ✅ MVP

- [x] Git URL submission
- [x] 4-tool scanning (Semgrep, TruffleHog, npm audit, ESLint Security)
- [x] OWASP 2025 mapping + scoring
- [x] SSE scan progress
- [x] Fix generation (template + AI-ready architecture)
- [x] Git apply flow (branch/commit/push/PR)
- [x] Report generation (HTML/PDF)

### 🚧 Next

- [ ] ZIP upload support
- [ ] Better UI states (empty/error/loading polish)
- [ ] Bulk accept/reject UX polish
- [ ] Additional tooling (composer audit, trivy, gitleaks, etc.)

---

## 🤝 Contributing

This is a personal learning project, but contributions are welcome.

If you’re adding a new scanner:

- implement a `ToolAdapter` in `api/scanner/`
- ensure it’s safe to run concurrently
- make parsing resilient (tools often return non-zero on “findings found”)
- map to OWASP via direct metadata, CWE, or a documented heuristic

---

## 📝 License

This project is licensed under the MIT License. See [LICENSE](LICENSE) for details.
