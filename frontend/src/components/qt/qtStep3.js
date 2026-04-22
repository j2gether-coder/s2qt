function escapeHtml(value) {
  return String(value ?? '')
    .replaceAll('&', '&amp;')
    .replaceAll('<', '&lt;')
    .replaceAll('>', '&gt;')
    .replaceAll('"', '&quot;')
    .replaceAll("'", '&#39;');
}

function renderOpenButton(filePath, formatKey) {
  const value = String(filePath || '').trim();

  if (!value) {
    return `
      <button class="button-ghost output-save-btn" type="button" disabled>
        링크 열기
      </button>
    `;
  }

  return `
    <button
      class="button-ghost output-save-btn"
      type="button"
      data-format="${escapeHtml(formatKey)}"
      data-file="${escapeHtml(value)}"
      data-action="open"
    >
      링크 열기
    </button>
  `;
}

function renderSaveButton(filePath, formatKey) {
  const value = String(filePath || '').trim();

  if (!value) {
    return `
      <button class="button-ghost output-save-btn" type="button" disabled>
        파일 저장
      </button>
    `;
  }

  return `
    <button
      class="button-ghost output-save-btn"
      type="button"
      data-format="${escapeHtml(formatKey)}"
      data-file="${escapeHtml(value)}"
      data-action="save"
    >
      파일 저장
    </button>
  `;
}

function renderOutputItem(title, filePath, statusText, openWrapId, saveWrapId, statusId, formatKey) {
  return `
    <div class="output-result-card">
      <div class="output-result-head">
        <div class="output-result-title">${escapeHtml(title)}</div>
        <div class="output-result-status" id="${statusId}">${escapeHtml(statusText || '대기')}</div>
      </div>

      <div class="topgap-sm" id="${openWrapId}">
        ${renderOpenButton(filePath, formatKey)}
      </div>

      <div class="topgap-sm" id="${saveWrapId}">
        ${renderSaveButton(filePath, formatKey)}
      </div>
    </div>
  `;
}

export function renderQTStep3(audienceId, appState) {
  const output = appState?.output || {};

  return `
    <section class="workspace-step-panel">
      <section class="card card-plain">
        <div class="step-badge">Step3. QT 문서 생성</div>
        <p class="body-note topgap-sm">QT 결과물에 대해 선택한 후 저장해 주세요.</p>
        <div id="qt-step3-message" class="ui-inline-message hidden"></div>
      </section>

      <section class="card">
        <div class="hint">다음의 확장자로 결과물이 생성됩니다.</div>

        <div class="output-check-line topgap">
          <label>
            <input type="checkbox" id="makePdfChk" checked />
            <span>PDF</span>
          </label>

          <label>
            <input type="checkbox" id="makePngChk" checked />
            <span>PNG</span>
          </label>
        </div>

        <div class="single-action-row topgap-sm">
          <button id="runQtOutputBtn" class="button" type="button">실행</button>
        </div>
      </section>

      <section class="output-result-grid qt-step3-result-grid">
        ${renderOutputItem('PDF', output.pdfFile || '', output.pdfFile ? '완료' : '대기', 'pdfOpenBtnWrap', 'pdfSaveBtnWrap', 'pdfFileStatus', 'pdf')}
        ${renderOutputItem('PNG', output.pngFile || '', output.pngFile ? '완료' : '대기', 'pngOpenBtnWrap', 'pngSaveBtnWrap', 'pngFileStatus', 'png')}
      </section>

      <section class="step-status-grid step-status-grid-3col">
        <div class="step-status-item">
          <div class="step-status-label">Step1. AI(LLM) 이용</div>
          <div class="step-status-value" id="qtStep1DoneState">대기</div>
        </div>

        <div class="step-status-item">
          <div class="step-status-label">Step2. 검토 및 편집</div>
          <div class="step-status-value" id="qtStep2DoneState">대기</div>
        </div>

        <div class="step-status-item">
          <div class="step-status-label">Step3. QT 문서 생성</div>
          <div class="step-status-value" id="qtStep3DoneState">대기</div>
        </div>
      </section>

      <section class="step-bottom-bar double">
        <div class="step-bottom-left">
          <button id="backToStep2Btn" class="button-ghost" type="button">이전</button>
        </div>
        <div class="step-bottom-right">
          <button id="finishQtFlowBtn" class="button" type="button">종료</button>
        </div>
      </section>
    </section>
  `;
}