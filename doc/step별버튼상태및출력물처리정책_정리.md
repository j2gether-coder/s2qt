아래에 바로 복사해서 사용할 수 있도록 `md` 전체 내용을 드립니다.

````markdown
# S2QT Step별 버튼 상태 및 출력물 처리 정책 정리

## 1. 목적

본 문서는 S2QT의 Step1~Step4 화면에서 필요한 버튼 활성화/비활성화 정책, 버튼 색상 기준, 출력물 링크 연결 방식, PDF 출력 개선 방향을 정리한 문서이다.

현재 Step3의 footer 중복 문제와 찬송가 정보 누락 문제는 해결되었으며, 다음 단계로 사용자 흐름을 안정적으로 제어하기 위한 버튼 상태 관리와 출력물 후속 서비스 연동이 필요하다.

---

## 2. 공통 정책

### 2.1 버튼 상태 구분

버튼 상태는 다음 3가지로 구분한다.

- **Primary Enabled**
  - 현재 단계에서 가장 중요한 핵심 실행 버튼
- **Secondary Enabled**
  - 보조 기능 버튼 또는 이전 버튼
- **Disabled**
  - 현재 시점에서는 실행할 수 없는 버튼

### 2.2 공통 버튼 색상 기준

#### Primary 버튼
- 배경색: `#2563eb`
- 글자색: `#ffffff`

#### Secondary 버튼
- 배경색: `#e5e7eb`
- 글자색: `#111827`

#### Disabled 버튼
- 배경색: `#cbd5e1`
- 글자색: `#ffffff`
- 커서: `not-allowed`

### 2.3 이전/다음 버튼 정책

- **다음 버튼**: Primary 스타일 적용
- **이전 버튼**: Secondary 스타일 적용
- 비활성 상태는 공통 Disabled 스타일 적용

### 2.4 공통 CSS 예시

