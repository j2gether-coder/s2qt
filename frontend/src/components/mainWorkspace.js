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
  setSourceProgress,
  clearSourceProgress,
  setAudienceStep,
} from '../state/appState';
import { EventsOn } from '../../wailsjs/runtime/runtime';
import { mountAppShell } from './appShell';
import { showToast, setInlineMessage, clearInlineMessage } from "../common/uiMessage";
import {
  SelectAudioFile,
  SelectTextFile,
  LoadTextFile,
  RunSourcePrepare,
  GetVideoMeta,
  PrepareRuntimeForInput,
} from '../../wailsjs/go/main/App';
import { renderQTStep1 } from './qt/qtStep1';
import { renderQTStep2 } from './qt/qtStep2';
import { renderQTStep3 } from './qt/qtStep3';
import { bindQTStep1Events } from './qt/bindQTStep1';
import { bindQTStep2Events } from './qt/bindQTStep2';
import { bindQTStep3Events } from './qt/bindQTStep3';
import { renderAppSettings, bindAppSettingsEvents } from './settings/appSettings';
import { renderHistoryWorkspace, bindHistoryWorkspaceEvents } from './history/historyWorkspace';

function isSettingsMenu(menu) {
  return menu === 'settings';
}

function isHistoryMenu(menu) {
  return menu === 'history';
}

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

// 헤더 우측 상태 라벨 텍스트. 실행 중에는 단계/진행률 메시지를 덧붙인다.
// (메시지는 백엔드가 생성하는 "전사 중 45%" 형태의 평문이라 별도 이스케이프가 불필요하다.)
function getQtPrepareStatusText(status) {
  const base = getSourceStatusLabel(status);
  if (status !== 'RUNNING') return base;

  const msg = (appState?.source?.progressMessage || '').trim();
  return msg ? `${base} · ${msg}` : base;
}

