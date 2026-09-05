# S2QT 프롬프트 통합안 검토

작성일: 2026-08-29
대상: `var/conf/prompt_qt_json.md`(연령대별 QT JSON) + `var/conf/prompt_infographic.md`(인포그래픽 MD)
작업 범위: READ ONLY 분석 (코드 변경 없음)

---

## 0. 결론 요약

| 요구사항 | 판정 | 비고 |
|---|---|---|
| 1. 프롬프트+전사문을 **한 번만** 사용해 JSON 생성 | **가능** | 단, JSON 스키마 확장이 전제 |
| 2. 생성된 JSON을 **로직으로** QT용/인포그래픽용 분리 | **부분 가능** | 순수 로직 파생은 불가. LLM이 인포그래픽 전용 3개 섹션을 **함께 생성**해야 함 |
| 3. 작업량 | **약 3.25일** | 상세는 6장 |

**권장안**: temp.json에 최상위 `infographic` 객체를 추가하고, Step1 LLM 호출 한 번으로 QT 4개 섹션 + 인포그래픽 6개 필드를 동시에 받는다. 기존 `sections[]` 배열과 `metadata`는 **한 글자도 바꾸지 않는다.** 따라서 HTML/PDF/PNG/DOCX/PPTX/blog 렌더링 경로는 변경 0.

### 확정 사항 (2026-08-29)

| # | 결정 |
|---|---|
| 1 | 인포그래픽 섹션은 **8개 기준** (제목 / 성경본문 / 말씀의 길잡이 / 말씀을 따라 / 더하는 말씀 / **말씀의 핵심** / 오늘의 적용 / 오늘의 기도) |
| 2 | 인포그래픽은 **장년(`audience = "adult"`) 실행에서만 생성**한다. young_adult / teen / child 실행에서는 프롬프트에 `infographic` 블록을 넣지 않고, Step3도 infographic.md를 만들지 않는다 |
| 3 | **Step2 편집 UI는 인포그래픽을 다루지 않는다.** Step2는 temp.html → PDF / PNG / blog.html 계열 전용이며, 인포그래픽 문구 수정은 생성된 `infographic.md`를 외부 에디터로 직접 편집한다 |
| 4 | **재작업이면 infographic.md를 생성하지 않는다.** 생성 조건 3중 게이트(adult / `infographic` 데이터 존재 / 대상 파일이 없거나 0바이트) + 명시적 재생성 플래그 |
| 5 | infographic.md 저장 위치는 **`var/temp/` 유지** (`var/doc/`는 git 추적 대상이라 부적합) |
| 6 | 재작업 진입 시 이전 파일은 **삭제가 아니라 0바이트로 비운다** (`os.Truncate(path, 0)`) |
| 8 | `var/conf/prompt_infographic.md`는 **규칙 참고용으로 보존**하고 코드에서는 참조하지 않는다 |
| 9 | **필드 단위 검증 메시지를 포함**한다 (5-2 완화책 3) |

미결은 **7번(구버전 JSON 폴백) 하나**이며, 검토 결과는 5-4에 있습니다.

2번 결정의 부수 효과가 큽니다. **인포그래픽 때문에 늘어나는 출력 길이와 JSON 실패 위험이 장년 실행에만 국한**되고, 나머지 3개 연령대의 프롬프트·출력은 지금과 완전히 동일하게 유지됩니다. 설교 1건당 인포그래픽도 1개로 고정됩니다. (상세: 5-2)

3번 결정으로 가변 개수 편집 UI 구현이 통째로 사라져 **공수가 2일 줄었습니다.** 대신 "Step3 재실행이 편집본을 덮어쓰는" 문제가 새로 생겨 4번으로 대응합니다. (상세: 5-1)

4번은 검토 과정에서 **작업내역 재작업 경로가 `infographic`을 애초에 통과시키지 못한다**는 사실이 확인되어 자연스럽게 성립합니다. 별도 "재작업 감지 로직" 없이 데이터 유무만으로 판정됩니다. 이 과정에서 **4장의 작업내역 관련 서술에 오류가 있어 정정했습니다.** (상세: 5-1)

---

## 1. 현재 구조 확인 (중요 - 오해 소지)

### 1-1. infographic.md는 "산출물"이 아니라 "프롬프트 문서"다

가장 먼저 짚어야 할 사실입니다. 현재 `var/temp/infographic.md`는 **완성된 요약문이 아니라, 외부 LLM에 붙여 넣을 프롬프트 원문**입니다.

