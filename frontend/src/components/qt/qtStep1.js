function escapeHtml(value) {
  return String(value ?? '')
    .replaceAll('&', '&amp;')
    .replaceAll('<', '&lt;')
    .replaceAll('>', '&gt;')
    .replaceAll('"', '&quot;')
    .replaceAll("'", '&#39;');
}

export function renderQTStep1(audienceId, appState) {
  const hasApiKey = false; // 추후 실제 값 연결
  const modeText = hasApiKey ? '자동' : '수동';

  return `
    <section class="workspace-step-panel">
      <section class="card card-plain">
        <div class="step-badge">Step1. AI(LLM) 이용</div>
        <p class="body-note topgap-sm">AI(LLM)를 이용하여 초안 작업을 합니다.</p>
        <div id="qt-step1-message" class="ui-inline-message hidden"></div>
      </section>

      <section class="card">
        <div class="row single-action-row">
          <button id="buildPromptBtn" class="button" type="button">프롬프트 생성</button>
        </div>

        <textarea
          id="qtPromptPreview"
          class="promptbox-sm topgap"
          placeholder="여기에 AI(LLM)용 프롬프트가 표시됩니다."
        ></textarea>

        <div class="row single-action-row topgap-sm">
          <button id="copyPromptBtn" class="button-ghost" type="button">복사</button>
        </div>
      </section>

      <section class="card">
        <div class="row between center">
          <h3 class="mini-title">AI(LLM) 처리 및 결과 저장</h3>

          <div class="mode-strip">
            <span class="mode-label">현재 모드</span>
            <span class="mode-value" id="llmModeBadge">${modeText}</span>
          </div>
        </div>

        ${
          hasApiKey
            ? `
              <div class="row single-action-row topgap-sm">
                <button id="runLlmBtn" class="button-success" type="button">LLM 실행</button>
              </div>
            `
            : `
              <div class="hint topgap-sm" id="llmGuideText">
                외부 AI(LLM) 결과를 붙여 넣는 방식입니다.
              </div>
            `
        }

        <textarea
          id="qtResultText"
          class="resultbox-sm topgap"
          placeholder="AI(LLM) 결과를 확인하거나 붙여넣어 주세요."
        ></textarea>

        <div class="row single-action-row topgap-sm">
          <button id="saveResultBtn" class="button" type="button">결과 저장</button>
        </div>
      </section>

      <section class="step-bottom-bar single">
        <button id="goStep2Btn" class="button" type="button" disabled>다음</button>
      </section>
    </section>
  `;
}