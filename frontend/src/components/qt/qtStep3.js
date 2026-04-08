function escapeHtml(value) {
  return String(value ?? '')
    .replaceAll('&', '&amp;')
    .replaceAll('<', '&lt;')
    .replaceAll('>', '&gt;')
    .replaceAll('"', '&quot;')
    .replaceAll("'", '&#39;');
}

function renderFileLink(filePath) {
  const value = String(filePath || '').trim();

  if (!value) {
    return '-';
  }

  return `
    <a href="#" class="output-file-link" data-file="${escapeHtml(value)}">
      ${escapeHtml(value)}
    </a>
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
    >
      파일 저장
    </button>
  `;
}

function renderOutputItem(title, filePath, statusText, fileId, statusId, formatKey) {
  return `
    <div class="output-result-card">
      <div class="output-result-head">
        <div class="output-result-title">${escapeHtml(title)}</div>
        <div class="output-result-status" id="${statusId}">${escapeHtml(statusText || '대기')}</div>
      </div>

      <div class="file-line topgap-sm">
        파일: <span id="${fileId}">${renderFileLink(filePath)}</span>
      </div>

      <div class="topgap-sm">
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
      </section>

      <section class="card">
        <div class="hint">다음의 확장자로 결과물이 생성됩니다.</div>

        <div class="output-check-line topgap">
          <label class="simple-check-item">
            <input type="checkbox" id="makeHtmlChk" checked />
            <span>HTML</span>
          </label>

          <label class="simple-check-item">
            <input type="checkbox" id="makePdfChk" checked />
            <span>PDF</span>
          </label>

          <label class="simple-check-item">
            <input type="checkbox" id="makeDocxChk" />
            <span>DOCX</span>
          </label>

          <label class="simple-check-item">
            <input type="checkbox" id="makePptxChk" />
            <span>PPTX</span>
          </label>

          <label class="simple-check-item">
            <input type="checkbox" id="makePngChk" />
            <span>PNG</span>
          </label>
        </div>

        <div class="single-action-row topgap-sm">
          <button id="runQtOutputBtn" class="button" type="button">실행</button>
        </div>
      </section>

      <section class="output-result-grid output-result-grid-3col">
        ${renderOutputItem('HTML', output.htmlFile || '', output.htmlFile ? '완료' : '대기', 'htmlFilePath', 'htmlFileStatus', 'html')}
        ${renderOutputItem('PDF', output.pdfFile || '', output.pdfFile ? '완료' : '대기', 'pdfFilePath', 'pdfFileStatus', 'pdf')}
        ${renderOutputItem('DOCX', output.docxFile || '', output.docxFile ? '완료' : '대기', 'docxFilePath', 'docxFileStatus', 'docx')}
        ${renderOutputItem('PPTX', output.pptxFile || '', output.pptxFile ? '완료' : '대기', 'pptxFilePath', 'pptxFileStatus', 'pptx')}
        ${renderOutputItem('PNG', output.pngFile || '', output.pngFile ? '완료' : '대기', 'pngFilePath', 'pngFileStatus', 'png')}
      </section>

      <section class="step-status-grid step-status-grid-3col">
        <div class="step-status-item">
          <div class="step-status-label">Step1. LLM 이용</div>
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
          <button id="finishQtFlowBtn" class="button" type="button" disabled>종료</button>
        </div>
      </section>
    </section>
  `;
}