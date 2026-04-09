import { RunQTStep3, OpenGeneratedFile, SaveGeneratedFile } from '../../../wailsjs/go/main/App';
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
    png: 'pngFileStatus',

    /*
    TODO(step3-docx-pptx):
    DOCX / PPTX 재개 시 상태 표시 복구

    docx: 'docxFileStatus',
    pptx: 'pptxFileStatus',
    */
  };

  const fileMap = {
    html: 'htmlFilePath',
    pdf: 'pdfFilePath',
    png: 'pngFilePath',

    /*
    TODO(step3-docx-pptx):
    DOCX / PPTX 재개 시 파일 경로 표시 복구

    docx: 'docxFilePath',
    pptx: 'pptxFilePath',
    */
  };

  setText(statusMap[resultKey], item?.status || '대기');
  setFileLink(fileMap[resultKey], item?.filePath || '');
}

function buildStep3Payload() {
  return {
    makeHtml: getChecked('makeHtmlChk'),
    makePdf: getChecked('makePdfChk'),
    makePng: getChecked('makePngChk'),

    /*
    TODO(step3-docx-pptx):
    DOCX / PPTX 재개 시 체크박스 payload 복구

    makeDocx: getChecked('makeDocxChk'),
    makePptx: getChecked('makePptxChk'),
    */

    dpi: 300,
  };
}

function bindFileOpenLinks() {
  const links = document.querySelectorAll('.output-file-link[data-file]');

  links.forEach((link) => {
    link.addEventListener('click', async (event) => {
      event.preventDefault();

      const filePath = link.dataset.file || '';
      if (!filePath.trim()) {
        window.alert('열 파일이 없습니다.');
        return;
      }

      try {
        await OpenGeneratedFile(filePath);
      } catch (error) {
        console.error(error);
        window.alert(error?.message || '파일 열기 중 오류가 발생했습니다.');
      }
    });
  });
}

function bindSaveAsButtons(audienceId) {
  const buttons = document.querySelectorAll('.output-save-btn[data-file]');

  buttons.forEach((btn) => {
    btn.addEventListener('click', async () => {
      const filePath = btn.dataset.file || '';
      const formatKey = btn.dataset.format || '';

      if (!filePath.trim()) {
        window.alert('저장할 파일이 없습니다.');
        return;
      }

      try {
        const savedPath = await SaveGeneratedFile(filePath, audienceId, formatKey);
        if (savedPath) {
          window.alert('파일 저장이 완료되었습니다.');
        }
      } catch (error) {
        console.error(error);
        window.alert(error?.message || '파일 저장 중 오류가 발생했습니다.');
      }
    });
  });
}

export function bindQTStep3Events(audienceId) {
  const runBtn = document.getElementById('runQtOutputBtn');
  const backBtn = document.getElementById('backToStep2Btn');
  const finishBtn = document.getElementById('finishQtFlowBtn');

  bindFileOpenLinks();
  bindSaveAsButtons(audienceId);

  if (runBtn) {
    runBtn.addEventListener('click', async () => {
      try {
        const req = buildStep3Payload();
        const result = await RunQTStep3(req);

        applyOne('html', result?.html);
        applyOne('pdf', result?.pdf);
        applyOne('png', result?.png);

        /*
        TODO(step3-docx-pptx):
        DOCX / PPTX 재개 시 결과 반영 복구

        applyOne('docx', result?.docx);
        applyOne('pptx', result?.pptx);
        */

        bindFileOpenLinks();
        bindSaveAsButtons(audienceId);

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