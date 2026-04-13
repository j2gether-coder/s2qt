import { RunQTStep3, OpenGeneratedFile, SaveGeneratedFile, FinishCurrentRun } from '../../../wailsjs/go/main/App';
import { Quit } from '../../../wailsjs/runtime/runtime';
import {
  appState,
  setAudienceStep,
  getAudienceStepStatus,
  setAudienceStepStatus,
  getStepStatusLabel,
} from '../../state/appState';
import { mountAppShell } from '../appShell';
import { showToast, setInlineMessage, clearInlineMessage } from "../../common/uiMessage";

let isStep3Running = false;

function getChecked(id) {
  return !!document.getElementById(id)?.checked;
}

function setText(id, value) {
  const el = document.getElementById(id);
  if (el) el.textContent = value || '';
}

function setOpenButton(containerId, filePath, formatKey) {
  const el = document.getElementById(containerId);
  if (!el) return;

  const value = String(filePath || '').trim();
  const format = String(formatKey || '').trim();

  if (!value) {
    el.innerHTML = `
      <button class="button-ghost output-save-btn" type="button" disabled>
        링크 열기
      </button>
    `;
    return;
  }

  el.innerHTML = `
    <button
      class="button-ghost output-save-btn"
      type="button"
      data-format="${format}"
      data-file="${value}"
      data-action="open"
    >
      링크 열기
    </button>
  `;
}

function setSaveButton(containerId, filePath, formatKey) {
  const el = document.getElementById(containerId);
  if (!el) return;

  const value = String(filePath || '').trim();
  const format = String(formatKey || '').trim();

  if (!value) {
    el.innerHTML = `
      <button class="button-ghost output-save-btn" type="button" disabled>
        파일 저장
      </button>
    `;
    return;
  }

  el.innerHTML = `
    <button
      class="button-ghost output-save-btn"
      type="button"
      data-format="${format}"
      data-file="${value}"
      data-action="save"
    >
      파일 저장
    </button>
  `;
}

function applyOne(resultKey, item) {
  const statusMap = {
    html: 'htmlFileStatus',
    pdf: 'pdfFileStatus',
    png: 'pngFileStatus',
  };

  const openBtnMap = {
    html: 'htmlOpenBtnWrap',
    pdf: 'pdfOpenBtnWrap',
    png: 'pngOpenBtnWrap',
  };

  const saveBtnMap = {
    html: 'htmlSaveBtnWrap',
    pdf: 'pdfSaveBtnWrap',
    png: 'pngSaveBtnWrap',
  };

  setText(statusMap[resultKey], item?.status || '대기');
  setOpenButton(openBtnMap[resultKey], item?.filePath || '', resultKey);
  setSaveButton(saveBtnMap[resultKey], item?.filePath || '', resultKey);
}

function buildStep3Payload() {
  return {
    makeHtml: getChecked('makeHtmlChk'),
    makePdf: getChecked('makePdfChk'),
    makePng: getChecked('makePngChk'),
    dpi: 300,
  };
}

function renderStepStatusFromState(audienceId) {
  const status = getAudienceStepStatus(audienceId);

  setText('qtStep1DoneState', getStepStatusLabel(status.step1));
  setText('qtStep2DoneState', getStepStatusLabel(status.step2));
  setText('qtStep3DoneState', getStepStatusLabel(status.step3));
}

function markSelectedOutputsRunning() {
  if (getChecked('makeHtmlChk')) {
    setText('htmlFileStatus', '생성중...');
  } else {
    setText('htmlFileStatus', '선택안함');
  }

  if (getChecked('makePdfChk')) {
    setText('pdfFileStatus', '생성중...');
  } else {
    setText('pdfFileStatus', '선택안함');
  }

  if (getChecked('makePngChk')) {
    setText('pngFileStatus', '생성중...');
  } else {
    setText('pngFileStatus', '선택안함');
  }
}

function updateOutputState(result) {
  if (!appState.output) {
    appState.output = {};
  }

  appState.output.htmlFile = result?.html?.filePath || '';
  appState.output.pdfFile = result?.pdf?.filePath || '';
  appState.output.pngFile = result?.png?.filePath || '';
}