// 진행 이벤트마다 전체 화면을 다시 그리지 않고 헤더 상태 라벨만 갱신한다.
function updateRunningStatusLabel() {
  if (appState?.source?.sourceStatus !== 'RUNNING') return;

  const el = document.querySelector('.workspace-header-status');
  if (el) {
    el.textContent = getQtPrepareStatusText('RUNNING');
  }
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

  // yt-dlp가 돌려준 표준 URL(재생목록 파라미터가 제거된 영상 단일 URL)로 입력값을
  // 정리한다. 다운로드 대상과 메타의 source_url에만 쓰이며,
  // 하단 출처 링크/QR은 footer 설정(교회 홈페이지 / 기본 주소)을 그대로 사용한다.
  const canonicalUrl = (meta.webpageUrl || '').trim();
  const effectiveUrl = canonicalUrl || url;
  if (effectiveUrl !== url) {
    setSourceUrl(effectiveUrl);
  }

  const basicInfo = appState?.source?.basicInfo || {};
  // URL이 이전에 메타정보를 가져온 URL과 다르면(=새 영상이면) 메타 파생 필드를
  // 새 값으로 덮어쓴다. 같은 URL을 다시 실행할 때는 비어 있는 필드만 채워
  // 사용자가 직접 수정한 값을 보존한다.
  const isNewVideo = (appState?.source?.basicInfoMetaUrl || '') !== effectiveUrl;
  let changed = false;

  const applyMetaField = (field, value) => {
    if (!(value || '').trim()) return;
    if (!isNewVideo && (basicInfo[field] || '').trim()) return;
    setBasicInfoField(field, value);
    changed = true;
  };

  applyMetaField('title', meta.title);
  applyMetaField('sermonDate', meta.uploadDateText);
  applyMetaField('churchName', meta.channel);

  appState.source.basicInfoMetaUrl = effectiveUrl;

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

  let unsubscribeProgress = null;

  try {
    const sourceType = appState?.source?.sourceType || '';

    setSourceStatus('RUNNING');
    setSourceProgress('init', '준비 중...');

    // 백엔드 파이프라인의 단계/진행률 이벤트를 구독하여 헤더 상태 라벨에 반영한다.
    unsubscribeProgress = EventsOn('qt:prepare:progress', (payload) => {
      setSourceProgress(payload?.stage || '', payload?.message || '');
      updateRunningStatusLabel();
    });

    mountAppShell('app');

    const runtimeResult = await PrepareRuntimeForInput(sourceType);
    if (!runtimeResult?.ok) {
      throw new Error(runtimeResult?.message || '런타임 준비 중 오류가 발생했습니다.');
    }

    if (sourceType === 'video') {
      await enrichVideoBasicInfoFromMeta();
      mountAppShell('app');
    }

    const payload = buildSourcePreparePayload();
    const result = await RunSourcePrepare(payload);

    if (result?.rawText) {
      setRawText(result.rawText);
    }

    // 백엔드가 오류 대신 실패 결과만 돌려주는 경우에도 catch에서 상태를 되돌리도록 한다.
    if (!result?.success) {
      throw new Error(result?.message || "자료 처리에 실패했습니다.");
    }

    setSourceStatus(result?.status || 'COMPLETED');
    appState.source.lastSavedAt = new Date().toLocaleString();
    showToast("자료 준비가 완료되었습니다.", "success");

    mountAppShell('app');
  } catch (error) {
    console.error(error);
    // updateQtPrepareStatus()는 RUNNING이면 조기 반환하므로, 실패 시 먼저 RUNNING을
    // 해제해야 입력값 기준으로 상태가 다시 계산되고 실행 버튼이 재활성화된다.
    // (동영상 다운로드 404/403/JS 런타임 오류 등)
    setSourceStatus('NOT_READY');
    updateQtPrepareStatus();
    setInlineMessage("workspace-message", error?.message || "자료 처리 중 오류가 발생했습니다.", "error");
    mountAppShell('app');
  } finally {
    if (unsubscribeProgress) {
      unsubscribeProgress();
    }
    clearSourceProgress();
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

const WEEKDAY_LABELS = ['일', '월', '화', '수', '목', '금', '토'];

// formatDateWithWeekday는 "2026-09-05" → "2026-09-05(토)"로 만든다.
//
// <input type="date">의 표시 형식은 브라우저 로캘이 정하므로 요일을 넣을 수 없다.
// 그래서 입력란 옆에 이 문자열을 따로 보여 준다.
// 저장되는 값 자체는 "YYYY-MM-DD"를 유지한다 — 파일명·프롬프트·DB가 이 형식을 쓴다.
export function formatDateWithWeekday(isoDate) {
  const value = String(isoDate ?? '').trim();
  if (!/^\d{4}-\d{2}-\d{2}$/.test(value)) return '';

  const [year, month, day] = value.split('-').map(Number);
  const d = new Date(year, month - 1, day);

  // 존재하지 않는 날짜(예: 2026-02-31)는 다른 날로 굴러가므로 되돌려 확인한다.
  if (d.getFullYear() !== year || d.getMonth() !== month - 1 || d.getDate() !== day) {
    return '';
  }

  return `${value}(${WEEKDAY_LABELS[d.getDay()]})`;
}

function renderQtPrepareLayout() {
  const { basicInfo, sourceStatus } = appState.source;
  const isRunning = sourceStatus === 'RUNNING';
  const isBasicInfoSavable = sourceStatus === 'COMPLETED';

  return `
    <section class="workspace-panel">
      <div class="workspace-header-row">
        <div class="workspace-header-title">QT 자료 준비</div>
        <div class="workspace-header-status">${getQtPrepareStatusText(sourceStatus || 'NOT_READY')}</div>
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
                <label class="form-label">시리즈</label>
                <input type="text" id="series-input" value="${escapeHtml(basicInfo.series || '')}" placeholder="예: 본받고 싶은 교회(1)" ${isRunning ? 'disabled' : ''} />
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
                <div class="field-with-note">
                  <input type="date" id="sermon-date-input" value="${escapeHtml(basicInfo.sermonDate || '')}" ${isRunning ? 'disabled' : ''} />
                  <span class="field-inline-note" id="sermon-date-note">${escapeHtml(formatDateWithWeekday(basicInfo.sermonDate))}</span>
                </div>
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

  const seriesInput = document.getElementById('series-input');
  if (seriesInput) {
    seriesInput.addEventListener('input', (e) => {
      if (appState?.source?.sourceStatus === 'RUNNING') return;
      clearInlineMessage("workspace-message");
      setBasicInfoField('series', e.target.value);
      clearBasicInfoSavedState();
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
    });
  }

  const hymnInput = document.getElementById('hymn-input');
  if (hymnInput) {
    hymnInput.addEventListener('input', (e) => {
      if (appState?.source?.sourceStatus === 'RUNNING') return;
      clearInlineMessage("workspace-message");
      setBasicInfoField('hymn', e.target.value);
      clearBasicInfoSavedState();
    });
  }

  const preacherInput = document.getElementById('preacher-input');
  if (preacherInput) {
    preacherInput.addEventListener('input', (e) => {
      if (appState?.source?.sourceStatus === 'RUNNING') return;
      clearInlineMessage("workspace-message");
      setBasicInfoField('preacher', e.target.value);
      clearBasicInfoSavedState();
    });
  }

  const churchNameInput = document.getElementById('church-name-input');
  if (churchNameInput) {
    churchNameInput.addEventListener('input', (e) => {
      if (appState?.source?.sourceStatus === 'RUNNING') return;
      clearInlineMessage("workspace-message");
      setBasicInfoField('churchName', e.target.value);
      clearBasicInfoSavedState();
    });
  }

  const sermonDateInput = document.getElementById('sermon-date-input');
  if (sermonDateInput) {
    sermonDateInput.addEventListener('input', (e) => {
      if (appState?.source?.sourceStatus === 'RUNNING') return;
      clearInlineMessage("workspace-message");
      setBasicInfoField('sermonDate', e.target.value);

      // 요일 표기는 입력란이 직접 못 보여 주므로 옆의 문구를 갱신한다.
      const note = document.getElementById('sermon-date-note');
      if (note) {
        note.textContent = formatDateWithWeekday(e.target.value);
      }

      clearBasicInfoSavedState();
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

  if (isHistoryMenu(appState.selectedMenu)) {
    bindHistoryWorkspaceEvents();
    return;
  }

  if (isSettingsMenu(appState.selectedMenu)) {
    bindAppSettingsEvents();
    return;
  }

  if (isAudienceMenu(appState.selectedMenu)) {
    bindAudienceWorkspaceEvents();
  }
}

export function renderMainWorkspace() {
  const menu = appState.selectedMenu;

  let content = '';

  if (menu === 'qt_prepare') {
    content = renderQtPrepareLayout();
  } else if (isHistoryMenu(menu)) {
    content = renderHistoryWorkspace();
  } else if (isSettingsMenu(menu)) {
    content = renderAppSettings();
  } else if (isAudienceMenu(menu)) {
    content = renderAudienceLayout(menu);
  } else {
    content = renderQtPrepareLayout();
  }

  return `
    <main class="main-workspace">
      ${content}
    </main>
  `;
}