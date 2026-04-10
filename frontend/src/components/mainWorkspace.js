import {
  appState,
  getMenuLabel,
  getSourceStatusLabel,
  setSourceType,
  setSourceUrl,
  setSourceFilePath,
  setRawText,
  setBasicInfoField,
  setSourceStatus,
  setAudienceStep,
} from '../state/appState';
import { mountAppShell } from './appShell';
import { showToast, setInlineMessage, clearInlineMessage } from "../common/uiMessage";
import {
  SelectAudioFile,
  SelectTextFile,
  LoadTextFile,
  RunSourcePrepare,
  GetVideoMeta,
} from '../../wailsjs/go/main/App';
import { renderQTStep1 } from './qt/qtStep1';
import { renderQTStep2 } from './qt/qtStep2';
import { renderQTStep3 } from './qt/qtStep3';
import { bindQTStep1Events } from './qt/bindQTStep1';
import { bindQTStep2Events } from './qt/bindQTStep2';
import { bindQTStep3Events } from './qt/bindQTStep3';

function escapeHtml(value) {
  return String(value ?? '')
    .replaceAll('&', '&amp;')
    .replaceAll('<', '&lt;')
    .replaceAll('>', '&gt;')
    .replaceAll('"', '&quot;')
    .replaceAll("'", '&#39;');
}

function isAudienceMenu(menu) {
  return ['adult', 'young_adult', 'teen', 'child'].includes(menu);
}

function getCurrentAudienceStep(audienceId) {
  return appState.audienceSteps?.[audienceId] || 'step1';
}

function getAudienceStatusText(audienceId) {
  const step = getCurrentAudienceStep(audienceId);

  if (step === 'step3') return '문서 생성 단계';
  if (step === 'step2') return '검토 및 편집 단계';
  return 'AI(LLM) 이용 단계';
}

function updateQtPrepareStatus() {
  const { sourceType, basicInfo, transcript, sourceRef, sourceStatus } = appState.source;

  if (sourceStatus === 'RUNNING') return;
  if (sourceStatus === 'COMPLETED') return;

  const hasTitle = (basicInfo.title || '').trim() !== '';
  const hasBibleText = (basicInfo.bibleText || '').trim() !== '';

  let hasSourceInput = false;

  if (sourceType === 'video') {
    hasSourceInput = (sourceRef.url || '').trim() !== '';
  } else if (sourceType === 'audio') {
    hasSourceInput = (sourceRef.filePath || '').trim() !== '';
  } else if (sourceType === 'text') {
    hasSourceInput =
      (transcript.rawText || '').trim() !== '' ||
      (sourceRef.filePath || '').trim() !== '';
  }

  setSourceStatus(hasTitle && hasBibleText && hasSourceInput ? 'READY' : 'NOT_READY');
}

function clearBasicInfoSavedState() {
  appState.source.basicInfoSavedAt = '';
}

function saveBasicInfoDraft() {
  appState.source.basicInfoSavedAt = new Date().toLocaleString();
}

function buildSourcePreparePayload() {
  const { source } = appState;

  if (source.sourceType === 'video') {
    return {
      sourceType: 'video',
      inputMode: 'url',
      sourceUrl: source.sourceRef.url || '',
      sourcePath: '',
      textContent: '',
    };
  }

  if (source.sourceType === 'audio') {
    return {
      sourceType: 'audio',
      inputMode: 'file',
      sourceUrl: '',
      sourcePath: source.sourceRef.filePath || '',
      textContent: '',
    };
  }

  return {
    sourceType: 'text',
    inputMode: (source.sourceRef.filePath || '').trim() ? 'file' : 'paste',
    sourceUrl: '',
    sourcePath: source.sourceRef.filePath || '',
    textContent: source.transcript.rawText || '',
  };
}