```css
.button {
  min-width: 140px;
  height: 44px;
  border: 0;
  border-radius: 10px;
  font-weight: 700;
  cursor: pointer;
  transition: opacity 0.15s ease, background 0.15s ease, color 0.15s ease;
}

.button-primary {
  background: #2563eb;
  color: #ffffff;
}

.button-secondary {
  background: #e5e7eb;
  color: #111827;
}

.button:disabled,
.button.is-disabled,
.step-action-btn:disabled {
  background: #cbd5e1;
  color: #ffffff;
  cursor: not-allowed;
  opacity: 1;
}

.step-action-btn.primary {
  background: #2563eb;
  color: #ffffff;
}

.step-action-btn.secondary {
  background: #e5e7eb;
  color: #111827;
}
````

---

## 3. Step1(inputStep.js) 버튼 활성화/비활성화 정책

### 3.1 대상 버튼

* 실행 버튼
* 다음 버튼

### 3.2 요구사항

1. 활성화/비활성화 버튼 색 정의 필요
2. 실행 버튼 클릭 시 다음 버튼 비활성화
3. 실행 완료 시 다음 버튼 활성화

### 3.3 동작 정책

#### 실행 전

* 필수 입력값이 모두 충족되어야 실행 버튼 활성화
* 실행이 가능하지 않으면 실행 버튼 비활성화
* 다음 버튼은 기본적으로 비활성화

#### 실행 중

* 실행 버튼 비활성화
* 다음 버튼 비활성화

#### 실행 완료

* 실행 성공 시 다음 버튼 활성화
* 실행 실패 시 다음 버튼 계속 비활성화

### 3.4 상태값 예시

* `inputValid`
* `inputRunning`
* `inputReady`

### 3.5 버튼 상태 판단 기준 예시

#### 실행 버튼

```javascript
disabled = !inputValid || inputRunning
```

#### 다음 버튼

```javascript
disabled = inputRunning || !appState.status.inputReady
```

---

## 4. Step2(draftStep.js) 버튼 활성화/비활성화 정책

### 4.1 대상 버튼

* 프롬프트 생성 버튼
* 프롬프트 복사 버튼
* 다음 버튼

### 4.2 요구사항

1. 활성화/비활성화 버튼 색 정의 필요
2. 프롬프트 생성 버튼 후 프롬프트 복사 버튼 사용 가능

### 4.3 동작 정책

#### 초기 상태

* Step1이 완료되지 않았다면 프롬프트 생성 버튼 비활성화
* 프롬프트 복사 버튼 비활성화
* 다음 버튼 비활성화

#### 프롬프트 생성 중

* 프롬프트 생성 버튼 비활성화
* 프롬프트 복사 버튼 비활성화
* 다음 버튼 비활성화

#### 프롬프트 생성 완료

* 프롬프트 복사 버튼 활성화
* 초안이 정상 생성되었으면 다음 버튼 활성화

### 4.4 상태값 예시

* `draftGenerating`
* `draftReady`
* `draftHtml`

### 4.5 버튼 상태 판단 기준 예시

#### 프롬프트 생성 버튼

```javascript
disabled = draftGenerating || !appState.status.inputReady
```

#### 프롬프트 복사 버튼

```javascript
disabled = !appState.draft?.draftHtml?.trim()
```

#### 다음 버튼

```javascript
disabled = draftGenerating || !appState.status.draftReady
```

---

## 5. Step3(editorStep.js) 버튼 활성화/비활성화 정책

### 5.1 대상 버튼

* 편집결과 미리보기 버튼
* 다음 버튼

### 5.2 요구사항

1. 활성화/비활성화 버튼 색 정의 필요
2. 편집결과 미리보기 버튼 제어 필요

### 5.3 참고사항

* footer 중복 문제 해결됨
* 찬송가 정보도 Step2 결과를 기반으로 가져오도록 개선됨

### 5.4 동작 정책

#### 초기 상태

* Step2 결과가 없으면 실질적인 편집 진행 불가
* 편집결과 미리보기 버튼은 비활성화 가능

#### 편집 가능 상태

* 요약/메시지/묵상/기도 중 편집 가능한 본문이 존재하면 미리보기 버튼 활성화

#### 편집 완료 상태

* `editReady`가 참이면 다음 버튼 활성화 가능

### 5.5 버튼 상태 판단 기준 예시

#### 편집결과 미리보기 버튼

```javascript
disabled = !appState.status.editReady
```

또는 완화형 기준:

```javascript
disabled = !(
  appState.editor.summaryHtml?.trim() ||
  appState.editor.messageHtml?.trim() ||
  appState.editor.reflectionHtml?.trim() ||
  appState.editor.prayerHtml?.trim()
)
```

---

## 6. Step4(outputStep.js) 출력물 처리 정책

### 6.1 대상 버튼

* HTML 생성
* DOCX 생성
* PDF 생성

### 6.2 출력물 정보 링크 문제

현재 출력물 정보 영역에 HTML / DOCX / PDF 경로 링크가 생성되지만 실제 연결은 되어 있지 않다.

### 6.3 요구사항

1. HTML의 경우 기본 브라우저 연결
2. DOCX의 경우 Word와 연결
3. PDF의 경우 기본 브라우저 또는 기본 PDF 뷰어 연결
4. PDF는 A4 한 장으로 출력되도록 개선 필요

---

## 7. Step4 출력물 링크 연결 정책

### 7.1 기본 원칙

파일 링크는 운영체제의 기본 연결 프로그램으로 연다.

### 7.2 기대 동작

* `.html` → 기본 브라우저
* `.docx` → Microsoft Word 또는 기본 워드 프로그램
* `.pdf` → 기본 브라우저 또는 기본 PDF 뷰어

### 7.3 구현 방향

Wails의 `app.go`에 공통 파일 열기 메서드를 추가한다.

### 7.4 app.go 예시

```go
func (a *App) OpenFile(filePath string) error {
	cmd := exec.Command("cmd", "/c", "start", "", filePath)
	return cmd.Start()
}
```

### 7.5 outputStep.js 예시

```javascript
import { OpenFile } from '../../wailsjs/go/main/App';
```

```javascript
linkEl.onclick = async (event) => {
  event.preventDefault();
  const file = linkEl.dataset.file;
  if (!file) return;

  try {
    await OpenFile(file);
  } catch (error) {
    alert(`파일 열기 실패: ${String(error)}`);
  }
};
```

---

## 8. Step4 PDF A4 한 장 출력 개선 방향

### 8.1 현재 문제

PDF가 A4 한 장으로 출력되어야 하나, 현재는 두 장으로 출력되는 경우가 발생한다.

### 8.2 원인

현재 `pdf_service.go`의 HTML 래핑 구조는 PDF 인쇄용 CSS 제어가 부족하여, 화면용 레이아웃이 그대로 인쇄에 적용되고 있을 가능성이 높다.

### 8.3 개선 원칙

PDF 출력용 HTML에는 별도의 인쇄 전용 CSS를 적용해야 한다.

### 8.4 개선 항목

* `@page size: A4`
* margin 축소
* 제목 폰트 축소
* section 간격 축소
* 본문 line-height 축소
* prayer / footer padding 축소
* page-break 방지 설정

### 8.5 pdf_service.go 내 wrapHTML 스타일 예시

```html
<style>
  @page {
    size: A4;
    margin: 12mm 14mm 12mm 14mm;
  }

  html, body {
    margin: 0;
    padding: 0;
    background: #ffffff;
    color: #111827;
    font-family: Arial, sans-serif;
    font-size: 12px;
    line-height: 1.45;
  }

  body {
    -webkit-print-color-adjust: exact;
    print-color-adjust: exact;
  }

  .qt-wrap {
    width: 100%;
    max-width: 180mm;
    margin: 0 auto;
  }

  .qt-title {
    margin: 0 0 10px;
    font-size: 22px;
    line-height: 1.2;
  }

  .qt-subbox,
  .qt-box {
    margin-top: 10px;
    padding: 10px 12px;
    border-radius: 8px;
    background: #f8fafc;
  }

  .qt-section-title {
    margin: 16px 0 8px;
    font-size: 16px;
    line-height: 1.25;
  }

  p, li, div {
    orphans: 2;
    widows: 2;
  }

  .qt-footer {
    margin-top: 18px;
    page-break-inside: avoid;
  }

  .qt-footer-line {
    height: 1px;
    background: #d1d5db;
    margin-bottom: 8px;
  }

  .qt-footer-text {
    text-align: center;
    font-size: 11px;
    color: #6b7280;
  }
