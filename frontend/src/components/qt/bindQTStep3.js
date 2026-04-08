import { RunQTStep3, SaveQTOutputAs } from '../../../wailsjs/go/main/App';
import { setAudienceStep } from '../../state/appState';
import { mountAppShell } from '../appShell';

function getChecked(id) {
  return !!document.getElementById(id)?.checked;
}

function setText(id, value) {
  const el = document.getElementById(id);
  if (el) el.textContent = value || '';
}

function setFileLink(containerId, filePath) {
  const el = document.getElementById(containerId);
  if (!el) return;

  if (!filePath) {
    el.innerHTML = '-';
    return;
  }

  el.innerHTML = `
    <a href="#" class="output-file-link" data-file="${filePath}">
      ${filePath}
    </a>
  `;
}

function applyOne(resultKey, item) {
  const statusMap = {
    html: 'htmlFileStatus',
    pdf: 'pdfFileStatus',
    docx: 'docxFileStatus',
    pptx: 'pptxFileStatus',
    png: 'pngFileStatus',
  };

  const fileMap = {
    html: 'htmlFilePath',
    pdf: 'pdfFilePath',
    docx: 'docxFilePath',
    pptx: 'pptxFilePath',
    png: 'pngFilePath',
  };

  setText(statusMap[resultKey], item?.status || '대기');
  setFileLink(fileMap[resultKey], item?.filePath || '');
}

function buildStep3Payload() {
  return {
    makeHtml: getChecked('makeHtmlChk'),
    makePdf: getChecked('makePdfChk'),
    makeDocx: getChecked('makeDocxChk'),
    makePptx: getChecked('makePptxChk'),
    makePng: getChecked('makePngChk'),
    dpi: 300,
  };
}

export function bindQTStep3Events(audienceId) {
  const runBtn = document.getElementById('runQtOutputBtn');
  const backBtn = document.getElementById('backToStep2Btn');
  const finishBtn = document.getElementById('finishQtFlowBtn');

  bindSaveAsButtons();
  
  if (runBtn) {
    runBtn.addEventListener('click', async () => {
      try {
        const req = buildStep3Payload();
        const result = await RunQTStep3(req);

        applyOne('html', result?.html);
        applyOne('pdf', result?.pdf);
        applyOne('docx', result?.docx);
        applyOne('pptx', result?.pptx);
        applyOne('png', result?.png);

        bindSaveAsButtons

        setText('qtStep1DoneState', '완료');
        setText('qtStep2DoneState', '완료');
        setText('qtStep3DoneState', '완료');
      } catch (error) {
        console.error(error);
        window.alert(error?.message || 'Step3 실행 중 오류가 발생했습니다.');
      }
    });
  }

  if (backBtn) {
    backBtn.addEventListener('click', () => {
      setAudienceStep(audienceId, 'step2');
      mountAppShell('app');
    });
  }

  if (finishBtn) {
    finishBtn.addEventListener('click', () => {
      window.alert('QT 문서 생성 작업이 완료되었습니다.');
    });
  }
}

function bindSaveAsButtons() {
  const buttons = document.querySelectorAll('.output-save-btn[data-file]');

  buttons.forEach((btn) => {
    btn.addEventListener('click', async () => {
      const filePath = btn.dataset.file || '';
      const format = btn.dataset.format || '';

      if (!filePath.trim()) {
        window.alert('저장할 파일이 없습니다.');
        return;
      }

      try {
        await SaveQTOutputAs({
          sourcePath: filePath,
          format,
        });
      } catch (error) {
        console.error(error);
        window.alert(error?.message || '파일 저장 중 오류가 발생했습니다.');
      }
    });
  });
}