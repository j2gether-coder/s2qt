# 📑 Project: S2QT (Sermon/Scripture to Quiet Time)

> **Version:** 1.5 (Standardized Design)
> **Goal:** '말씀(S)'을 기록하고 묵상(QT)으로 전환하는 보안 중심 자동화 도구.
> Rust 보안 엔진과 Go-Wails를 결합하여 설교 전사 및 A4 최적화 템플릿 변환 수행.

---

## 1. 기술 스택 (Finalized)

| 구분 | 내용 |
|------|------|
| **언어** | Go (Main Control) + Rust (Core Security & Memory Safety) |
| **UI** | Wails (v3) + Vite/Vue (한국어 IME 완벽 지원) |
| **엔진** | yt-dlp, ffmpeg, whisper.cpp (BIN 폴더 내 상주) |
| **LLM** | GPT-4o-mini / Gemini 1.5 Flash (JSON Structured Output) |

---

## 2. 디렉토리 구조 전략

### A. 개발 환경 구조 (Source Tree)

```text
S2QT_Project/
├── frontend/               # UI 소스 (Wails/Vite/Vue)
├── backend/
│   ├── service/            # 전사 및 LLM 비즈니스 로직
│   ├── db/                 # SQLite 및 파일 시스템 제어
│   └── bridge/             # Go-Rust FFI 인터페이스
└── core_security/          # Rust 기반 Vault 엔진 (Memory Safety)
```

### B. 배포 및 실행 구조 (Production Layout)

```text
S2QT_Root/
├── BIN/                    # 실행 바이너리 및 엔진
│   ├── S2QT.exe            # 메인 프로그램
│   ├── yt-dlp.exe
│   ├── ffmpeg.exe
│   ├── whisper.exe
│   └── models/             # Whisper GGML 모델
└── VAR/                    # 가변 데이터 및 보안 격리 영역
    ├── CONF/               # 암호화된 API KEY 및 설정
    ├── DB/                 # 묵상 기록 SQLite
    ├── IMAGE/              # 썸네일 및 스테가노그래피 이미지
    ├── STATE/              # 작업 진행 상태 (Checkpoint)
    ├── TEMP/               # 임시 작업 파일 (Secure Wipe 대상)
    └── LOG/                # 시스템 로그
```

---

## 3. 핵심 설계 및 보안 원칙

- 🔐 **보안:** Rust 기반 Vault 사용, `Zeroize`를 통한 메모리 즉시 삭제, API Key 평문 노출 절대 금지.
- 📁 **파일 시스템:** 모든 데이터는 `VAR` 폴더에 격리하며, 상대 경로(`../VAR`)를 통해 접근.
- ⚙️ **실행 원칙:** Zero-Footprint 지향. 작업 종료 시 `TEMP` 폴더 내 데이터를 물리적으로 파쇄(Scrubbing).

---

## 4. 전체 데이터 파이프라인 (Pipeline)

```
1. Input        → 사용자 유튜브 URL 입력
2. Init         → Directory Initializer 실행 (VAR 구조 체크)
3. Security     → Rust Vault에서 API Key 안전 로딩 (메모리 고정)
4. Extraction   → yt-dlp & ffmpeg를 통한 16kHz Mono WAV 추출
5. Transcription→ whisper.cpp 로컬 전사 (Raw Text 생성)
6. Processing   → LLM API를 통해 정형화된 JSON 데이터 획득
7. Rendering    → Markdown 생성 → HTML 변환 → PDF 출력
8. Cleanup      → TEMP 폴더 내 잔적 제거 (Secure Delete)
```

---

## 5. 데이터 구조 (LLM JSON Output)

```json
{
  "title": "설교 제목",
  "scripture": "본문 말씀",
  "hymns": ["찬송가 번호/제목"],
  "summary": "3문장 이내 요약",
  "messages": [
    { "title": "소제목", "content": "내용" }
  ],
  "reflection": ["질문 1", "질문 2"],
  "prayer": "마침 기도문"
}
```

---

## 6. A4 단일 페이지 출력 전략 (A4 Single Page Strategy)

### 목표

모든 결과물을 **A4 1페이지** 내에 고정 배치. 텍스트 넘침 방지 및 가독성 유지.

### LLM 출력 제약 (가장 중요)

| 항목 | 제약 조건 |
|------|-----------|
| **전체 글자 수** | 1,800 ~ 2,200자 내외 |
| **요약** | 3문장 이하 |
| **메시지** | 최대 3개 (각 2~3문장) |
| **묵상 질문** | 최대 3개 |
| **기도문** | 4문장 이하 |

### CSS 최적화

```css
body {
    font-family: 'Nanum Gothic', sans-serif;
    font-size: 9.5pt;
    line-height: 1.45;
    max-height: 260mm; /* A4 가용 높이 제한 */
    overflow: hidden;
}
blockquote, h3, h4 { break-inside: avoid; } /* 박스 깨짐 방지 */
```

### 다단(Columns) Fallback 자동 제어 로직

1. **1차 렌더링:** 1단 구성으로 렌더링.
2. **높이 체크:** `document.body.scrollHeight` 측정.
3. **Fallback 1:** 높이 초과 시 2단 (`column-count: 2`) 적용.
4. **Fallback 2:** 2단 적용 후에도 초과 시 LLM에 **[내용 축소]** 재요청.

---

## 7. Zero-Footprint (Secure Delete)

작업 파일 삭제 시 단순 삭제가 아닌 **물리적 덮어쓰기** 수행.

```go
func SecureDelete(path string) error {
    data := make([]byte, 1024)
    // 3회 덮어쓰기 후 삭제
    for i := 0; i < 3; i++ {
        os.WriteFile(path, data, 0644)
    }
    return os.Remove(path)
}
```

---

## 8. 결론 및 향후 과제

- **LLM 제어**가 출력 품질의 90%를 결정함.
- **Rust 보안 계층**이 사용자 신뢰의 핵심.
- **다단 전략**은 가독성을 저해하지 않는 선에서 Fallback으로만 사용.