async function enrichVideoBasicInfoFromMeta() {
  const url = (appState?.source?.sourceRef?.url || '').trim();
  if (!url) return null;

  const meta = await GetVideoMeta(url);
  if (!meta) return null;

  const basicInfo = appState?.source?.basicInfo || {};
  let changed = false;

  if (!(basicInfo.title || '').trim() && (meta.title || '').trim()) {
    setBasicInfoField('title', meta.title);
    changed = true;
  }

  if (!(basicInfo.sermonDate || '').trim() && (meta.uploadDateText || '').trim()) {
    setBasicInfoField('sermonDate', meta.uploadDateText);
    changed = true;
  }

  if (!(basicInfo.churchName || '').trim() && (meta.channel || '').trim()) {
    setBasicInfoField('churchName', meta.channel);
    changed = true;
  }

  if (changed) {
    clearBasicInfoSavedState();
  }

  return meta;
}

async function runQtPrepare() {
  if (appState?.source?.sourceStatus === 'RUNNING') {
    return;
  }

  clearInlineMessage("workspace-message");

  try {
    const sourceType = appState?.source?.sourceType || '';

    if (sourceType === 'video') {
      await enrichVideoBasicInfoFromMeta();
      updateQtPrepareStatus();
      mountAppShell('app');
    }

    setSourceStatus('RUNNING');
    mountAppShell('app');

    const payload = buildSourcePreparePayload();
    const result = await RunSourcePrepare(payload);

    if (result?.rawText) {
      setRawText(result.rawText);
    }

    setSourceStatus(result?.status || 'COMPLETED');

    if (result?.success) {
      appState.source.lastSavedAt = new Date().toLocaleString();
      showToast("자료 준비가 완료되었습니다.", "success");
    }

    mountAppShell('app');
  } catch (error) {
    console.error(error);
    updateQtPrepareStatus();
    setInlineMessage("workspace-message", error?.message || '자료 처리 중 오류가 발생했습니다.', "error");
    mountAppShell('app');
  }
}

function renderSourceTypeSelector() {
  const { sourceType } = appState.source;

  return `
    <div class="radio-group">
      <label class="radio-item">
        <input type="radio" name="sourceType" value="video" ${sourceType === 'video' ? 'checked' : ''} />
        <span>동영상</span>
      </label>

      <label class="radio-item">
        <input type="radio" name="sourceType" value="audio" ${sourceType === 'audio' ? 'checked' : ''} />
        <span>오디오</span>
      </label>

      <label class="radio-item">
        <input type="radio" name="sourceType" value="text" ${sourceType === 'text' ? 'checked' : ''} />
        <span>텍스트</span>
      </label>
    </div>
  `;
}