function updateStep3ButtonState() {
  const runBtn = document.getElementById('runQtOutputBtn');
  const backBtn = document.getElementById('backToStep2Btn');
  const finishBtn = document.getElementById('finishQtFlowBtn');
  const progressText = document.getElementById('qtStep3ProgressText');

  if (runBtn) {
    runBtn.disabled = isStep3Running;
    runBtn.textContent = isStep3Running ? '실행중...' : '실행';
    runBtn.classList.toggle('button-running', isStep3Running);
  }

  if (backBtn) {
    backBtn.disabled = isStep3Running;
  }

  if (finishBtn) {
    finishBtn.disabled = isStep3Running;
  }

  if (progressText) {
    progressText.textContent = isStep3Running
      ? '산출물을 생성하고 있습니다. 잠시만 기다려 주세요.'
      : '';
    progressText.classList.toggle('is-running', isStep3Running);
  }
}

function bindOpenButtons() {
  const buttons = document.querySelectorAll('.output-save-btn[data-file][data-action="open"]');

  buttons.forEach((btn) => {
    btn.onclick = async () => {
      clearInlineMessage("qt-step3-message");

      const filePath = btn.dataset.file || '';
      if (!filePath.trim()) {
        setInlineMessage("qt-step3-message", '열 파일이 없습니다.', "warning");
        return;
      }

      try {
        await OpenGeneratedFile(filePath);
        showToast('파일을 열었습니다.', 'success');
      } catch (error) {
        console.error(error);
        setInlineMessage("qt-step3-message", error?.message || '파일 열기 중 오류가 발생했습니다.', "error");
      }
    };
  });
}

function bindSaveAsButtons(audienceId) {
  const buttons = document.querySelectorAll('.output-save-btn[data-file][data-action="save"]');

  buttons.forEach((btn) => {
    btn.onclick = async () => {
      clearInlineMessage("qt-step3-message");

      const filePath = btn.dataset.file || '';
      const formatKey = btn.dataset.format || '';

      if (!filePath.trim()) {
        setInlineMessage("qt-step3-message", '저장할 파일이 없습니다.', "warning");
        return;
      }

      try {
        const savedPath = await SaveGeneratedFile(filePath, audienceId, formatKey);
        if (savedPath) {
          showToast('파일 저장이 완료되었습니다.', 'success');
        }
      } catch (error) {
        console.error(error);
        setInlineMessage("qt-step3-message", error?.message || '파일 저장 중 오류가 발생했습니다.', "error");
      }
    };
  });
}

async function runStep3(audienceId) {
  if (isStep3Running) return;

  clearInlineMessage("qt-step3-message");

  try {
    isStep3Running = true;
    updateStep3ButtonState();

    setAudienceStepStatus(audienceId, 'step3', 'running');
    renderStepStatusFromState(audienceId);
    markSelectedOutputsRunning();

    const req = buildStep3Payload();
    const result = await RunQTStep3(req);

    updateOutputState(result);

    applyOne('html', result?.html);
    applyOne('pdf', result?.pdf);
    applyOne('png', result?.png);

    bindOpenButtons();
    bindSaveAsButtons(audienceId);

    setAudienceStepStatus(audienceId, 'step3', 'done');
    renderStepStatusFromState(audienceId);
    showToast('Step3 산출물 생성을 완료했습니다.', 'success');
  } catch (error) {
    console.error(error);
    setAudienceStepStatus(audienceId, 'step3', 'error');
    renderStepStatusFromState(audienceId);
    setInlineMessage("qt-step3-message", error?.message || 'Step3 실행 중 오류가 발생했습니다.', "error");
  } finally {
    isStep3Running = false;
    updateStep3ButtonState();
  }
}

async function finishStep3Flow() {
  if (isStep3Running) return;

  clearInlineMessage("qt-step3-message");

  try {
    await FinishCurrentRun();
    await Quit();
  } catch (error) {
    console.error(error);
    setInlineMessage("qt-step3-message", error?.message || '종료 중 오류가 발생했습니다.', "error");
  }
}

export function bindQTStep3Events(audienceId) {
  const runBtn = document.getElementById('runQtOutputBtn');
  const backBtn = document.getElementById('backToStep2Btn');
  const finishBtn = document.getElementById('finishQtFlowBtn');

  renderStepStatusFromState(audienceId);

  bindOpenButtons();
  bindSaveAsButtons(audienceId);
  updateStep3ButtonState();

  if (runBtn) {
    runBtn.onclick = async () => {
      await runStep3(audienceId);
    };
  }

  if (backBtn) {
    backBtn.onclick = () => {
      if (isStep3Running) return;
      setAudienceStep(audienceId, 'step2');
      mountAppShell('app');
    };
  }

  if (finishBtn) {
    finishBtn.onclick = async () => {
      await finishStep3Flow();
    };
  }
}