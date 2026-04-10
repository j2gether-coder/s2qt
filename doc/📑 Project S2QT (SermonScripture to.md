# 📑 Project: S2QT (Sermon/Scripture to Quiet Time)
> **목표:** '말씀(S)'을 기록하고 묵상(QT)으로 전환하는 보안 중심 자동화 도구.
> Rust 보안 엔진과 Go-Wails를 결합하여 설교 전사 및 A4 최적화 템플릿 변환 수행.

---

## 1. 기술 스택 (Finalized)
* **언어:** Go (Main Control) + Rust (Core Security & Memory Safety)
* **UI:** Wails (v3) + Vite/Vue (한국어 IME 완벽 지원)
* **엔진:** yt-dlp, ffmpeg, whisper.cpp (BIN 폴더 내 상주)
* **LLM:** GPT-4o-mini / Gemini 1.5 Flash (JSON Structured Output)

---

## 2. 배포 및 실행 디렉토리 구조 (Standardized)
모든 실행 바이너리는 `BIN`에 위치하며, 데이터 및 상태 값은 `VAR`에서 격리 관리됩니다.

디렉토리 구조 전략

### A. 개발 시 (Source Structure)
* /frontend: UI 소스 (Wails/Vite)
* /backend/service: 전사 및 LLM 비즈니스 로직
* /core_security: Rust 기반 메모리 안전 보안 엔진
* /backend/db: SQLite 및 파일 시스템 관리

### B. 배포 시 (Production Layout)
* /BIN: S2QT.exe 및 종속 바이너리 상주
* /VAR: CONF, DB, IMAGE, STATE, TEMP, LOG (가변 데이터 및 보안 격리 영역)

```text
S2QT_Root/
├── BIN/                  # 실행 파일 저장소
│   ├── S2QT.exe          # 메인 프로그램 실행 파일
│   ├── yt-dlp.exe        # 유튜브 추출 도구
│   ├── ffmpeg.exe        # 오디오 변환 도구
│   ├── whisper.exe       # STT 엔진 전용 실행 파일
│   └── models/            # ggml-small.bin 등 위스퍼 모델
└── VAR/                  # 가변 데이터 저장소 (보안 및 격리)
    ├── CONF/             # 사용자 설정 및 암호화된 API KEY (Vault)
    ├── DB/               # 묵상 기록 및 작업 이력 (SQLite)
    ├── IMAGE/            # 설교 썸네일 및 스테가노그래피용 이미지
    ├── STATE/            # 작업 진행 상태 (Checkpoint)
    ├── TEMP/             # 임시 오디오 분할/작업 파일 (Wipe 대상)
    └── LOG/              # 시스템 로그


S2QT_Project/ (Root)
├── build/                 # 컴파일된 결과물 (BIN/ 폴더로 복사될 파일들)
├── frontend/              # [UI] Vite + Vue/React (Wails 프론트엔드)
│   ├── src/
│   │   ├── components/    # URL 입력바, 재생바, 설정 모달 등
│   │   ├── views/         # 메인 전사 화면, 템플릿 프리뷰 화면
│   │   └── store/         # 상태 관리 (전사 진행률 등)
├── backend/               # [SERVICE] Go 백엔드 로직
│   ├── service/           # 핵심 비즈니스 로직
│   │   ├── transcriber.go # yt-dlp, whisper 실행 제어
│   │   ├── processor.go   # LLM API 통신 및 JSON 핸들링
│   │   └── template.go    # HTML/CSS 템플릿 결합
│   ├── db/                # [DB] SQLite 및 로컬 스토리지 제어
│   └── bridge/            # [BRIDGE] Rust 보안 모듈과의 인터페이스 (CGO/FFI)
├── core_security/         # [RUST] 핵심 보안 엔진 (Rust Cargo Project)
│   ├── src/
│   │   ├── lib.rs         # Master Key 관리 및 Zeroize 로직
│   │   └── crypto.rs      # AES-GCM 암호화 구현
│   └── Cargo.toml
├── main.go                # Wails Entry Point (어플리케이션 설정 및 시작)
├── wails.json             # Wails 프로젝트 설정 파일
└── dev_assets/            # 개발 시 필요한 테스트용 오디오/이미지 샘플

## 3. 핵심 보안 및 설계 원칙
S(Sermon) Focus: 단순 음성 인식을 넘어 설교의 맥락(성경 구절, 신학적 용어)을 LLM이 정확히 파악하도록 프롬프트 최적화.

Rust Core Security: API KEY 및 민감 데이터는 Rust 모듈에서 관리. Zeroize를 통해 메모리 해제 시 물리적 덮어쓰기 수행.

Path Management: S2QT.exe가 실행될 때 상대 경로를 통해 ../VAR/ 하위 디렉토리에 접근하도록 설계.

Zero-Footprint: 작업 완료 후 VAR/TEMP 내 모든 데이터는 흔적 없이 제거(Scrubbing).

## 4. 템플릿 및 출력 가이드
Format: Markdown -> HTML(A4 CSS) -> PDF 변환.

Style: 이전 합의된 A4 최적화 스타일(Nanum Gothic, Green/Blue 포인트 박스).

Process: Whisper(Raw) -> LLM(Sermon Context JSON) -> Template Rendering.

## 5. 차기 작업 순서 (Next Steps)
Directory Initializer (Go): 실행 시 BIN 위치를 파악하고 필요한 VAR/ 구조를 자동 생성/검사하는 로직.

Master Key Vault (Rust): API KEY를 암호화하여 VAR/CONF/에 저장하고 메모리에서 안전하게 다루는 모듈.

Pipeline Integration: BIN/ 내 도구들을 순차적으로 호출하여 VAR/TEMP에서 처리하는 파이프라인 구축.