function renderSourceInputArea() {
  const { sourceType, sourceRef, transcript, sourceStatus } = appState.source;
  const isRunning = sourceStatus === 'RUNNING';

  if (sourceType === 'video') {
    return `
      <div class="inline-form-row">
        <label class="inline-form-label">URL</label>
        <input
          type="text"
          id="source-url-input"
          value="${escapeHtml(sourceRef.url || '')}"
          placeholder="동영상 URL을 입력해 주세요."
          ${isRunning ? 'disabled' : ''}
        />
      </div>

      <div class="form-actions full-width-actions">
        <button
          class="primary-button full-width-button ${isRunning ? 'is-disabled' : ''}"
          type="button"
          id="run-source-btn"
          ${isRunning ? 'disabled' : ''}
        >
          ${isRunning ? '실행 중...' : '실행'}
        </button>
      </div>
    `;
  }

  if (sourceType === 'audio') {
    return `
      <div class="form-inline-note-row">
        <div class="form-inline-note">
          ${sourceRef.filePath ? `선택 파일: ${escapeHtml(sourceRef.filePath)}` : '오디오 파일은 파일 탐색기를 이용하세요.'}
        </div>
      </div>

      <div class="equal-action-row">
        <button
          class="secondary-button equal-action-button ${isRunning ? 'is-disabled' : ''}"
          type="button"
          id="audio-file-select-btn"
          ${isRunning ? 'disabled' : ''}
        >
          파일 탐색기
        </button>
        <button
          class="primary-button equal-action-button ${isRunning ? 'is-disabled' : ''}"
          type="button"
          id="run-audio-btn"
          ${isRunning ? 'disabled' : ''}
        >
          ${isRunning ? '실행 중...' : '실행'}
        </button>
      </div>
    `;
  }

  return `
    <div class="form-field">
      <div class="field-header-row">
        <label class="form-label">텍스트 붙여넣기</label>
        <div class="field-header-note">
          ${sourceRef.filePath ? `선택 파일: ${escapeHtml(sourceRef.filePath)}` : '&nbsp;'}
        </div>
      </div>

      <textarea
        id="raw-text-direct-input"
        placeholder="작성한 설교 원고를 메모장에 복사하여 붙이기한 순수 텍스트를 사용하세요."
        ${isRunning ? 'disabled' : ''}
      >${escapeHtml(transcript.rawText || '')}</textarea>
    </div>

    <div class="equal-action-row">
      <button
        class="secondary-button equal-action-button ${isRunning ? 'is-disabled' : ''}"
        type="button"
        id="text-file-select-btn"
        ${isRunning ? 'disabled' : ''}
      >
        파일 탐색기
      </button>
      <button
        class="primary-button equal-action-button ${isRunning ? 'is-disabled' : ''}"
        type="button"
        id="run-text-btn"
        ${isRunning ? 'disabled' : ''}
      >
        ${isRunning ? '실행 중...' : '실행'}
      </button>
    </div>
  `;
}

function renderBasicInfoSaveGuide() {
  const savedAt = appState?.source?.basicInfoSavedAt || '';
  if (!savedAt) return '';

  return `
    <div class="completion-guide topgap-sm">
      기본정보 저장 완료: ${escapeHtml(savedAt)}
    </div>
  `;
}

function renderCompletionGuide() {
  if (appState.source.sourceStatus !== 'COMPLETED') return '';

  return `
    <div class="completion-guide">
      자료 준비가 완료되었습니다. 기본정보를 확인하고 저장한 뒤, 좌측 메뉴에서 원하는 QT를 선택하여 QT 만들기를 진행해 주세요.
    </div>
  `;
}

