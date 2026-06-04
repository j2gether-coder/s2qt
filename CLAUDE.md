# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

S2QT (Sermon to Quiet Time) is a Windows desktop application that converts sermon content (text, audio, video/URL) into Quiet Time devotional materials using AI. Built with **Wails v2** (Go 1.25 backend + React 19 frontend).

UI language is Korean (한글).

## Build & Development Commands

```bash
# Development with live reload
wails dev

# Production build (outputs bin/s2qt.exe)
wails build

# Frontend only
cd frontend && npm install    # Install frontend dependencies
cd frontend && npm run dev    # Vite dev server
cd frontend && npm run build  # Build to frontend/dist/

# Run Go tests
go test ./service/...         # Service tests
go test ./service/ -run TestName  # Single test

# Windows installer
# Uses Inno Setup with s2qt_setup.iss
```

## Architecture

### Wails IPC Bridge

`app.go` defines the `App` struct with ~54 exported methods callable from the React frontend via Wails-generated bindings (`frontend/src/wailsjs/go/main/App.js`). This is the primary integration point between Go and JS.

### 3-Step QT Workflow

The core user flow is a pipeline through three steps:

1. **Step 1 - Source Preparation**: Input (text file, audio file, or video URL) → transcription/extraction → `var/temp/temp.txt`
2. **Step 2 - LLM Processing & Editing**: Raw text → LLM prompt → structured JSON (`var/temp/temp.json`) → manual editing via CKEditor 5
3. **Step 3 - Output Generation**: JSON → multi-format output (HTML, PDF, DOCX, PPTX, PNG with QR codes)

### Backend Services (`service/`)

Business logic is organized into service files, all operating under the `service` package. Key services:
- `pdf_service.go` / `png_service.go` / `word_service.go` / `ppt_service.go` — Output format generators
- `template_service.go` — Output template management
- `prompt_qt_json.go` — LLM prompt construction
- `llm_service.go` — LLM API integration
- `audio_service.go` / `video_service.go` — Media processing (delegates to ffmpeg/whisper/yt-dlp)
- `db_service.go` / `history_service.go` — SQLite persistence
- `crypto_service.go` — Encryption and PIN security
- `types.go` — Shared data structures

### Frontend Components (`frontend/src/components/`)

- `appShell.js` → `sideNav.js` + `mainWorkspace.js`
- `qt/qtStep1.js`, `qt/qtStep2.js`, `qt/qtStep3.js` — Step UI components
- `qt/bindQTStep*.js` — Event binding layers between UI and Wails IPC
- `settings/` — Configuration panels
- `appState.js` — Centralized React state management (useState-based)

### External Binaries (`bin/`)

The app bundles Windows executables: `ffmpeg.exe`, `ffprobe.exe`, `yt-dlp.exe`, `whisper-cli.exe`, and `pdfium.dll`. These are invoked by Go services for media processing and PDF rendering.

### Runtime Data (`var/`)

All runtime data lives under `var/` (gitignored): SQLite database (`db/s2qt.db`), config (`conf/`), temp files (`temp/`), templates (`template/`), logs (`log/`), and generated documents (`doc/`).

### Path Resolution

`util/path.go` defines `AppPaths` which resolves all runtime paths relative to the project root (found by locating `go.mod` or `wails.json`).
