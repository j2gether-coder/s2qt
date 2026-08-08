# S2QT

S2QT(Sermon to Quiet Time)는 설교 원고, 오디오 파일, 동영상 URL을 QT 묵상 자료로 변환하는 Windows 데스크톱 애플리케이션입니다. Go 기반 Wails 백엔드와 React/Vite 프론트엔드로 구성되어 있으며, AI를 이용해 QT JSON을 만들고 HTML, PDF, PNG 형식의 결과물을 생성합니다.

## 주요 기능

- 텍스트 파일 또는 직접 입력한 설교 원고 처리
- 오디오 파일에서 음성 원문 추출
- 동영상 URL에서 메타데이터 및 원문 추출
- 대상별 QT 생성 흐름 지원: 장년, 청년, 청소년, 어린이
- AI(LLM) 기반 QT JSON 생성 및 수동 결과 저장
- Step2 편집/미리보기 후 Step3 결과물 생성
- HTML, PDF, PNG 출력
- QR, 로고, 푸터, 템플릿, 보안 PIN, SMTP 등 앱 설정 관리
- SQLite 기반 작업 이력 저장 및 재작업 준비

## 기술 스택

- Backend: Go 1.25, Wails v2
- Frontend: React 19, Vite, CKEditor 5
- Storage: SQLite (`modernc.org/sqlite`)
- Runtime tools: ffmpeg, ffprobe, yt-dlp, whisper-cli, pdfium
- Installer: Inno Setup (`s2qt_setup.iss`)

## 작업 흐름

1. QT 자료 준비
   - 입력 방식 선택: 동영상 URL, 오디오 파일, 텍스트 파일/직접 입력
   - 제목, 본문 성구, 찬송, 설교자, 교회명, 설교일 등 기본 정보 입력
   - 원문을 `var/temp/temp.txt`로 저장

2. AI(LLM) 이용 및 편집
   - 대상 메뉴에서 Step1을 실행해 QT JSON 생성
   - 생성 결과는 `var/temp/temp.json`에 저장
   - Step2에서 내용을 확인하고 편집/미리보기

3. QT 문서 생성
   - Step3에서 필요한 출력 형식을 선택
   - HTML, PDF, PNG 결과물 생성
   - 결과 저장 및 이력 관리

## 디렉터리 구조

```text
.
├── app.go                    # Wails에 바인딩되는 Go App 메서드
├── main.go                   # Wails 앱 진입점
├── service/                  # QT 생성, 출력, 설정, 이력, 보안 등 핵심 서비스
├── util/                     # 앱 경로 탐색 및 런타임 디렉터리 관리
├── frontend/                 # React/Vite 프론트엔드
├── bin/                      # 배포/실행용 외부 실행 파일과 DLL
├── var/
│   ├── conf/                 # 앱 설정, 프롬프트, CSS, 보안 설정
│   ├── db/                   # SQLite DB 런타임 위치
│   ├── temp/                 # 단계별 임시 산출물
│   ├── doc/                  # 사용자 문서/가이드
│   ├── image/                # 로고, QR 이미지
│   └── template/             # 출력 템플릿
├── tools/                    # 문서 변환/검증 보조 스크립트
├── doc/                      # 설계, 정책, 작업 절차 문서
├── test/                     # 테스트 및 샘플 리소스
└── release/                  # 릴리스 관련 문서
```

## 사전 준비

- Windows
- Go 1.25 이상
- Node.js 및 npm
- Wails CLI
- OpenAI API 키

AI 생성 기능은 `OPENAI_API_KEY` 환경 변수를 사용합니다.

```powershell
$env:OPENAI_API_KEY="your-api-key"
```

프론트엔드 의존성은 `frontend/package.json` 기준으로 설치합니다.

```powershell
cd frontend
npm install
```

## 개발 실행

프로젝트 루트에서 실행합니다.

```powershell
wails dev
```

Wails 개발 모드는 Vite 개발 서버를 함께 실행하며 프론트엔드 변경 사항을 빠르게 반영합니다.

프론트엔드만 확인하려면 다음 명령을 사용할 수 있습니다.

```powershell
cd frontend
npm run dev
```

## 빌드

프로덕션 빌드는 프로젝트 루트에서 실행합니다.

```powershell
wails build
```

빌드 결과 실행 파일은 Wails 설정상 `s2qt` 이름으로 생성됩니다. 현재 저장소에는 `bin/s2qt.exe` 형태의 실행 파일과 실행에 필요한 외부 바이너리가 함께 관리됩니다.

## 테스트

Go 서비스 테스트는 다음과 같이 실행합니다.

```powershell
go test ./service/...
```

특정 테스트만 실행하려면 `-run` 옵션을 사용합니다.

```powershell
go test ./service/ -run TestName
```

## 런타임 파일

앱은 실행 시 프로젝트 루트를 기준으로 `var/` 하위 디렉터리를 사용합니다. 루트는 작업 디렉터리에서 `go.mod` 또는 `wails.json`을 찾거나, 배포 실행 파일 기준으로 `bin`의 상위 디렉터리를 사용해 결정됩니다.

주요 임시 파일은 다음과 같습니다.

- `var/temp/temp.txt`: 입력 원문
- `var/temp/temp.json`: LLM 생성 QT JSON
- `var/temp/temp.html`: 미리보기/HTML 출력

Step3 산출물 중 PDF/PNG는 프로젝트 루트의 `reports/`에 저장됩니다.

- `reports/report.pdf`: PDF 출력
- `reports/report.png`: PNG 출력

설정과 데이터 파일은 다음 위치에 저장됩니다.

- `var/conf/app.yaml`: 프롬프트 및 CSS 설정 파일 경로
- `var/conf/security.json`: 보안/PIN 관련 설정
- `var/db/s2qt.db`: SQLite 데이터베이스
- `var/log/event.log`: 이벤트 로그

## 외부 실행 파일

`bin/`에는 미디어 처리와 문서 렌더링에 필요한 Windows 실행 파일/DLL이 포함됩니다.

- `ffmpeg.exe`, `ffprobe.exe`: 오디오/비디오 변환
- `yt-dlp.exe`: 동영상 URL 처리
- `whisper-cli.exe` 및 관련 DLL: 음성 전사
- `pdfium.dll`: PDF 렌더링/PNG 변환

## 배포

Windows 설치 패키지는 Inno Setup 스크립트인 `s2qt_setup.iss`를 기준으로 구성됩니다. 무서명 릴리스 패키징 보조 스크립트로 `package_release_unsigned.bat`가 제공됩니다.

릴리스 무결성 및 배포 관련 메모는 `release/`와 `var/doc/release_integrity.md`를 참고합니다.
