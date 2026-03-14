# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

**Contract Scanner IA** is a full-stack intelligent contract analysis platform. It extracts text from PDF contracts (native + OCR fallback) and uses an LLM to identify risks, key clauses, obligations, and provide structured recommendations.

**Stack:**
- **Frontend:** Next.js (React 19, TypeScript, Tailwind CSS, Radix UI)
- **Backend:** Go (Gin framework, Clean Architecture)
- **Database:** PostgreSQL + GORM
- **Storage:** AWS S3
- **LLM:** OpenAI SDK connecting to OpenRouter
- **OCR:** Tesseract + pdftoppm (for scanned PDFs)
- **Auth:** Clerk (currently disabled for local dev)

---

## Commands

### Infrastructure (Docker)
```bash
docker-compose up -d       # Start PostgreSQL + pgAdmin
docker-compose down        # Stop services
```
- PostgreSQL: `localhost:5432` (admin/admin)
- pgAdmin: `localhost:5050`

### Backend (Go)
```bash
cd api
go mod download
go run ./cmd/contract-scanner-api/main.go    # Run dev server (port 8080)
go build -o contract-scanner-api ./cmd/contract-scanner-api  # Build binary

# Tests
go test ./...                                        # All tests
go test ./internal/infra/pdf/pdfpipeline/...        # PDF pipeline tests
go test ./internal/infra/pdf/pdfpipeline/ -run TestClassifier  # Single test
```

### Frontend (Next.js)
```bash
cd UI
npm install        # or yarn
npm run dev        # Dev server (port 3000)
npm run build      # Production build
npm run lint       # ESLint
```

---

## Architecture

### Backend: Clean Architecture Layers

```
handler → usecase → providers (interfaces) → infra (implementations)
```

- **`api/internal/handler/`** — HTTP handlers (Gin). Parses requests, calls usecases, returns JSON.
- **`api/internal/usecase/`** — Business logic. Orchestrates providers/repos without knowing infra details.
- **`api/internal/usecase/providers/`** — Interfaces: `PDFExtractor`, `LLMProvider`, `StorageProvider`.
- **`api/internal/infra/`** — Concrete implementations: S3, PostgreSQL, OpenAI client, PDF pipeline.

### PDF Extraction Pipeline (`api/internal/infra/pdf/pdfpipeline/`)

Hybrid extraction strategy per page:
1. Native text extraction (`ledongthuc/pdf`)
2. Page classification: `native_text` | `scanned_image` | `hybrid` | `unknown`
3. OCR (Tesseract + pdftoppm) for non-native pages
4. Text cleaning, header/footer removal, aggregation, chunking

The pipeline returns a `QualityScore` (0–1); scores below 0.35 trigger warnings.

### Contract Processing Flow

```
POST /api/uploads         → S3 upload + DB record (status=UPLOADED)
POST /api/analyses/:id/process → orchestrated by process_contract.go:
  1. Download PDF from S3 → temp file
  2. Run PDF pipeline (native + OCR)
  3. Call LLM with system prompt (embedded from prompt.md)
  4. Save structured JSON result to DB
  5. Cache extracted text in S3 for retries
```

### Frontend Proxy

Next.js API routes in `UI/app/api/` proxy requests to the Go backend. The `lib/api.ts` module centralises all API calls from the UI.

### LLM System Prompt

Located at `api/cmd/contract-scanner-api/prompt.md` — embedded into the binary at build time via `//go:embed`. Contains detailed instructions and the expected JSON output schema for contract analysis.

### Database Model (`Analyse`)

Key fields: `ID` (UUID), `Status` (UPLOADED → PROCESSING → COMPLETED | FAILED), `S3Key`, `ExtractedTextS3Key`, `ResultJSON` (JSONB), `Model`, `PromptVersion`.

---

## Environment Variables

All config is in `.env` at the project root:

| Variable | Purpose |
|---|---|
| `DB_HOST`, `DB_USER`, `DB_PASSWORD`, `DB_NAME`, `DB_PORT` | PostgreSQL connection |
| `ACCESS_KEY`, `SECRET_ACCESS_KEY`, `AWS_REGION`, `AWS_BUCKET` | S3 storage |
| `LLM_API_KEY`, `LLM_BASE_URL`, `LLM_MODEL` | LLM provider (OpenRouter) |
| `NEXT_PUBLIC_CLERK_PUBLISHABLE_KEY`, `CLERK_SECRET_KEY` | Auth (disabled) |

---

## Agent Team

Sub-agents are defined in `.claude/agents/`. Use them for specialized tasks:

| Agent | When to use |
|---|---|
| `backend-engineer` | Go handlers, usecases, infra implementations, API routes |
| `frontend-engineer` | Next.js pages, React components, UI/app/api proxy routes |
| `pdf-pipeline-engineer` | PDF extraction, OCR, page classification, LLM prompt tuning |
| `infra-engineer` | Docker, PostgreSQL schema/migrations, S3 config, env setup |

---

## Key Patterns

- **OCR requires system dependencies:** Tesseract and pdftoppm must be installed. The `api/Dockerfile` installs them via apk (Alpine).
- **Status guard:** `process_contract.go` skips re-processing if status is already `PROCESSING`.
- **Result enrichment:** After LLM response, the handler enriches the JSON with PDF extraction metadata (`analysis_warnings`, `confidence`).
- **Auth middleware** (`clerk_auth.go`) is wired but disabled — routes are public in `routes.go`.