function renderQtPrepareLayout() {
  const { basicInfo, sourceStatus } = appState.source;
  const isRunning = sourceStatus === 'RUNNING';
  const isBasicInfoSavable = sourceStatus === 'COMPLETED';

  return `
    <section class="workspace-panel">
      <div class="workspace-header-row">
        <div class="workspace-header-title">QT 자료 준비</div>
        <div class="workspace-header-status">${getSourceStatusLabel(sourceStatus || 'NOT_READY')}</div>
      </div>

      <div class="workspace-meta-note">
        QT 준비에서는 원문 확보만 수행합니다. AI(LLM) 이용은 각 QT 화면의 Step1에서 진행됩니다.<br>
        자료에 따라 소요시간은 약 5~10분정도입니다. 
      </div>

      <div id="workspace-message" class="ui-inline-message hidden"></div>

      <div class="workspace-content">
        <div class="section-block">
          <div class="section-header">자료 입력 방식</div>
          <div class="section-body">
            ${renderSourceTypeSelector()}
            <div class="section-subbody">
              ${renderSourceInputArea()}
            </div>
          </div>
        </div>

        <div class="section-block">
          <div class="section-header">
            <span>기본 정보</span>
            <span class="section-header-note">제목과 본문 성구는 필수이며, 그 외 항목은 선택입니다.</span>
          </div>

          <div class="section-body">
            <div class="form-grid two-column-grid">
              <div class="form-field">
                <label class="form-label">제목 <span class="required-mark">*</span></label>
                <input type="text" id="title-input" value="${escapeHtml(basicInfo.title || '')}" placeholder="제목을 입력해 주세요." ${isRunning ? 'disabled' : ''} />
              </div>

              <div class="form-field">
                <label class="form-label">본문 성구 <span class="required-mark">*</span></label>
                <input type="text" id="bible-text-input" value="${escapeHtml(basicInfo.bibleText || '')}" placeholder="예: 시 1:1" ${isRunning ? 'disabled' : ''} />
              </div>

              <div class="form-field">
                <label class="form-label">찬송</label>
                <input type="text" id="hymn-input" value="${escapeHtml(basicInfo.hymn || '')}" placeholder="찬송을 입력해 주세요." ${isRunning ? 'disabled' : ''} />
              </div>

              <div class="form-field">
                <label class="form-label">설교자</label>
                <input type="text" id="preacher-input" value="${escapeHtml(basicInfo.preacher || '')}" placeholder="설교자를 입력해 주세요." ${isRunning ? 'disabled' : ''} />
              </div>

              <div class="form-field">
                <label class="form-label">교회명</label>
                <input type="text" id="church-name-input" value="${escapeHtml(basicInfo.churchName || '')}" placeholder="교회명을 입력해 주세요." ${isRunning ? 'disabled' : ''} />
              </div>

              <div class="form-field">
                <label class="form-label">설교일</label>
                <input type="date" id="sermon-date-input" value="${escapeHtml(basicInfo.sermonDate || '')}" ${isRunning ? 'disabled' : ''} />
              </div>
            </div>

            <div class="form-actions topgap-sm full-width-actions">
              <button
                class="secondary-button full-width-button 
                ${!isBasicInfoSavable ? 'is-disabled' : ''}"
                type="button"
                id="save-basic-info-btn"
                ${!isBasicInfoSavable ? 'disabled' : ''}
              >
                기본정보 저장
              </button>
            </div>

            ${renderBasicInfoSaveGuide()}
            ${renderCompletionGuide()}
          </div>
        </div>
      </div>
    </section>
  `;
}

function renderAudienceStepContent(audienceId) {
  const currentStep = getCurrentAudienceStep(audienceId);

  if (currentStep === 'step2') {
    return renderQTStep2(audienceId, appState);
  }

  if (currentStep === 'step3') {
    return renderQTStep3(audienceId, appState);
  }

  return renderQTStep1(audienceId, appState);
}

function renderAudienceLayout(audienceId) {
  const label = getMenuLabel(audienceId);
  const currentStep = getCurrentAudienceStep(audienceId);

  return `
    <section class="workspace-panel">
      <div class="workspace-header-row">
        <div class="workspace-header-title">${label}</div>
        <div class="workspace-header-status">${getAudienceStatusText(audienceId)}</div>
      </div>

      <div class="workspace-meta-note">
        QT 준비에서 생성된 자료를 기반으로 작업합니다.
      </div>

      <div class="workspace-step-row">
        <button
          class="step-tab ${currentStep === 'step1' ? 'active' : ''}"
          type="button"
          data-audience-step="step1"
          data-audience-id="${audienceId}"
        >
          Step1. AI(LLM) 이용
        </button>

        <button
          class="step-tab ${currentStep === 'step2' ? 'active' : ''}"
          type="button"
          data-audience-step="step2"
          data-audience-id="${audienceId}"
        >
          Step2. 검토 및 편집
        </button>

        <button
          class="step-tab ${currentStep === 'step3' ? 'active' : ''}"
          type="button"
          data-audience-step="step3"
          data-audience-id="${audienceId}"
        >
          Step3. QT 문서 생성
        </button>
      </div>

      <div class="workspace-content">
        ${renderAudienceStepContent(audienceId)}
      </div>
    </section>
  `;
}