</style>
```

### 8.6 정책적 판단

내용이 많은 경우 무조건 1장 고정은 어려울 수 있으므로 아래 정책을 권장한다.

* 기본 목표: A4 1장 우선
* 초과 시: 자동 축소 시도
* 그래도 초과 시: 2페이지 허용

---

## 9. 단계별 다음 작업 우선순위

### 9.1 1차 작업

* 공통 버튼 활성/비활성 스타일 정의
* Step1~Step4 버튼 제어 반영

### 9.2 2차 작업

* `OpenFile` 서비스 추가
* outputStep.js 링크 실제 연결

### 9.3 3차 작업

* PDF 인쇄 전용 CSS 반영
* A4 한 장 우선 최적화

### 9.4 4차 작업

* DOCX 생성 서비스 구현
* PPT 생성 서비스 구현

---

## 10. 향후 서비스 확장 방향

향후 Step4에서는 단순 PDF 외에도 다음 서비스 확장이 가능하다.

### 10.1 DOCX 생성

* Word 문서로 저장
* 기본 워드 편집 가능 형식 제공
* 설교문 수정/배포에 적합

### 10.2 PPT 생성

* 슬라이드 자동 생성
* 설교 요약 / 제목 / 본문 / 기도문 등을 페이지 단위로 분리
* 발표용, 예배용 자료로 활용 가능

### 10.3 기타 확장

* 인쇄 최적화 템플릿
* 표지 포함 문서 생성
* 교회별 서식 템플릿 선택 기능

---

## 11. 결론

현재 시점에서 우선적으로 필요한 것은 다음 세 가지이다.

1. Step1~Step4 버튼 활성화/비활성화 정책의 일관된 적용
2. Step4 출력물 링크를 실제 파일 열기 기능으로 연결
3. PDF 인쇄 레이아웃을 A4 한 장 우선 기준으로 개선

이후 DOCX, PPT 생성 기능을 순차적으로 채워나가면 전체 S2QT 작업 흐름이 완성도 있게 구성될 수 있다.

```

원하시면 다음 답변에서 이 md를 기준으로 바로 `style.css` 수정안부터 이어가겠습니다.
```
