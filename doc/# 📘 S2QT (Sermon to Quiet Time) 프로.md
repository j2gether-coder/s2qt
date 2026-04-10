# 📘 S2QT (Sermon to Quiet Time) 프로젝트 설계 정리

---

# 1. 🎯 프로젝트 개요

## ✔ 이름

**S2QT (Sermon to Quiet Time)**

## ✔ 슬로건

> 설교를 묵상으로, 묵상을 삶으로

---

## ✔ 목적

```text
설교 → 묵상(QT) → 삶의 적용
```

단순 요약이 아니라
👉 **개인 묵상(QT) 생성 도구**

---

# 2. 🔄 핵심 파이프라인

```text
YouTube URL
→ 오디오 추출 (yt-dlp + ffmpeg)
→ STT (Whisper)
→ 문장 정리 (LLM)
→ 구조 분석 (LLM)
→ QT 생성 (LLM)
→ Markdown 출력
→ (옵션) PDF 변환
```

---

# 3. 📄 QT 교재 구조 (A4 1장 기준)

```text
[제목]

[묵상(첫송)]
- 찬송가 1곡

[성구]
- 1~2절

[묵상 포인트]
- 3~4개 (각 1~2줄)

[적용 질문]
- 2~3개

[기도문]
- 4~6줄

[끝송]
- 찬송가 1곡
```

---

# 4. 🎵 찬송가 전략

* 최대 2곡
* 최소 1곡 가능

## ✔ 역할 분리

```text
첫송 → 마음 열기
끝송 → 결단/적용
```

## ✔ 필수 조건

* 추천 이유 포함
* 설교 메시지와 연결

---

# 5. 🤖 LLM 설계 구조

## ✔ 핵심 원칙

```text
LLM은 교체 가능해야 한다
```

---

## ✔ 아키텍처

```text
[App]
   ↓
[LLM Interface]
   ↓
[Provider]
   ├ OpenAI
   ├ Anthropic
   ├ Local (Ollama)
   └ 기타
```

---

## ✔ 인터페이스

```go
type LLM interface {
    Generate(req LLMRequest) (LLMResponse, error)
}
```

---

## ✔ 표준 데이터 구조

```go
type LLMRequest struct {
    Prompt string
}

type LLMResponse struct {
    Text string
}
```

---

## ✔ Factory 패턴

```go
func NewLLM(cfg Config) LLM
```

---

## ✔ config.json 예시

```json
{
  "llm": {
    "provider": "openai",
    "api_key": "xxxx",
    "model": "gpt-4"
  }
}
```

---

# 6. 🧱 디렉토리 구조

```text
/root
 ├ main.go

 ├ /app
 │   └ app.go

 ├ /core
 │   ├ pipeline.go
 │   ├ qt_generator.go
 │   └ stt.go

 ├ /service
 │   ├ youtube.go
 │   ├ whisper.go
 │   ├ llm/
 │   └ hymn.go

 ├ /util
 │   ├ path.go
 │   ├ file.go
 │   └ config.go

 ├ /bin
 │   ├ ffmpeg
 │   └ yt-dlp

 ├ /var
 │   ├ /conf
 │   │   └ config.json
 │   ├ /image
 │   ├ /db
 │   │   └ app.db
 │   └ /temp

 ├ /frontend
 │   ├ index.html
 │   └ src/

 └ /build
```

---

# 7. 🖥️ 앱 구조 (Wails 기반)

```text
Frontend (HTML/JS)
    ↓
Wails Bridge
    ↓
Backend (Go)
```

---

## ✔ 역할 분리

| 레이어      | 역할      |
| -------- | ------- |
| frontend | UI      |
| app      | 연결      |
| core     | 비즈니스 로직 |
| service  | 외부 처리   |

---

# 8. 👤 사용자 흐름

```text
1. 앱 실행
2. API Key 입력
3. YouTube URL 입력
4. QT 생성 클릭
5. 결과 확인
6. Markdown / PDF 저장
```

---

# 9. 🔐 설계 철학

```text
LLM = 사용자 책임
시스템 = 자동화 도구
```

---

## ✔ 특징

* API Key 사용자 입력
* 서버 의존 없음
* 비용 사용자 부담

---

# 10. 🎨 아이콘/브랜드 컨셉

## ✔ 컨셉

```text
책 + 빛
```

👉 의미:

* 말씀 → 깨달음 → 삶

---

## ✔ 구성

* 열린 책
* 위로 퍼지는 빛

---

## ✔ 텍스트

```text
S2QT
Sermon to Quiet Time
```

---

# 11. 🚀 확장 프로젝트 (QT 책 생성)

## ✔ 개념

```text
QT 여러 개 → 책 생성
```

---

## ✔ 구조

```text
[표지]
[목차]

Day 1
Day 2
...

[부록]
```

---

## ✔ 모듈 구조

```text
/service/book/
 ├ builder.go
 ├ template.go
 ├ exporter.go
```

---

## ✔ 데이터 구조

```json
{
  "title": "...",
  "verse": "...",
  "points": [],
  "questions": [],
  "prayer": "...",
  "hymns": []
}
```

---

# 12. 📊 발전 단계

## 1단계

```text
설교 → QT 생성
```

## 2단계

```text
QT 저장
```

## 3단계

```text
QT 묶기 → 책 생성
```

## 4단계

```text
출판 수준 PDF 생성
```

---

# 13. 🔥 최종 구조

```text
S2QT Engine
 ├ QT 생성
 ├ 데이터 저장
 └ Book 생성 (확장)
```

---

# 14. 📌 최종 한 줄 정리

> **설교를 개인 묵상으로 변환하고, 나아가 QT 콘텐츠를 생산하는 시스템**

---