function bindQtPrepareEvents() {
  const radios = document.querySelectorAll('input[name="sourceType"]');
  radios.forEach((radio) => {
    radio.addEventListener('change', (e) => {
      if (appState?.source?.sourceStatus === 'RUNNING') return;
      clearInlineMessage("workspace-message");
      setSourceType(e.target.value);
      clearBasicInfoSavedState();
      updateQtPrepareStatus();
      mountAppShell('app');
    });
  });

  const sourceUrlInput = document.getElementById('source-url-input');
  if (sourceUrlInput) {
    sourceUrlInput.addEventListener('input', (e) => {
      if (appState?.source?.sourceStatus === 'RUNNING') return;
      clearInlineMessage("workspace-message");
      setSourceUrl(e.target.value);
      updateQtPrepareStatus();
    });

    sourceUrlInput.addEventListener('keydown', (e) => {
      if (e.key === 'Enter') {
        e.preventDefault();

        if (appState?.source?.sourceStatus === 'RUNNING') {
          return;
        }

        runQtPrepare();
      }
    });
  }

  const rawTextDirectInput = document.getElementById('raw-text-direct-input');
  if (rawTextDirectInput) {
    rawTextDirectInput.addEventListener('input', (e) => {
      if (appState?.source?.sourceStatus === 'RUNNING') return;
      clearInlineMessage("workspace-message");
      setRawText(e.target.value);
      updateQtPrepareStatus();
    });
  }

  const titleInput = document.getElementById('title-input');
  if (titleInput) {
    titleInput.addEventListener('input', (e) => {
      if (appState?.source?.sourceStatus === 'RUNNING') return;
      clearInlineMessage("workspace-message");
      setBasicInfoField('title', e.target.value);
      clearBasicInfoSavedState();
      updateQtPrepareStatus();
      mountAppShell('app');
    });
  }

  const bibleTextInput = document.getElementById('bible-text-input');
  if (bibleTextInput) {
    bibleTextInput.addEventListener('input', (e) => {
      if (appState?.source?.sourceStatus === 'RUNNING') return;
      clearInlineMessage("workspace-message");
      setBasicInfoField('bibleText', e.target.value);
      clearBasicInfoSavedState();
      updateQtPrepareStatus();
      mountAppShell('app');
    });
  }

  const hymnInput = document.getElementById('hymn-input');
  if (hymnInput) {
    hymnInput.addEventListener('input', (e) => {
      if (appState?.source?.sourceStatus === 'RUNNING') return;
      clearInlineMessage("workspace-message");
      setBasicInfoField('hymn', e.target.value);
      clearBasicInfoSavedState();
      mountAppShell('app');
    });
  }

  const preacherInput = document.getElementById('preacher-input');
  if (preacherInput) {
    preacherInput.addEventListener('input', (e) => {
      if (appState?.source?.sourceStatus === 'RUNNING') return;
      clearInlineMessage("workspace-message");
      setBasicInfoField('preacher', e.target.value);
      clearBasicInfoSavedState();
      mountAppShell('app');
    });
  }

  const churchNameInput = document.getElementById('church-name-input');
  if (churchNameInput) {
    churchNameInput.addEventListener('input', (e) => {
      if (appState?.source?.sourceStatus === 'RUNNING') return;
      clearInlineMessage("workspace-message");
      setBasicInfoField('churchName', e.target.value);
      clearBasicInfoSavedState();
      mountAppShell('app');
    });
  }

  const sermonDateInput = document.getElementById('sermon-date-input');
  if (sermonDateInput) {
    sermonDateInput.addEventListener('input', (e) => {
      if (appState?.source?.sourceStatus === 'RUNNING') return;
      clearInlineMessage("workspace-message");
      setBasicInfoField('sermonDate', e.target.value);
      clearBasicInfoSavedState();
      mountAppShell('app');
    });
  }

  const saveBasicInfoBtn = document.getElementById('save-basic-info-btn');
  if (saveBasicInfoBtn) {
    saveBasicInfoBtn.addEventListener('click', () => {
      if (appState?.source?.sourceStatus !== 'COMPLETED') return;

      clearInlineMessage("workspace-message");
      saveBasicInfoDraft();
      mountAppShell('app');
      showToast('기본 정보가 저장되었습니다.', 'success');
    });
  }
  
  const audioFileSelectBtn = document.getElementById('audio-file-select-btn');
  if (audioFileSelectBtn) {
    audioFileSelectBtn.addEventListener('click', async () => {
      if (appState?.source?.sourceStatus === 'RUNNING') return;

      clearInlineMessage("workspace-message");

      try {
        const filePath = await SelectAudioFile();
        if (!filePath) return;

        setSourceFilePath(filePath);
        updateQtPrepareStatus();
        mountAppShell('app');
      } catch (error) {
        console.error(error);
        setInlineMessage("workspace-message", '오디오 파일 선택 중 오류가 발생했습니다.', "error");
      }
    });
  }

  const textFileSelectBtn = document.getElementById('text-file-select-btn');
  if (textFileSelectBtn) {
    textFileSelectBtn.addEventListener('click', async () => {
      if (appState?.source?.sourceStatus === 'RUNNING') return;

      clearInlineMessage("workspace-message");

      try {
        const filePath = await SelectTextFile();
        if (!filePath) return;

        setSourceFilePath(filePath);

        const text = await LoadTextFile(filePath);
        if (text) {
          setRawText(text);
        }

        updateQtPrepareStatus();
        mountAppShell('app');
      } catch (error) {
        console.error(error);
        setInlineMessage("workspace-message", error?.message || '텍스트 파일 선택 중 오류가 발생했습니다.', "error");
      }
    });
  }

  const runSourceBtn = document.getElementById('run-source-btn');
  if (runSourceBtn) {
    runSourceBtn.addEventListener('click', () => {
      if (appState?.source?.sourceStatus === 'RUNNING') return;
      runQtPrepare();
    });
  }

  const runAudioBtn = document.getElementById('run-audio-btn');
  if (runAudioBtn) {
    runAudioBtn.addEventListener('click', () => {
      if (appState?.source?.sourceStatus === 'RUNNING') return;
      runQtPrepare();
    });
  }

  const runTextBtn = document.getElementById('run-text-btn');
  if (runTextBtn) {
    runTextBtn.addEventListener('click', () => {
      if (appState?.source?.sourceStatus === 'RUNNING') return;
      runQtPrepare();
    });
  }
}