[service/infographic_service.go:39-54](../service/infographic_service.go#L39-L54)

```go
func (s *InfographicService) BuildInfographicMD() (string, error) {
    rawText, err := s.loadTranscript()      // var/temp/temp.txt (전사문)
    title, bibleText := s.loadMetadata()    // var/temp/temp.json 의 title, bible_text
    content := BuildInfographicPrompt(title, bibleText, rawText)  // 프롬프트 템플릿 + 전사문 치환
    os.WriteFile(s.Paths.TempInfographic, []byte(content), 0644)
}
```

파일 상단 주석에도 명시되어 있습니다.

> `LLM 호출은 하지 않고, 외부 인포그래픽 생성에 그대로 붙여 넣을 수 있는 입력 문서를 만드는 것이 목적이다.`

실제 `var/temp/infographic.md`의 첫 줄도 `# [역할]`로 시작합니다. 즉 **프롬프트 그 자체**입니다.

**따라서 현재 워크플로는 LLM을 2회 사용합니다.**

| 회차 | 시점 | 입력 | 출력 | 실행 주체 |
|---|---|---|---|---|
| 1회 | Step1 | QT 프롬프트 + 전사문 | temp.json | 사용자가 복사→외부 LLM→결과 붙여넣기 ([bindQTStep1.js:72-123](../frontend/src/components/qt/bindQTStep1.js#L72-L123)) |
| 2회 | Step3 이후 | infographic.md (프롬프트+전사문) | 작업지시서의 샘플 문서 | 사용자가 수동으로 외부 LLM 실행 |

작업지시서의 "**프롬프트+전사문을 한번만 사용해서 JSON을 생성하고 싶다**"는 요구는 정확히 이 2회차를 제거하자는 뜻으로 이해했으며, 이 검토는 그 전제로 진행했습니다.

### 1-2. 문서 간 섹션 목록 불일치

작업지시서 본문과 실제 프롬프트/샘플의 섹션 개수가 다릅니다.

| 출처 | 섹션 목록 |
|---|---|
| 작업지시서 본문 (2장) | 말씀의 길잡이 / 말씀을 따라 / 더하는 말씀 / 오늘의 적용 / 오늘의 기도 (**5개**) |
| `prompt_infographic.md` [출력 순서] | 제목 / 성경본문 / 말씀의 길잡이 / 말씀을 따라 / 더하는 말씀 / **말씀의 핵심** / 오늘의 적용 / 오늘의 기도 (**8개**) |
| 작업지시서 첨부 샘플 | 위 8개와 동일 (`## 말씀의 핵심` 포함) |

**`말씀의 핵심`이 본문 목록에서만 빠져 있습니다.** 프롬프트 파일과 기대 산출물 샘플이 일치하므로, 이 검토는 **8개 기준**으로 진행했습니다.

---

## 2. 두 프롬프트의 항목 대응 관계

핵심 질문은 "인포그래픽 8개 항목 중 몇 개가 기존 QT JSON에서 나오는가"입니다.

| 인포그래픽 항목 | QT JSON 대응 | 재사용 판정 |
|---|---|---|
| 제목 | `metadata.title` (`[QT] ` 접두 제거) | **그대로 재사용** — `stripQTTitlePrefix()` 이미 존재 |
| 성경본문 | `metadata.bible_text` | **그대로 재사용** |
| 말씀의 길잡이 | `sections[summary].blocks[0].text` | **개념 동일 / 분량 규칙 충돌** (QT 5~6문장 vs 인포 2~3문장) |
| 말씀을 따라 | — | **없음. 신규 생성 필요** |
| 더하는 말씀 | — | **없음. 신규 생성 필요** |
| 말씀의 핵심 | — | **없음. 신규 생성 필요** |
| 오늘의 적용 | `sections[reflection].blocks[0].items[]` | **개념 유사 / 문체 규칙 충돌** (아래 2-2) |
| 오늘의 기도 | `sections[prayer].blocks[0].text` | **개념 동일 / 분량 규칙 충돌** (QT 5~6문장 vs 인포 2~3문장) |

**8개 중 완전 재사용 2개, 조건부 재사용 3개, 신규 생성 3개.**

### 2-1. 신규 3개 항목은 로직으로 파생할 수 없다

작업지시서의 요구 2번("생성된 JSON에서 로직으로 분리")을 항목별로 검증했습니다.

**말씀을 따라 (3~5단계, 각 한 문장)**

`message_title` 3개를 나열하는 방식을 검토했으나 성립하지 않습니다. 실제 temp.json의 `message_title`은 다음과 같습니다.

```
"공의로 판단하시는 하나님을 신뢰하기"
"소망 가운데 인내하며 경건을 지키기"
"최종 승리와 영광의 면류관을 바라보기"
```

이는 **설교 요점의 소제목**(명사구)입니다. 반면 샘플의 "말씀을 따라"는 **본문 자체의 흐름을 진술한 완결 문장**입니다.

```
"성령께서 우리의 연약함을 도우시며, 우리가 마땅히 기도할 바를 알지 못할 때
 말할 수 없는 탄식으로 우리를 위하여 친히 간구하신다."
```

성격도 다르고 개수도 다릅니다(3고정 vs 3~5). **로직 변환 불가.**

**더하는 말씀 (예화 정리)**

QT 프롬프트는 예화를 오히려 **억제**하는 방향입니다.

[var/conf/prompt_qt_json.md:101](../var/conf/prompt_qt_json.md#L101), [:117](../var/conf/prompt_qt_json.md#L117)
```
✗ Do NOT create illustrative material not in transcript
- Illustrations: Include only if directly supporting main message; otherwise summarize as one sentence
```

즉 예화는 별도 필드로 분리되어 있지 않고 `message` 문단 산문 안에 한 문장으로 녹아 있거나 아예 빠져 있습니다. 실제 temp.json의 message 3개 문단에도 예화가 독립적으로 존재하지 않습니다. **추출 불가.**

또한 인포그래픽 프롬프트에는 QT 프롬프트에 없는 고유 규칙이 있습니다.

[var/conf/prompt_infographic.md:70](../var/conf/prompt_infographic.md#L70)
```
- 설교자 본인의 이야기는 제3자 시점으로 서술합니다.
```

이 시점 변환은 원문 전사문을 봐야만 가능합니다. JSON만으로는 불가능합니다.

**말씀의 핵심 (한 문장)**

`summary`의 첫 문장을 잘라 쓰는 방식이 이론상 가능하나, summary는 "배경 설명"이고 말씀의 핵심은 "명제형 한 문장"입니다.

```
summary 첫 문장 : "인생의 시련과 환난 속에서 좌절하지 않는 힘은 ... 비롯됩니다."
말씀의 핵심 샘플 : "내가 무너지고 연약해도 성령께서 나를 위해 기도하시고 ... 선을 이루신다."
```

문체(`~습니다` vs `~신다`)와 역할이 모두 다릅니다. 기계적 절단은 품질을 보장하지 못합니다. **LLM 생성 권장.**

### 2-2. 오늘의 적용 - 문체 규칙 충돌

`reflection`을 재활용하려 해도 두 프롬프트의 문장 형식이 정면으로 다릅니다.

| | 형식 규칙 | 실제 예 |
|---|---|---|
| QT `reflection` | 점검형: `~고 있는지 생각해봅시다/살펴봅시다/확인해봅시다` | "억울한 일이나 환난을 만날 때 사람을 원망하지 않고 공의의 하나님께 온전히 맡기고 있는지 생각해봅시다." |
| 인포 `오늘의 적용` | 실천형 평서: `~한다` | "문제가 닥쳐 기도할 바를 모를 때, 완벽한 문장을 만들지 말고 주의 이름을 부르며 성령의 탄식에 나를 맡긴다." |

거기에 audience별로 QT `reflection` 형식이 또 갈립니다.

[service/prompt_qt_json.go:388](../service/prompt_qt_json.go#L388), [:423](../service/prompt_qt_json.go#L423), [:457](../service/prompt_qt_json.go#L457), [:491](../service/prompt_qt_json.go#L491)

| audience | reflection 형식 |
|---|---|
| adult | `~고 있는지 생각해봅시다` |
| young_adult | `나는...할 것이다` / `이번 주 나는...할 수 있다` |
| teen | `나는...할 것이다` |
| child | `나는...할 수 있다` |

문자열 치환으로 `~고 있는지 생각해봅시다` → `~한다`를 만들면 어법이 깨집니다("맡기고 있는지 생각해봅시다" → "맡기고 있는지한다"). **전용 필드를 두는 편이 안전합니다.**

> 확정 사항 2(장년 한정)에 따라, 실제로 대비해야 할 충돌은 **adult의 `~고 있는지 생각해봅시다` 하나**로 줄었습니다. young_adult/teen/child의 `나는...할 것이다` 형식은 인포그래픽과 만나지 않습니다.

---

## 3. 권장 통합안

### 3-1. 방향

> **한 번의 LLM 호출로 통합 JSON을 생성하되, "분리"는 로직이 아니라 "필드 분리"로 달성한다.**

QT 4개 섹션은 지금 그대로 두고, 인포그래픽 전용 필드 묶음을 **최상위 `infographic` 객체**로 나란히 추가합니다. 산출 단계에서는 각자 자기 영역만 읽으므로 서로 간섭하지 않습니다.

**장년(adult) 실행**

```
                  ┌─ sections[]  ──→ temp.html ─→ PDF / PNG / DOCX / PPTX
전사문 ─(LLM 1회)─→ temp.json ─┤                └→ blog.html
                  └─ infographic ──→ infographic.md   ← 신규 렌더러
```

**그 외 연령대(young_adult / teen / child) — 현재와 완전히 동일**

```
전사문 ─(LLM 1회)─→ temp.json ──→ sections[] ──→ temp.html ─→ PDF / PNG / DOCX / PPTX
                                             └→ blog.html
                   (infographic 없음)          infographic.md 생성 안 함
```

### 3-2. 제안 스키마

```jsonc
{
  "version": "1.1",              // 1.0 → 1.1 (구버전 JSON 판별용)
  "doc_type": "qt",
  "audience": "adult",
  "template_id": "qt_classic",

  "metadata":  { /* 기존 그대로, 변경 없음 */ },
  "sections":  [ /* 기존 4개(summary/message/reflection/prayer) 그대로, 변경 없음 */ ],

  // audience = "adult" 실행에서만 존재한다 (omitempty)
  "infographic": {
    "guide":  "말씀의 길잡이 2~3문장",
    "follow": ["말씀을 따라 1", "…", "…"],          // 3~5개, 각 한 문장
    "extra":  ["더하는 말씀 문단1", "문단2"],        // 예화 정리, 0~3문단
    "core":   "말씀의 핵심 한 문장",
    "apply":  ["오늘의 적용 1", "…"],               // 2~3개
    "prayer": "오늘의 기도 2~3문장"
  }
}
```

**설계 근거**

- **제목/성경본문은 넣지 않는다.** `metadata.title`에서 `[QT] `만 떼면 되고([infographic_service.go:90](../service/infographic_service.go#L90)의 `stripQTTitlePrefix()`가 이미 있음), `bible_text`는 그대로 씁니다. 중복 필드를 만들면 Step2에서 편집했을 때 두 곳이 어긋납니다.
- **`sections[]` 배열에 넣지 않는다.** 배열에 5번째 섹션을 추가하면 `buildBlogSections()`([blog_service.go:176](../service/blog_service.go#L176))와 `Load()`의 switch가 `default`로 무시하긴 하지만, 순회 로직에 계속 노출됩니다. 최상위 형제 필드로 두면 기존 순회 코드가 아예 보지 않습니다.
- **`version`을 1.1로 올린다.** 구버전 JSON(작업내역 DB에 이미 쌓인 것들)과 신버전을 렌더러가 판별할 수 있게 합니다. `infographic` 유무만으로는 "구버전"과 "장년이 아니라서 없음"을 구분할 수 없으므로, 버전 필드가 반드시 필요합니다.
- **`infographic` 객체는 `audience = "adult"`일 때만 존재한다.** 다른 연령대의 temp.json에는 이 키 자체가 없습니다(`omitempty`).

### 3-2-1. 프롬프트 조립부의 audience 분기

[service/prompt_qt_json.go:526-540](../service/prompt_qt_json.go#L526-L540)의 `BuildQTPromptJSON()`은 현재 4개 블록을 무조건 이어 붙입니다. 여기에 조건 하나만 추가하면 됩니다.

```go
parts := []string{basePrompt, audienceRules, schemaPrompt}

// 인포그래픽은 장년용으로 한정한다. 다른 연령대에서는 블록 자체를 넣지 않아
// 프롬프트/출력 길이를 지금과 동일하게 유지한다.
if strings.TrimSpace(meta.Audience) == "adult" {
    parts = append(parts, infographicSchemaPrompt)
}

parts = append(parts, transcriptPrompt)
prompt := strings.Join(parts, "\n\n")
```

`meta.Audience`는 [app.go:524](../app.go#L524)에서 채워지고 빈 값은 [app.go:537](../app.go#L537)에서 이미 거부되므로, 별도 방어 코드는 필요 없습니다.

### 3-3. 반드시 처리해야 할 함정: Step2 저장 시 유실

**이 통합안에서 유일하게 위험한 지점입니다.**

`QTStep2Service.Save()`는 기존 temp.json을 읽어 수정하는 방식이 아니라, `QTSectionDoc`을 **처음부터 새로 조립해서 통째로 덮어씁니다.**

[service/qtstep2_service.go:182-240](../service/qtstep2_service.go#L182-L240)

```go
doc := QTSectionDoc{
    Version:  "1.0",
    Metadata: map[string]any{ /* 화면 입력값으로 새로 구성 */ },
    Sections: []QTSectionData{ /* 4개 섹션 새로 구성 */ },
}
// ... MarshalIndent 후 파일 덮어쓰기
```

`QTSectionDoc`에 없는 필드는 저장 순간 **소리 없이 사라집니다.** Step2 저장은 Step3 진입의 필수 절차이므로, 대책이 없으면 인포그래픽 데이터는 100% 유실됩니다.

다행히 이 프로젝트에는 **이미 같은 문제를 푼 전례**가 있습니다. `support_scriptures_full`(블로그 전용 필드)이 정확히 같은 이유로 유실되던 것을 round-trip 보존으로 해결했습니다.

[service/qtstep2_service.go:133-158](../service/qtstep2_service.go#L133-L158)

```go
// preserveInternalBlogFields는 기존 temp.json의 blog 전용 내부 필드(support_scriptures_full)를
// 새로 저장할 metadata 맵에 복사해 유실을 방지한다.
func preserveInternalBlogFields(tempJsonPath string, metadata map[string]any) { ... }
```

**동일 패턴을 그대로 따르면 됩니다.** `QTSectionDoc`에 `Infographic` 필드를 정식 선언하고, `Save()` 직전에 기존 파일에서 읽어 복사합니다. 검증된 방식이라 리스크가 낮습니다.

> 참고: `support_scriptures_full`은 프롬프트 스키마에 정의되어 있음에도([prompt_qt_json.go:140](../service/prompt_qt_json.go#L140)) **실제 `var/temp/temp.json`에는 존재하지 않습니다.** LLM이 누락한 것으로 보입니다. 필드를 늘릴수록 이런 누락이 늘어나므로, 5-2의 완화책이 필요합니다.

### 3-4. infographic.md 생성 로직 교체

`BuildInfographicMD()`가 "프롬프트 조립"에서 "**temp.json → 마크다운 렌더**"로 바뀝니다. `blog_service.go`가 `temp.json → blog.html`을 하는 것과 같은 구조이므로, 그 패턴을 그대로 복제하면 됩니다.

```
[현재]  prompt_infographic.md + temp.txt  ──조립──→ infographic.md (프롬프트)
[변경]  temp.json (metadata + infographic) ──렌더──→ infographic.md (완성 문서)
```

`var/conf/prompt_infographic.md`의 [작성 규칙]은 통합 프롬프트로 흡수되며, 파일 자체는 규칙 참고용으로 남기거나 제거합니다. `app.yaml`의 `prompt_infographic_file` 설정과 `writeDefaultInfographicPrompt()`의 자동 생성 로직도 함께 정리 대상입니다.

`makeInfographic()`([qtstep3_service.go:156](../service/qtstep3_service.go#L156))은 다음 3가지를 구분해 처리해야 합니다.

| temp.json 상태 | Step3 결과 (`QTStep3Result.Infographic`) |
|---|---|
| adult + `infographic` 있음 + 파일 없음 | `Success: true`, `Status: "완료"` — infographic.md 생성 |
| adult + `infographic` 있음 + **파일 이미 있음** | `Success: true`, `Status: "유지"` — 편집본 보호, 생성 건너뜀 (5-1) |
| `audience ≠ "adult"` | `Success: true`, `Status: "해당 없음"` — **실패가 아님** |
| **`version = "1.0"`** + `infographic` 없음 | `Success: true`, `Status: "해당 없음"` — 구버전 또는 작업내역 재작업 |
| **`version = "1.1"`** + adult + `infographic` 없음 | `Success: false`, `Status: "정보 없음"` — LLM 누락 (5-2) |

마지막 두 줄의 구분에 `version` 필드가 쓰입니다. `buildQTSectionDocFromFlat()`이 재작업 복원 시 `Version: "1.0"`을 하드코딩하므로([history_service.go:571](../service/history_service.go#L571)), **"재작업이라 없는 것"과 "LLM이 빠뜨린 것"을 코드로 구별할 수 있습니다.** 3-2에서 버전을 1.1로 올리자고 한 이유가 여기서 실효를 냅니다.

"실패가 아닌 경우"를 실패로 표시하면 장년 외 연령대와 모든 재작업 건이 매번 에러로 뜨므로, **이 구분이 중요합니다.**

`infographic` 객체가 없는 구버전 JSON에 대한 폴백 정책은 결정이 필요합니다(7장).

---

## 4. 기존 산출물 영향도

요구사항 2번의 "기존 산출물 로직에 최소의 영향"을 파일 단위로 점검했습니다.

| 산출물 | 데이터 경로 | 영향 |
|---|---|---|
| temp.html | `QTStep2Data` → `buildQTStep2HTML()` | **없음** — 평면 구조체를 그대로 사용 |
| report.pdf | temp.html → pdfium | **없음** |
| report.png | temp.html → pdfium | **없음** |
| DOCX / PPTX | 미연결 (`"준비중"`, [qtstep3_service.go:129-143](../service/qtstep3_service.go#L129-L143)) | **없음** |
| blog.html | temp.json → `buildBlogSections()` | **없음** — `sections[]`만 읽음 |
| **infographic.md** (adult) | prompt + temp.txt | **전면 교체** (의도된 변경) |
| **infographic.md** (adult 외) | prompt + temp.txt | **생성 중단** — `"해당 없음"` 상태 표시 (의도된 변경) |
| Step2 편집 UI | `QTStep2Data` 평면 필드 | **없음** — 확정 사항 3에 따라 인포그래픽을 다루지 않음 (5-1) |
| 작업내역 DB | `history_qt_json.qt_result_json` (**평면 페이로드** 저장) | **스키마 변경 불필요** — 단, `infographic`은 이 경로를 통과하지 못함. 재작업 시 인포그래픽이 복원되지 않는 것은 **의도된 동작**입니다 (5-1) |
| adult 외 연령대의 프롬프트/출력 | `BuildQTPromptJSON()` audience 분기 | **없음** — 블록 자체가 붙지 않아 현재와 동일 |

**의도적으로 바꾸는 infographic.md 외에는 영향이 0입니다.** 이는 `sections[]`와 `metadata`를 건드리지 않기로 한 3-2의 설계 덕분입니다.

---

## 5. 남는 문제와 완화책

### 5-1. 인포그래픽 편집은 md 파일에서 직접 한다 (확정)

**Step2 편집 UI는 인포그래픽을 다루지 않습니다.** Step2는 `temp.html` → PDF / PNG / blog.html 계열 산출물을 위한 편집 화면이고, 인포그래픽은 그 계열과 별개의 산출물이기 때문입니다. 인포그래픽 문구 수정은 생성된 `infographic.md`를 외부 에디터로 직접 편집합니다.

이 결정으로 다음이 모두 불필요해집니다.

| 원래 필요했던 작업 | 상태 |
|---|---|
| `QTStep2Data`에 인포그래픽 필드 추가 | **불필요** |
| 가변 개수 입력 UI (`follow` 3~5개, `apply` 2~3개, `extra` 0~3문단) | **불필요** |
| Step2 Load/Save 바인딩 확장 | **불필요** |

`QTStep2Data`는 `MessageTitle1/2/3`처럼 개수가 코드에 고정된 평면 구조체라([types.go:111-141](../service/types.go#L111-L141)) 가변 배열을 넣으려면 없던 패턴을 새로 도입해야 했는데, 그 부담이 통째로 사라졌습니다. **공수 2일 감소** (6장).

이때 `infographic` 객체는 Step2 화면에서 편집되지 않으므로, 3-3의 round-trip 보존은 **여전히 필수**입니다. 오히려 편집하지 않기 때문에 더 중요합니다 — Step2 저장 한 번으로 통째로 날아가면 복구 경로가 없습니다.

#### 반드시 처리해야 할 함정: Step3 재실행이 편집본을 덮어쓴다

**이 결정에서 새로 생기는 유일한 위험입니다.**

`makeInfographic()`은 Step3 산출물 생성 시 **아무 조건 없이 매번 실행**됩니다.

[service/qtstep3_service.go:149-151](../service/qtstep3_service.go#L149-L151)

```go
// infographic.md도 산출물 생성 시 항상 자동 생성한다(화면에는 노출하지 않는다).
s.makeInfographic(result)
```

`QTStep3Request`에는 HTML/PDF/DOCX/PPTX/PNG 플래그만 있고([types.go:149-156](../service/types.go#L149-L156)) 인포그래픽 플래그는 없습니다. 따라서 다음 시나리오에서 편집본이 소실됩니다.

```
1. Step3 실행          → infographic.md 생성
2. 사용자가 md 직접 편집  → 문구 다듬기
3. PDF만 다시 뽑으려고
   Step3 재실행         → infographic.md 무조건 덮어쓰기, 2번 작업 소실
```

PDF/PNG 재생성은 템플릿을 바꿔가며 여러 번 돌리는 것이 자연스러운 사용 패턴이므로, **실제로 발생할 시나리오입니다.**

#### 해법 — "재작업이면 만들지 않는다" (검토 결과: 타당함)

"재작업 상황을 알 수 있으면 Step3에서 인포그래픽을 아예 만들지 않으면 되지 않느냐"는 방향이 맞습니다. 다만 **"재작업"이 한 종류가 아니어서** 신호를 나눠 봐야 합니다.

| 유형 | 상황 | 감지 신호 | 처리 |
|---|---|---|---|
| **A. 최초 생성** | Step1 → 2 → 3 첫 실행 | temp.json에 `infographic` 있음 + 대상 파일 없음 | **생성** |
| **B. 세션 내 Step3 재실행** | 같은 temp.json으로 PDF/PNG만 다시 뽑기 | 대상 파일이 이미 있음 | **건너뜀** (편집본 보호) |
| **C. 작업내역 재작업** | 히스토리에서 불러와 Step2부터 다시 | **temp.json에 `infographic` 자체가 없음** (아래 참조) | **건너뜀** |

여기서 중요한 발견이 있습니다. **C는 별도 감지 로직이 필요 없습니다. 데이터가 아예 없기 때문입니다.**

#### 작업내역 재작업 경로는 `infographic`을 통과시키지 못한다 (4장 정정)

4장에서 "작업내역 DB는 JSON 전문을 저장하므로 스키마 변경 불필요"라고 적었는데, **이는 사실과 다릅니다.** 실제 경로를 다시 확인한 결과는 이렇습니다.

**저장** — temp.json 원문이 아니라, 프론트가 화면 입력값으로 만든 **평면 페이로드**가 들어갑니다.

[frontend/src/components/qt/bindQTStep2.js:194](../frontend/src/components/qt/bindQTStep2.js#L194)
```js
qtResultJson: JSON.stringify(step2Payload),   // = QTStep2Data 평면 구조
```

**복원** — 평면 구조체로 파싱한 뒤 temp.json을 **새로 조립**합니다.

[service/history_service.go:559-629](../service/history_service.go#L559-L629)
```go
func buildQTSectionDocFromFlat(flat *flatQTStep2Data, ...) *QTSectionDoc {
    return &QTSectionDoc{
        Metadata: map[string]any{ /* 평면 필드 9개만 */ },
        Sections: []QTSectionData{ /* 4개 섹션만 */ },
    }
}
```

`flatQTStep2Data`([history_service.go:68-92](../service/history_service.go#L68-L92))에 없는 것은 저장 시점에 이미 사라지고, 복원 시점에 다시 만들어지지도 않습니다. **`infographic`은 이 경로를 통과할 수 없습니다.**

> 같은 이유로 `support_scriptures_full`도 재작업에서 소실됩니다. 재작업으로 만든 blog.html은 관련 성구 전체 본문 없이 참조만 표시됩니다([blog_service.go:97-103](../service/blog_service.go#L97-L103)의 폴백 경로). 이번 통합안과 무관한 **기존 동작**이지만, 같은 함정이 이미 한 번 밟혀 있다는 증거입니다.

이 사실은 오히려 **결정을 단순하게 만듭니다.**

- 히스토리에 `infographic`을 실어 나르려면 `flatQTStep2Data` + `buildQTSectionDocFromFlat` + 프론트 `buildHistoryPayload`를 모두 확장해야 합니다. 그런데 **확정 사항 3에 따라 Step2는 인포그래픽을 다루지 않으므로**, Step2 페이로드에 인포그래픽을 실어 보내는 것 자체가 설계에 어긋납니다.
- 따라서 **히스토리 확장은 하지 않습니다.** 재작업하면 `infographic`이 없고, 없으면 안 만듭니다. 요구사항과 데이터 구조가 저절로 일치합니다.

#### 최종 생성 조건 — 3중 게이트

`makeInfographic()`은 아래 3가지가 모두 참일 때만 실행합니다.

```go
// 1) 장년 전용                      (확정 2)
// 2) 데이터가 있을 때만               → 작업내역 재작업이면 자동으로 걸러짐 (유형 C)
// 3) 대상 파일이 없거나 "비어 있을" 때만  → 세션 내 재실행에서 편집본 보호 (유형 B)
if audience == "adult" && doc.Infographic != nil && isEmptyOrMissing(paths.Infographic) {
    generate()
}
```

세 조건 모두 **파일과 데이터만 보고 판단**합니다. 프론트 세션 상태(`appState`)나 별도 "재작업 플래그"에 의존하지 않으므로, 앱을 재시작하거나 화면을 새로 그려도 판단이 흔들리지 않습니다.

> **주의 — 게이트 3은 `fileExists()`를 쓰면 안 됩니다.** 확정 6이 "파일 삭제"가 아니라 **"0바이트로 비우기"** 이기 때문입니다. 기존 `fileExists()`([util_service.go:1150](../service/util_service.go#L1150))는 `os.Stat` 성공 + 디렉터리 아님만 보므로 **0바이트 파일에도 `true`를 반환**합니다. 그대로 쓰면 한 번 비운 뒤에는 영원히 생성되지 않습니다.
>
> ```go
> func isEmptyOrMissing(path string) bool {
>     info, err := os.Stat(path)
>     if err != nil {
>         return true          // 없음 → 생성 가능
>     }
>     return !info.IsDir() && info.Size() == 0   // 비어 있음 → 생성 가능
> }
> ```
>
> 이렇게 하면 "0바이트 = 비워진 슬롯 = 다시 채워도 되는 상태"라는 의미가 코드에 그대로 드러납니다.

**보조 처리 2가지**

1. **새 작업 시작 시 이전 내용 비우기** — 3번 게이트가 성립하려면 "새 설교 = 빈 파일"이 보장돼야 합니다. `PipelineService.cleanupSourcePrepareTempFiles()`([pipeline_service.go:36](../service/pipeline_service.go#L36))에 인포그래픽 파일 비우기를 추가합니다. 현재 이 목록에는 `TempTxt` / `TempAudioSrc` / `TempWav`만 있고, 방식도 `os.Remove`입니다. 인포그래픽만 **비우기(0바이트화)** 로 처리합니다 — 확정 6과 동일한 방식입니다.

   Windows 환경이므로 `cp /dev/null`에 해당하는 Go 코드는 다음과 같습니다.

   ```go
   // 파일을 지우지 않고 내용만 비운다(경로/핸들 유지).
   if err := os.Truncate(path, 0); err != nil && !os.IsNotExist(err) {
       LogError("infographic: 파일 비우기 실패: " + err.Error())
   }
   ```

2. **명시적 재생성 수단** — 3번 게이트 때문에 "내용을 바꿨으니 다시 만들고 싶다"가 막힙니다. `QTStep3Request.MakeInfographic` 플래그를 두어 **켜면 게이트 3을 무시하고 덮어쓰도록** 합니다. 기본값은 **꺼짐**입니다. Step3 화면에는 PDF/PNG 체크박스가 이미 있으므로([bindQTStep3.js:106-113](../frontend/src/components/qt/bindQTStep3.js#L106-L113)) 하나 더 추가하는 형태가 자연스럽습니다.

#### 재작업 진입 시 이전 내용 비우기 (확정 6)

작업내역 재작업은 Step1을 거치지 않고 **Step2로 바로 진입합니다**([historyWorkspace.js:543](../frontend/src/components/history/historyWorkspace.js#L543)). 따라서 위 보조 처리 1의 정리 로직이 돌지 않아, 조치가 없으면 **이전 설교의 infographic.md가 그대로 남습니다.**

생성 자체는 게이트 2가 막아 주므로 데이터 오염은 없지만, 사용자가 폴더에서 옛 파일을 보고 이번 작업의 산출물로 오인할 수 있습니다.

**확정: `PrepareReworkFromHistory()`에서 `infographic.md`를 0바이트로 비웁니다.** 파일을 삭제하지 않고 내용만 비우므로 경로와 열려 있는 편집기 핸들이 유지되고, 사용자가 "이 작업에는 인포그래픽이 없다"를 파일 크기로 바로 확인할 수 있습니다.

```go
// PrepareReworkFromHistory 내부, temp.json 복원 직후
// 이전 설교의 인포그래픽이 남아 오인되지 않도록 내용만 비운다.
_ = os.Truncate(paths.TempInfographic, 0)
```

여기에 Step3 결과 `Status: "해당 없음"` 표시를 함께 두어, 화면에서도 의도된 상태임이 드러나게 합니다.

#### 저장 위치 — `var/temp/` 유지 (확정 5)

앞선 개정에서 `var/doc/`로 옮기자고 했으나, **git 추적 상태를 확인한 결과 현 위치 유지가 맞습니다. 이전 권고를 철회합니다.**

`.gitignore`는 `var/` 전체가 아니라 하위 폴더를 골라서 제외합니다.

```
/var/model/   /var/temp/   /var/state/   /var/template/   /var/log/   /var/db/   /reports/
```

즉 **`var/doc/`는 git 추적 대상**입니다(`var/doc/user_guide.md`, `license.md` 등이 실제로 커밋되어 있습니다). 설교마다 새로 생성·편집되는 파일을 여기 두면 매 작업이 git 변경으로 잡혀 저장소가 오염됩니다.

| 경로 | git | 적합성 |
|---|---|---|
| `var/temp/infographic.md` (현행) | ignored | **적합** — 설교별 산출물 |
| `var/doc/` | **tracked** | 부적합 — 매 작업이 git diff로 잡힘 |
| `reports/` (PDF·PNG) | ignored | 참고: 산출물은 ignored 위치에 두는 것이 이 프로젝트의 기존 방식 |

경로 변경 작업이 사라지므로 6장 4번 항목에서 `util/path.go` 수정이 빠집니다.

#### 편의 기능 (선택)

md를 직접 편집하려면 파일 위치를 찾아가야 합니다. Step3 화면에서 파일을 바로 여는 버튼이 있으면 편하지만, **현재 `App`에 파일 열기 메서드가 없습니다.** ([doc/step별버튼상태및출력물처리정책_정리.md:299](step별버튼상태및출력물처리정책_정리.md#L299)에 `OpenFile` 설계안만 존재하고 미구현)

필수는 아니므로 이번 범위에서는 제외하고, 필요 시 별건으로 추가하시면 됩니다.

### 5-2. 출력 길이 증가로 인한 JSON 실패 위험

현재 Step1은 **사용자가 프롬프트를 복사해 외부 LLM 웹 UI에 붙여 넣고, 결과를 다시 복사해 오는 수동 방식**입니다([bindQTStep1.js:72-123](../frontend/src/components/qt/bindQTStep1.js#L72-L123)). 출력이 길어질수록 응답 잘림과 JSON 파싱 실패가 늘어납니다.

인포그래픽 6개 필드가 추가되면 출력이 늘어납니다. 다만 확정 사항 2에 따라 **이 증가는 장년 실행에만 발생**하고, 나머지 3개 연령대는 지금과 동일합니다.

#### 실측 결과 — 추정치 하향 (2026-08-29)

이 항목은 **추정이 아니라 실측으로 대체되었습니다.** LLM 3종에 통합 프롬프트를 실제로 돌린 결과입니다([260829_S2QT_장년용_통합프롬프트_테스트.md](260829_S2QT_장년용_통합프롬프트_테스트.md) 8-5).

| 모델 | 등급 | 전체 | infographic | 증가율 |
|---|---|---|---|---|
| Grok | 무료 | 4,789자 | 946자 | **1.25배** |
| Gemini | 무료 | 4,729자 | 686자 | **1.17배** |
| Copilot | 무료 | 2,792자 | 597자 | **1.27배** |
| ChatGPT | 유료 | 3,905자 | 1,115자 | **1.40배** |
| Claude | 유료 | 2,526자 | 605자 | **1.31배** |

**당초 추정 1.4~1.6배는 과대평가였습니다. 실측 범위는 1.17~1.40배(중앙값 1.27배)입니다.** 무료 3종 + 유료 2종 **5개 모델 모두 응답 잘림이 없었고**, `infographic` 6개 필드도 전부 채워졌습니다. 이 절의 위험도는 처음 평가보다 **낮게** 보아야 합니다.

#### 위험이 실재한다는 증거

`support_scriptures_full`은 프롬프트 스키마에 명시되어 있는데([prompt_qt_json.go:140](../service/prompt_qt_json.go#L140)) **실제 `var/temp/temp.json`에는 존재하지 않습니다.** JSON은 완전히 유효했고 필드만 빠졌으며, 아무 에러도 발생하지 않았습니다.

테스트 5종 중 4종은 이 필드를 생성했고(3·3·2·2개), **ChatGPT만 빈 배열 `[]`** 이었습니다. 따라서 **상시 누락은 아니고 간헐적**입니다. 간헐적이기 때문에 오히려 코드로 잡아야 하며, 완화책 3(필드 단위 검증)의 근거는 그대로 유효합니다.

#### 실측에서 새로 확인된 문제 — QT 문장 복사

테스트에서 **Copilot이 `message` 문단을 어미만 바꿔 `follow`로 옮기는** 패턴이 나왔습니다(최장 공통 문자열 21·25·15·22자, 4항목 중 4항목). Grok·Gemini는 정상이었습니다.

```
message : "두려움은 현실을 왜곡시키고 미래를 불안하게 만듭니다."
follow  : "두려움은 현실을 왜곡시키고 미래를 불안하게 만든다."
```

복사가 일어나면 인포그래픽이 QT의 문체 변환본에 그치므로 통합의 이득이 줄어듭니다.

**다만 유료 2종 추가 테스트에서 이 문제는 모델 한계로 결론 났습니다.** ChatGPT는 최대 16자, Claude는 `follow` 4항목 전부 **공통 문자열 0자**로 완전히 독립 작성했습니다. 프롬프트 결함이 아니므로 **수정은 필수가 아닙니다.** 무료 등급 운영을 염두에 둔다면 ❌ 예시 한 줄을 보험으로 넣는 정도면 충분합니다.

#### 실측에서 새로 확인된 문제 — 메타데이터 날조

통합과 무관한 **기존 프롬프트의 빈틈**이며, 테스트에서 가장 심각하게 드러난 문제입니다.

| 모델 | 등급 | 처리 |
|---|---|---|
| Grok | 무료 | 빈 문자열 유지 — **정상** |
| Gemini | 무료 | `preacher`/`church_name` 날조 |
| Copilot | 무료 | `"홍길동 목사"`, `https://example.com/sermon` 날조. `month_accent: "말씀과 함께"` 타입 위반 |
| ChatGPT | 유료 | 전부 `"-"` — **정상** |
| **Claude** | **유료** | `"김신일 목사"`, `"하늘가족교회"`, `"2023-05-14"`, `https://youtu.be/example` 날조. `month_accent: "오"` 타입 위반 |

**등급이 해결해 주지 않습니다.** 유료 Claude가 무료 Copilot과 동일하게 날조했습니다. 5종 중 3종 발생이며 등급과 상관관계가 없습니다.

원인은 규칙 부재입니다. 현행 프롬프트는 `hymn`에만 처리 규칙이 있고, `preacher` / `church_name` / `sermon_date` / `source_url` / `month_*`에는 **아무 규칙이 없습니다.**

**실제 피해 경로도 확인했습니다.** `buildStep2Payload()`는 기본정보가 비어 있으면 temp.json 값을 그대로 채택합니다.

[frontend/src/components/qt/bindQTStep2.js:151-154](../frontend/src/components/qt/bindQTStep2.js#L151-L154)
```js
preacher:   (basicInfo.preacher   || loadedMeta.preacher   || '').trim(),
churchName: (basicInfo.churchName || loadedMeta.churchName || '').trim(),
```

사용자가 기본정보를 입력하지 않았다면 **LLM이 지어낸 설교자·교회명이 PDF·PNG·blog에 그대로 실립니다.** 교회 주보로 배포되는 문서이므로 통합 착수와 함께 **반드시** 처리해야 합니다. 상세 규칙안은 [테스트 문서 8-6](260829_S2QT_장년용_통합프롬프트_테스트.md)에 있습니다.

여기서 두 가지 실패를 구분해야 합니다.

| 실패 유형 | 탐지 | 위험도 |
|---|---|---|
| **JSON 자체가 깨짐** (잘림, 콤마 누락) | `SaveManualLLMResult`가 파싱 실패로 즉시 거부 ([app.go:570](../app.go#L570)) | 낮음 — 시끄럽게 실패하고, 사용자는 LLM 창을 열어 둔 상태라 재시도가 쌈 |
| **JSON은 유효한데 필드가 누락** | **현재 탐지 수단 없음** | **높음** — 조용히 저열한 산출물이 나감 |

**설계로 막아야 할 대상은 두 번째입니다.** 첫 번째는 이미 잡히고 있습니다.

#### 통합안 vs 현행 방식의 실패 비교

| | 현행 (LLM 2회) | 통합 후 (LLM 1회) |
|---|---|---|
| 필요한 수동 실행 | **최소 2회** 고정 (+ QT 실패 시 재시도) | **1회** (+ 실패 시 재시도) |
| QT JSON 실패 시 | QT만 재시도, 인포그래픽 영향 없음 | QT + 인포그래픽 동시 차단 → 재시도 |
| 인포그래픽 생성 실패 | 사실상 없음 (마크다운은 파싱 대상이 아님) | 필드 누락 가능 → 완화책 2로 QT는 분리 |
| 최악의 결과 | 인포그래픽용 LLM을 매 건 한 번 더 실행 | 이번 건만 인포그래픽 없이 진행 |

실패율이 올라가도 **평균 실행 횟수는 줄어듭니다.** 예시(추정치, 실측 아님)로 현행 QT 실패율 10%, 통합 후 25%를 가정하면 — 현행 평균 2.11회, 통합 후 평균 1.33회입니다. 실패율이 2.5배가 되어도 통합안이 유리한 이유는, 재시도는 확률적으로만 발생하는 반면 "2회 실행"은 매 건 고정 비용이기 때문입니다.

#### 완화책

1. **`infographic` 블록을 JSON의 맨 뒤에 배치** — 응답이 잘려도 QT 본체는 온전하고, 잘린 JSON은 파싱 실패로 잡힙니다.
2. **누락은 에러가 아닌 경고로 처리** — `SaveManualLLMResult`([app.go:550](../app.go#L550))는 `infographic`이 없어도 저장을 허용하고, Step3의 infographic만 `"정보 없음"`으로 표시합니다. QT 산출물(HTML/PDF/PNG/blog)은 정상 진행됩니다. 실패가 전체를 막지 않습니다.
3. **필드 단위 검증 메시지** — 현재의 `"유효한 JSON 형식이 아닙니다"` 대신, 무엇이 어긋났는지 짚어 줍니다.

   ```
   ❌ 유효한 JSON 형식이 아닙니다
   ✓ infographic.follow 가 2개입니다 (3~5개 필요)
   ✓ infographic.core 가 비어 있습니다
   ```

   그러면 사용자가 LLM에 "follow만 다시 써 줘"라고 요청할 수 있습니다. 전체 재생성보다 훨씬 저렴한 복구 경로입니다.
4. **프롬프트 품질 체크리스트**에 필드 존재·개수 조건을 명시합니다 (8장 초안의 QUALITY CHECKS 참고).
5. `CleanLLMJSONOutput()`의 기존 콤마 복구 로직([llm_service.go:84](../service/llm_service.go#L84))은 그대로 유효합니다.

#### 장기 대안 — API 경로 전환

[service/llm_service.go:133](../service/llm_service.go#L133)에 이미 API 직접 호출 경로(`GenerateQTJSON`)가 구현되어 있습니다(`OPENAI_API_KEY` 필요). 이 경로로 전환하면 **Structured Outputs(JSON 스키마 강제)** 를 적용할 수 있어 스키마 위반 자체가 구조적으로 불가능해지고, 위 완화책 대부분이 불필요해집니다.

이 관점에서 보면 통합안은 **API 전환과 방향이 맞습니다.** 필드가 하나의 스키마로 모여 있으므로 스키마 정의를 한 벌만 관리하면 됩니다. 반대로 프롬프트를 두 갈래로 유지하면 API 전환 시 작업이 두 배가 됩니다.

### 5-3. 문체 규칙 충돌을 프롬프트가 감당해야 한다

한 번의 호출로 **같은 내용을 두 문체로** 써야 합니다.

| | QT (adult) | 인포그래픽 |
|---|---|---|
| 길잡이 | 5~6문장, `~습니다` | 2~3문장, `~습니다` |
| 적용 | `~고 있는지 생각해봅시다` | `~한다` |
| 기도 | 5~6문장 | 2~3문장 |

LLM이 두 규칙을 섞을 위험이 실재합니다. 프롬프트에 **대조 예시(❌/✓)를 나란히** 배치해야 하며, 이는 기존 프롬프트가 이미 쓰고 있는 방식이라 스타일 일관성은 유지됩니다.

대비 대상이 adult 한 가지로 고정되었으므로, 대조 예시도 한 쌍만 쓰면 됩니다(8장 초안 참고).

### 5-4. 구버전 JSON 폴백 검토 (미결 7)

**결론: 폴백(b)은 기술적으로 성립하지 않습니다. (a) 미생성 + 안내 문구를 권장합니다.**

#### 대상 정의

"구버전 JSON" = `version: "1.0"`이면서 `infographic`이 없는 temp.json. 발생 경로는 둘입니다.

| 경로 | 빈도 |
|---|---|
| 작업내역 재작업 복원 (`buildQTSectionDocFromFlat`가 `Version: "1.0"` 하드코딩, [history_service.go:571](../service/history_service.go#L571)) | **상시** — 재작업할 때마다 |
| 통합 배포 이전에 만들어져 남아 있는 temp.json | 1회성 — 전환 직후 한동안 |

폴백안 (b)의 정의: 예전처럼 `BuildInfographicPrompt(title, bibleText, rawText)`로 **프롬프트 문서**를 만들어 주고, 사용자가 외부 LLM에 돌려 인포그래픽을 얻게 한다.

#### 근거 1 — 전사문이 없습니다 (결정적)

기존 프롬프트 방식은 `loadTranscript()`가 `var/temp/temp.txt`를 읽어야 성립합니다([infographic_service.go:57-69](../service/infographic_service.go#L57-L69)). 그런데 **작업내역에는 전사문이 저장되지 않습니다.**

- `SaveHistoryRequest`([history_service.go:43-52](../service/history_service.go#L43-L52)) — 전사문 필드 없음
- `flatQTStep2Data`([history_service.go:68-92](../service/history_service.go#L68-L92)) — 전사문 필드 없음
- `history_master` / `history_qt_json` 어느 테이블에도 없음

재작업은 Step1을 건너뛰므로 temp.txt가 복원되지도 않습니다. 결과적으로 폴백을 켜도 다음 에러로 끝납니다.

```
temp.txt가 비어 있습니다. Step1을 먼저 실행해 주세요
```

**동작하지 않는 폴백을 위해 코드를 유지하는 셈입니다.**

#### 근거 2 — 우연히 동작하면 오히려 위험합니다

temp.txt가 지워지는 시점은 Step1 시작 시(`cleanupSourcePrepareTempFiles`)뿐입니다. 따라서 이런 순서가 가능합니다.

```
1. A설교로 Step1~3 진행       → temp.txt = A설교 전사문
2. 앱을 끄지 않고 작업내역 이동
3. B설교를 재작업으로 불러옴    → temp.json만 B로 교체, temp.txt는 A 그대로
4. Step3 실행 → 폴백 동작      → B설교 제목·본문 + A설교 전사문 프롬프트
```

**조용히 틀린 문서가 만들어집니다.** 5-2에서 정리한 두 실패 유형 중 위험도가 높은 쪽(유효하지만 내용이 틀림)에 정확히 해당합니다. 에러가 나서 아무것도 안 만드는 것보다 나쁩니다.

#### 근거 3 — 확정 8과 충돌합니다

(b)를 택하면 `prompt_infographic.md` 로드 → 치환 → 파일 생성 경로를 계속 살려 둬야 합니다. 확정 8(참고용으로만 보존, **코드에서 참조하지 않음**)과 정면으로 어긋납니다.

#### 권장 처리

확정 4의 게이트 2와 **완전히 같은 처리**이므로 추가 코드가 없습니다. `version`으로 판정만 갈라 메시지를 다르게 줍니다.

```
version 1.0 + infographic 없음
  → Status: "해당 없음"  (Success: true)
     "이 작업에는 인포그래픽 데이터가 없습니다.
      인포그래픽이 필요하면 Step1부터 다시 실행해 주세요."

version 1.1 + adult + infographic 없음
  → Status: "정보 없음"  (Success: false)   ← LLM 누락, 5-2 완화책 3이 상세 사유 표시
```

#### 참고 — 재작업에서도 인포그래픽을 살리려면

전사문을 작업내역에 함께 저장해야 합니다(`history_qt_json`에 컬럼 추가 또는 별도 테이블). 그러면 재작업 시 LLM 1회 실행으로 인포그래픽만 다시 만들 수 있습니다.

다만 **확정 4(재작업 시 미생성)와 배치**되고, DB 스키마 변경 + 설교 1건당 수만 자 전사문 저장이 추가됩니다. 이번 범위에서는 제외하고, 재작업에서 인포그래픽 요구가 실제로 생기면 그때 별건으로 검토하는 편이 낫습니다.

---

### 5-5. 용어 금지 규칙 상속

인포그래픽 프롬프트에는 QT 프롬프트에 없는 규칙이 있습니다.

[var/conf/prompt_infographic.md:29](../var/conf/prompt_infographic.md#L29)
```
출력 문서에는 "설교", "설교 흐름", "설교 예화" 등의 표현을 사용하지 않습니다.
```

이 규칙은 `infographic` 블록에**만** 적용되어야 합니다. QT 본문에서는 "설교"라는 단어가 문제되지 않습니다. 프롬프트에서 적용 범위를 명시적으로 한정해야 합니다. (배경: [doc/260722_S2QT Infographic Style Guide.md](260722_S2QT%20Infographic%20Style%20Guide.md)의 "말씀 중심" 원칙)

---

## 6. 작업량 산정

확정 사항 3(Step2 편집 UI 제외)으로 단계 구분 없이 한 벌로 끝납니다.

| # | 작업 | 파일 | 난이도 | 공수 |
|---|---|---|---|---|
| 1 | 통합 프롬프트 작성 (인포그래픽 스키마·규칙·예시 추가, 문체 대조 예시, 용어 금지 범위 한정) + **adult 전용 조립 분기** | `service/prompt_qt_json.go`, `var/conf/prompt_qt_json.md` | 중 | **0.75일** |
| 2 | `InfographicData` 타입 정의 | `service/types.go` | 하 | **0.25일** |
| 3 | Step2 저장 시 round-trip 보존 (`preserveInternalBlogFields` 패턴 복제) | `service/qtstep2_service.go` | 중 | **0.5일** |
| 4 | infographic.md 렌더러 재작성 (JSON → MD) + **Step3 상태 구분**(완료 / 유지 / 해당 없음 / 정보 없음) + `prompt_infographic.md` 참조 코드 제거 (확정 8) | `service/infographic_service.go`, `service/qtstep3_service.go`, `service/prompt_infographic.go` | 중 | **0.5일** |
| 5 | **필드 단위 검증 메시지**(5-2 완화책 3) + 누락 시 경고 처리 + 테스트 (보존 round-trip, 렌더러, audience 분기, 폴백) | `app.go`, `service/*_test.go` | 중 | **0.75일** |
| 6 | **생성 3중 게이트**(adult / 데이터 존재 / **파일이 없거나 0바이트**) + `isEmptyOrMissing()` 헬퍼 + `MakeInfographic` 강제 재생성 플래그 + Step3 체크박스 + 새 작업·재작업 진입 시 **0바이트화**(`os.Truncate`) | `service/types.go`, `service/qtstep3_service.go`, `service/pipeline_service.go`, `service/history_service.go`, `frontend/src/components/qt/qtStep3.js`, `bindQTStep3.js` | 중 | **0.5일** |
| | **합계** | | | **3.25일** |

프롬프트 원문은 Go 상수(`defaultQTPromptBase`, `defaultQTPromptJSONSchema`)와 `var/conf/*.md` **두 곳에 중복 존재**하며([prompt_qt_json.go:268-290](../service/prompt_qt_json.go#L268-L290)의 파일 우선 → 상수 폴백 구조), 양쪽을 반드시 함께 갱신해야 합니다. 1번 공수에 반영되어 있습니다.

### 확정 사항에 따른 공수 변동 이력

| 시점 | 공수 | 변동 사유 |
|---|---|---|
| 최초 검토 | 2.5일 (+ 편집 UI 2일) | — |
| 확정 2 (장년 한정) | 2.75일 (+ 편집 UI 2일) | 필드 단위 검증 메시지 추가 (+0.25) |
| 확정 3 (편집 UI 제외) | 3일 | 편집 UI 2일 제거, 덮어쓰기 방지 +0.25 |
| 확정 4 (재작업 시 미생성) | 3.25일 | 3중 게이트 + 파일 정리 로직 +0.25 |
| 확정 5·6·8·9 | **3.25일** | 경로 이동 제거(−) ↔ 0바이트 처리·헬퍼 추가(+) 상쇄 |

**참고 — 기존 방식 유지 시 공수는 0**입니다. 통합의 실질 이득은 "장년 QT 1건당 LLM 수동 실행 2회 → 1회"이므로, 절약되는 사용자 조작 시간과 위 공수를 비교해 판단하시면 됩니다. 확정 사항 2로 인포그래픽이 장년으로 한정되면서, 이득도 위험도 모두 장년 실행에만 발생합니다.

---

## 7. 결정이 필요한 사항

### 확정됨 (2026-08-29)

| # | 항목 | 결정 |
|---|---|---|
| 1 | 인포그래픽 섹션 개수 | **8개** (`말씀의 핵심` 포함). 작업지시서 본문의 5개 목록은 누락이며, 프롬프트 파일과 첨부 샘플 기준이 맞음 |
| 2 | 인포그래픽 적용 범위 | **장년(`audience = "adult"`) 실행에서만 생성.** 다른 연령대는 프롬프트에 블록을 넣지 않고 infographic.md도 만들지 않음 |
| 3 | 인포그래픽 편집 방식 | **Step2 편집 UI에서 다루지 않음.** 생성된 `infographic.md`를 외부 에디터로 직접 편집 |
| 4 | 재작업 시 처리 | **재작업이면 infographic.md를 생성하지 않는다.** 생성 조건을 3중 게이트(adult / `infographic` 데이터 존재 / 대상 파일이 없거나 0바이트)로 두고, 명시적 재생성만 플래그로 허용 |
| 5 | infographic.md 저장 위치 | **`var/temp/` 유지.** `var/doc/`는 git 추적 대상이라 설교별 산출물을 두기에 부적합 (5-1) |
| 6 | 재작업 진입 시 이전 파일 처리 | **삭제가 아니라 0바이트로 비우기**(`os.Truncate(path, 0)`). 경로와 편집기 핸들이 유지됨 (5-1) |
| 8 | `var/conf/prompt_infographic.md` | **규칙 참고용으로 보존.** 코드에서는 참조하지 않음. 이 파일은 git 추적 중이므로 현 위치 그대로 보존됨 |
| 9 | 필드 단위 검증 메시지 | **포함** (5-2 완화책 3) |

### 미결

| # | 항목 | 선택지 | 검토자 의견 |
|---|---|---|---|
**없음.** 아래 7번 검토 결과를 승인하시면 착수 가능합니다.

| # | 항목 | 검토 결과 | 상태 |
|---|---|---|---|
| 7 | 구버전 JSON 폴백 | **(a) 미생성 + 안내 문구.** 폴백(b)은 전사문이 작업내역에 저장되지 않아 **동작 자체가 불가능**하고, 우연히 동작하면 다른 설교의 전사문을 붙이는 오염 위험이 있습니다 (5-4) | 승인 대기 |

---

## 8. 참고 - 통합 후 프롬프트 추가분 초안

기존 `defaultQTPromptJSONSchema` 뒤에, **`audience = "adult"`일 때만** 붙일 블록의 개략입니다. (실제 작성은 6장 작업 1번, 조립 분기는 3-2-1)

```
[Infographic Output Contract]

sections[] 와 별개로, 최상위에 "infographic" 객체를 하나 추가한다.
같은 전사문에서 뽑되, 아래 규칙은 QT 본문 규칙과 독립적으로 적용한다.

용어 고정: 말씀의 길잡이 / 말씀을 따라 / 더하는 말씀 / 말씀의 핵심 / 오늘의 적용 / 오늘의 기도
금지어(이 블록에 한함): "설교", "설교 흐름", "설교 예화"

- guide  : 본문의 배경과 상황 2~3문장. 왜 이 말씀이 필요한지 설명한다.
           (QT summary는 5~6문장이지만 여기서는 반드시 2~3문장으로 줄인다)
- follow : 3~5개. 각 항목은 완결된 한 문장. 말씀의 흐름이 순서대로 이어지게 한다.
           ❌ "생각은 말씀으로 지키기"          (명사구 - message_title 형식)
           ✓ "성령께서 우리의 연약함을 도우시며 친히 간구하신다."
- extra  : 전사문에 실제로 등장한 예화만 정리한다. 없으면 빈 배열 [].
           예화가 본문 메시지를 어떻게 뒷받침하는지 함께 서술한다.
           설교자 본인의 이야기는 제3자 시점("한 목회자는...")으로 바꾼다.
- core   : 한 문장. 명제형으로 단정한다.
           ✓ "하나님의 평강은 두려움 가운데서도 우리를 붙드신다."
- apply  : 2~3개. 실천형 평서문(~한다)으로 쓴다.
           ❌ "...하고 있는지 생각해봅시다."     (QT reflection 형식)
           ✓ "주의 이름을 부르며 성령의 탄식에 나를 맡긴다."
- prayer : 2~3문장. (QT prayer 5~6문장을 그대로 복사하지 않는다)

QUALITY CHECKS (추가분):
✓ infographic 객체의 6개 필드(guide/follow/extra/core/apply/prayer)가 모두 존재한다
✓ follow 는 3~5개, apply 는 2~3개이다
✓ apply 항목이 "생각해봅시다/살펴봅시다/확인해봅시다"로 끝나지 않는다
✓ guide 와 prayer 가 QT 본문과 같은 문장을 그대로 복사하지 않았다
✓ 이 블록 안에 "설교"라는 단어가 없다
```

이 체크리스트 항목은 5-2 완화책 3의 **필드 단위 검증 메시지와 1:1로 대응**시키는 것이 좋습니다. 프롬프트가 요구하는 조건과 코드가 검사하는 조건이 같아야, 사용자에게 "무엇을 다시 시키면 되는지"를 정확히 안내할 수 있습니다.

---

## 부록 - 검토한 파일

| 파일 | 확인 내용 |
|---|---|
| [var/conf/prompt_qt_json.md](../var/conf/prompt_qt_json.md) | QT 프롬프트 원문 (126행) |
| [var/conf/prompt_infographic.md](../var/conf/prompt_infographic.md) | 인포그래픽 프롬프트 원문 (93행) |
| [service/prompt_qt_json.go](../service/prompt_qt_json.go) | 프롬프트 조립, audience별 규칙 4종, JSON 스키마 상수 |
| [service/prompt_infographic.go](../service/prompt_infographic.go) | 인포그래픽 프롬프트 로드/기본값 생성 |
| [service/infographic_service.go](../service/infographic_service.go) | **LLM 미호출** 확인, 프롬프트+전사문 조립 |
| [service/qtstep2_service.go](../service/qtstep2_service.go) | Load/Save 전체 덮어쓰기 구조, `preserveInternalBlogFields` 패턴 |
| [service/blog_service.go](../service/blog_service.go) | JSON→HTML 렌더러 패턴 (신규 렌더러의 참고 모델) |
| [service/qtstep3_service.go](../service/qtstep3_service.go) | 산출물 생성 순서, infographic 호출 지점 |
| [service/llm_service.go](../service/llm_service.go) | JSON 정리/콤마 복구 로직 |
| [service/types.go](../service/types.go) | `QTStep2Data` 평면 구조체, `QTStep3Result` |
| [service/history_service.go](../service/history_service.go) | **평면 페이로드 저장/복원 구조 확인** — `flatQTStep2Data`, `buildQTSectionDocFromFlat`, `PrepareReworkFromHistory` |
| [frontend/src/components/qt/bindQTStep3.js](../frontend/src/components/qt/bindQTStep3.js) | Step3 요청 페이로드(PDF/PNG만), `appState.output.infographicFile` |
| [frontend/src/components/history/historyWorkspace.js](../frontend/src/components/history/historyWorkspace.js) | 재작업 진입 흐름 (Step1 건너뛰고 Step2로 진입) |
| [frontend/src/state/appState.js](../frontend/src/state/appState.js) | audience별 step1/2/3 상태 |
| [app.go](../app.go) | `BuildPrompt`, `SaveManualLLMResult` |
| [frontend/src/components/qt/qtStep2.js](../frontend/src/components/qt/qtStep2.js) | 편집 필드 구성 (고정 개수) |
| [frontend/src/components/qt/bindQTStep1.js](../frontend/src/components/qt/bindQTStep1.js) | 프롬프트 복사 → 수동 결과 저장 흐름 |
| var/temp/temp.json | 실제 산출 JSON (`support_scriptures_full` 누락 확인) |
| var/temp/infographic.md | **프롬프트 문서임** 확인 |
| var/temp/blog.html, temp.html | 렌더 결과 구조 |
| [doc/260722_S2QT Infographic Style Guide.md](260722_S2QT%20Infographic%20Style%20Guide.md) | 용어 규칙 배경 |
