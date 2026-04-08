import { appState, updateOutput, updateStatus } from '../state/appState';
import { refreshShellStatus } from '../components/appShell';
import {
  SaveHtmlAndMakePDF,
  OpenFile
} from '../../wailsjs/go/main/App';

function escapeHtml(value) {
  return String(value ?? '')
    .replaceAll('&', '&amp;')
    .replaceAll('<', '&lt;')
    .replaceAll('>', '&gt;')
    .replaceAll('"', '&quot;')
    .replaceAll("'", '&#39;');
}

function syncOutputReady() {
  const ready = !!(
    appState.output.htmlFile?.trim() ||
    appState.output.pdfFile?.trim()
  );

  updateStatus({ outputReady: ready });
  refreshShellStatus();

  const statusEl = document.getElementById('outputStepStatus');
  if (statusEl) {
    statusEl.textContent = ready
      ? '출력물이 생성되었습니다.'
      : '실행 버튼을 눌러 산출물을 생성해 주세요.';
  }

  const step4StatusEl = document.getElementById('step4StatusText');
  if (step4StatusEl) {
    step4StatusEl.textContent = ready ? '완료' : '대기';
  }
}

async function handleRunOutput() {
  const finalHtml = appState.output.finalHtml || appState.editor.editedHtml || '';

  if (!finalHtml.trim()) {
    updateOutput({
      htmlFile: '',
      pdfFile: '',
      outputMessage: '생성할 최종 내용이 없습니다. Step3 편집 내용을 먼저 확인해 주세요.'
    });
    renderOutputState();
    syncOutputReady();
    return;
  }

  try {
    updateOutput({
      finalHtml,
      outputMessage: '산출물 생성 중입니다...'
    });
    renderOutputState();

    const result = await SaveHtmlAndMakePDF(finalHtml);

    if (!result || result.success !== true) {
      updateOutput({
        htmlFile: '',
        pdfFile: '',
        outputMessage: result?.message || '산출물 생성에 실패했습니다.'
      });
    } else {
      updateOutput({
        finalHtml,
        htmlFile: result.htmlFile || '',
        pdfFile: result.pdfFile || '',
        outputMessage: result.message || '산출물 생성이 완료되었습니다.'
      });
    }
  } catch (error) {
    updateOutput({
      htmlFile: '',
      pdfFile: '',
      outputMessage: `산출물 생성 오류: ${String(error)}`
    });
  }

  renderOutputState();
  syncOutputReady();
}

function renderFileLink(filePath) {
  if (!filePath || filePath === '-') {
    return '-';
  }

  return `<a href="#" class="output-file-link" data-file="${escapeHtml(filePath)}">${escapeHtml(filePath)}</a>`;
}

function renderOutputState() {
  const htmlFileEl = document.getElementById('htmlFile');
  const pdfFileEl = document.getElementById('pdfFile');
  const outputMessageEl = document.getElementById('outputMessage');

  const step1StatusEl = document.getElementById('step1StatusText');
  const step2StatusEl = document.getElementById('step2StatusText');
  const step3StatusEl = document.getElementById('step3StatusText');
  const step4StatusEl = document.getElementById('step4StatusText');

  if (htmlFileEl) htmlFileEl.innerHTML = renderFileLink(appState.output.htmlFile || '-');
  if (pdfFileEl) pdfFileEl.innerHTML = renderFileLink(appState.output.pdfFile || '-');
  if (outputMessageEl) outputMessageEl.textContent = appState.output.outputMessage || '-';

  if (step1StatusEl) step1StatusEl.textContent = appState.status.inputReady ? '완료' : '대기';
  if (step2StatusEl) step2StatusEl.textContent = appState.status.draftReady ? '완료' : '대기';
  if (step3StatusEl) step3StatusEl.textContent = appState.status.editReady ? '완료' : '대기';
  if (step4StatusEl) step4StatusEl.textContent = appState.status.outputReady ? '완료' : '대기';

  bindOutputFileLinkEvents();
}

function bindOutputFileLinkEvents() {
  const linkEls = document.querySelectorAll('.output-file-link');

  linkEls.forEach((linkEl) => {
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
  });
}

export function renderOutputStep() {
  return `
    <section class="card">
      <div class="hint">
        Step3에서 확정된 최종 내용을 산출물로 생성합니다.
      </div>
      <div class="status" id="outputStepStatus">
        ${
          appState.status.outputReady
            ? '출력물이 생성되었습니다.'
            : '실행 버튼을 눌러 산출물을 생성해 주세요.'
        }
      </div>
    </section>

    <section class="card">
      <h3 class="mini-title">Step4. 산출물 생성</h3>

      <div class="output-action-row topgap-sm">
        <button id="runOutputBtn" class="button" type="button">실행</button>
      </div>

      <div class="fileitem topgap">
        <strong>상태 메시지:</strong>
        <span id="outputMessage">${escapeHtml(appState.output.outputMessage || '-')}</span>
      </div>
    </section>

    <section class="card">
      <h3 class="mini-title">출력물 정보</h3>

      <div class="fileitem"><strong>HTML:</strong> <span id="htmlFile">${renderFileLink(appState.output.htmlFile || '-')}</span></div>
      <div class="fileitem"><strong>PDF:</strong> <span id="pdfFile">${renderFileLink(appState.output.pdfFile || '-')}</span></div>
      <div class="fileitem"><strong>DOCX:</strong> <span>-</span></div>
      <div class="fileitem"><strong>PPT:</strong> <span>-</span></div>
    </section>

    <section class="card">
      <h3 class="mini-title">단계별 절차 상태 정보</h3>

      <div class="step-status-grid">
        <div class="step-status-item">
          <div class="step-status-label">Step1. 자료입력</div>
          <div class="step-status-value" id="step1StatusText">${appState.status.inputReady ? '완료' : '대기'}</div>
        </div>

        <div class="step-status-item">
          <div class="step-status-label">Step2. 프롬프트 생성</div>
          <div class="step-status-value" id="step2StatusText">${appState.status.draftReady ? '완료' : '대기'}</div>
        </div>

        <div class="step-status-item">
          <div class="step-status-label">Step3. 본문편집</div>
          <div class="step-status-value" id="step3StatusText">${appState.status.editReady ? '완료' : '대기'}</div>
        </div>

        <div class="step-status-item">
          <div class="step-status-label">Step4. 산출물 생성</div>
          <div class="step-status-value" id="step4StatusText">${appState.status.outputReady ? '완료' : '대기'}</div>
        </div>
      </div>
    </section>
  `;
}

export function bindOutputStepEvents() {
  const runOutputBtn = document.getElementById('runOutputBtn');

  if (runOutputBtn) {
    runOutputBtn.addEventListener('click', handleRunOutput);
  }

  renderOutputState();
  syncOutputReady();
}