function bindAudienceStepTabs() {
  const tabButtons = document.querySelectorAll('[data-audience-step][data-audience-id]');

  tabButtons.forEach((button) => {
    button.addEventListener('click', () => {
      const audienceId = button.dataset.audienceId;
      const stepId = button.dataset.audienceStep;

      if (!audienceId || !stepId) return;

      setAudienceStep(audienceId, stepId);
      mountAppShell('app');
    });
  });
}

function bindAudienceWorkspaceEvents() {
  bindAudienceStepTabs();

  const audienceId = appState.selectedMenu;
  const currentStep = appState.audienceSteps?.[audienceId] || 'step1';

  if (currentStep === 'step1') {
    bindQTStep1Events(audienceId);
    return;
  }

  if (currentStep === 'step2') {
    bindQTStep2Events(audienceId);
    return;
  }

  if (currentStep === 'step3') {
    bindQTStep3Events(audienceId);
    return;
  }
}

export function bindMainWorkspaceEvents() {
  if (appState.selectedMenu === 'qt_prepare') {
    bindQtPrepareEvents();
    return;
  }

  if (isAudienceMenu(appState.selectedMenu)) {
    bindAudienceWorkspaceEvents();
  }
}

export function renderMainWorkspace() {
  const menu = appState.selectedMenu;

  return `
    <main class="main-workspace">
      ${menu === 'qt_prepare' ? renderQtPrepareLayout() : renderAudienceLayout(menu)}
    </main>
  `;
}