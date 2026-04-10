import { appState, updateMeta, updateStatus, resetDraftState } from '../state/appState';
import { refreshShellStatus } from '../components/appShell';

function escapeHtml(value) {
  return String(value ?? '')
    .replaceAll('&', '&amp;')
    .replaceAll('<', '&lt;')
    .replaceAll('>', '&gt;')
    .replaceAll('"', '&quot;')
    .replaceAll("'", '&#39;');
}

function isInputReady(meta) {
  const hasCommonRequired = !!(
    meta.title?.trim() &&
    meta.bibleText?.trim() &&
    meta.inputType?.trim()
  );

  if (!hasCommonRequired) {
    return false;
  }

  switch (meta.inputType) {
    case 'v':
    case 'a':
      return !!meta.sourceUrl?.trim();
    case 't':
      return !!meta.transcriptText?.trim();
    default:
      return false;
  }
}

function getInputReadyMessage() {
  if (isInputReady(appState.meta)) {
    return '입력이 완료되었습니다. 다음 단계로 진행할 수 있습니다.';
  }

  switch (appState.meta.inputType) {
    case 'v':
      return '동영상 URL과 필수 항목을 입력해 주세요.';
    case 'a':
      return '음성 파일 정보와 필수 항목을 입력해 주세요.';
    case 't':
      return '원문 텍스트와 필수 항목을 입력해 주세요.';
    default:
      return '입력 유형과 필수 항목을 먼저 입력해 주세요.';
  }
}

function syncInputReady() {
  const ready = isInputReady(appState.meta);
  updateStatus({ inputReady: ready });
  refreshShellStatus();

  const inputStatusEl = document.getElementById('inputStepStatus');
  if (inputStatusEl) {
    inputStatusEl.textContent = getInputReadyMessage();
  }
}

function setExecutionStatus(message) {
  const statusEl = document.getElementById('sourceExecutionStatus');
  if (statusEl) {
    statusEl.textContent = message;
  }
}

function setSourceButtonsDisabled(disabled) {
  const executeSourceBtn = document.getElementById('executeSourceBtn');
  const browseFileBtn = document.getElementById('browseFileBtn');
  const browseFileBtnText = document.getElementById('browseFileBtnText');

  if (executeSourceBtn) executeSourceBtn.disabled = disabled;
  if (browseFileBtn) browseFileBtn.disabled = disabled;
  if (browseFileBtnText) browseFileBtnText.disabled = disabled;
}

function renderSourceActionButtons(inputType) {
  if (inputType === 'v') {
    return `
      <button id="executeSourceBtn" class="button" type="button">실행</button>
    `;
  }

  if (inputType === 'a') {
    return `
      <button id="browseFileBtn" class="button button-ghost" type="button">파일 탐색기</button>
      <button id="executeSourceBtn" class="button" type="button">실행</button>
    `;
  }

  return '';
}

function syncInputTypeUI() {
  const { inputType } = appState.meta;

  const sourceFieldEl = document.getElementById('sourceField');
  const transcriptFieldEl = document.getElementById('transcriptField');
  const sourceLabelEl = document.getElementById('sourceLabel');
  const sourceInputEl = document.getElementById('sourceUrl');
  const actionAreaEl = document.getElementById('sourceActionArea');
  const sourceActionButtonsEl = document.getElementById('sourceActionButtons');

  if (sourceFieldEl) {
    sourceFieldEl.style.display = inputType === 't' ? 'none' : 'block';
  }

  if (transcriptFieldEl) {
    transcriptFieldEl.style.display = inputType === 't' ? 'block' : 'none';
  }

  if (sourceLabelEl) {
    sourceLabelEl.textContent = inputType === 'v' ? '동영상 URL' : '음성 파일';
  }

  if (sourceInputEl) {
    sourceInputEl.placeholder =
      inputType === 'v'
        ? '동영상 URL을 입력해 주세요.'
        : '음성 파일 경로 또는 원본 식별값을 입력해 주세요.';
  }

  if (actionAreaEl) {
    actionAreaEl.style.display = inputType === 't' ? 'none' : 'block';
  }

  if (sourceActionButtonsEl) {
    sourceActionButtonsEl.innerHTML = renderSourceActionButtons(inputType);
    bindSourceActionEvents();
  }
}

function bindMetaInput(id, key) {
  const el = document.getElementById(id);
  if (!el) return;

  el.addEventListener('input', (event) => {
    updateMeta({ [key]: event.target.value });
    resetDraftState();
    syncInputReady();
  });
}

function bindInputTypeChange() {
  const radios = document.querySelectorAll('input[name="inputType"]');
  radios.forEach((radio) => {
    radio.addEventListener('change', (event) => {
      updateMeta({ inputType: event.target.value });
      resetDraftState();
      syncInputTypeUI();
      syncInputReady();
    });
  });
}

async function handleExecuteSource() {
  const { inputType, sourceUrl } = appState.meta;

  if (inputType === 'v') {
    if (!sourceUrl?.trim()) {
      setExecutionStatus('동영상 URL을 먼저 입력해 주세요.');
      return;
    }

    try {
      setSourceButtonsDisabled(true);
      setExecutionStatus('동영상 메타정보 조회 중입니다...');

      const metaResult = await window.go.main.App.FetchVideoMeta(sourceUrl);

      if (!metaResult?.title && metaResult?.success === false) {
        setExecutionStatus(metaResult?.message || '메타정보 조회에 실패했습니다.');
        return;
      }

      updateMeta({
        sourceUrl: metaResult?.sourceUrl || sourceUrl,
        title: metaResult?.title || appState.meta.title,
        // preacher는 자동 채움보다 사용자 입력 유지 권장
        // churchName도 자동 채움보다 사용자 입력 유지
      });

      const titleEl = document.getElementById('title');
      if (titleEl && metaResult?.title && !titleEl.value.trim()) {
        titleEl.value = metaResult.title;
      }

      setExecutionStatus('전사 실행 중입니다. 잠시만 기다려 주세요...');

      const pipelineResult = await window.go.main.App.RunVideoPipeline(sourceUrl);

      if (!pipelineResult?.success) {
        setExecutionStatus(pipelineResult?.message || '동영상 전사 실행에 실패했습니다.');
        return;
      }

      updateMeta({
        transcriptText: pipelineResult.transcriptText || ''
      });

      setExecutionStatus(
        pipelineResult?.message ||
        '동영상 처리 및 전사가 완료되었습니다.'
      );

      syncInputReady();
      return;
    } catch (error) {
      setExecutionStatus(`동영상 처리 중 오류가 발생했습니다: ${error?.message || error}`);
      return;
    } finally {
      setSourceButtonsDisabled(false);
    }
  }

  if (inputType === 'a') {
    if (!sourceUrl?.trim()) {
      setExecutionStatus('음성 파일 정보를 먼저 입력해 주세요.');
      return;
    }

    setExecutionStatus('음성 처리 서비스 연결 예정입니다.');
    return;
  }

  setExecutionStatus('텍스트 입력 유형은 별도 실행 단계가 없습니다.');
}

function handleFileBrowse() {
  const { inputType } = appState.meta;

  if (inputType === 'a') {
    setExecutionStatus('파일 탐색기로 음성 파일을 선택하는 기능이 연결될 예정입니다.');
    return;
  }

  if (inputType === 't') {
    setExecutionStatus('파일 탐색기로 텍스트 파일을 선택하는 기능이 연결될 예정입니다.');
    return;
  }

  setExecutionStatus('현재 입력 유형에서는 파일 탐색기를 사용하지 않습니다.');
}

function handleSourceEnter(event) {
  if (event.key !== 'Enter') return;
  event.preventDefault();
  handleExecuteSource();
}

function bindSourceActionEvents() {
  const executeSourceBtn = document.getElementById('executeSourceBtn');
  if (executeSourceBtn) {
    executeSourceBtn.addEventListener('click', handleExecuteSource);
  }

  const browseFileBtn = document.getElementById('browseFileBtn');
  if (browseFileBtn) {
    browseFileBtn.addEventListener('click', handleFileBrowse);
  }

  const browseFileBtnText = document.getElementById('browseFileBtnText');
  if (browseFileBtnText) {
    browseFileBtnText.addEventListener('click', handleFileBrowse);
  }
}

export function renderInputStep() {
  const { meta } = appState;

  return `
    <section class="card">
      <h3 class="mini-title">입력 유형 및 원본 자료</h3>

      <div class="input-type-inline" role="radiogroup" aria-label="입력 유형 선택">
        <label class="input-type-inline-item">
          <input type="radio" name="inputType" value="v" ${meta.inputType === 'v' ? 'checked' : ''}>
          <span>동영상</span>
        </label>

        <label class="input-type-inline-item">
          <input type="radio" name="inputType" value="a" ${meta.inputType === 'a' ? 'checked' : ''}>
          <span>음성</span>
        </label>

        <label class="input-type-inline-item">
          <input type="radio" name="inputType" value="t" ${meta.inputType === 't' ? 'checked' : ''}>
          <span>텍스트</span>
        </label>
      </div>

      <div class="status" id="inputStepStatus">${escapeHtml(getInputReadyMessage())}</div>

      <div id="sourceField" class="topgap" style="${meta.inputType === 't' ? 'display:none;' : ''}">
        <label class="label" id="sourceLabel" for="sourceUrl">
          ${meta.inputType === 'v' ? '동영상 URL' : '음성 파일'}
        </label>
        <input
          id="sourceUrl"
          class="input"
          type="text"
          value="${escapeHtml(meta.sourceUrl || '')}"
          placeholder="${meta.inputType === 'v' ? '동영상 URL을 입력해 주세요.' : '음성 파일 경로 또는 원본 식별값을 입력해 주세요.'}"
        />
      </div>

      <div id="sourceActionArea" class="topgap" style="${meta.inputType === 't' ? 'display:none;' : ''}">
        <div class="row" id="sourceActionButtons">
          ${renderSourceActionButtons(meta.inputType)}
        </div>

        <div class="status" id="sourceExecutionStatus">입력 유형에 맞는 원본 작업을 수행해 주세요.</div>
      </div>

      <div id="transcriptField" class="topgap" style="${meta.inputType === 't' ? 'display:block;' : 'display:none;'}">
        <label class="label" for="transcriptText">원문 텍스트</label>

        <div class="row topgap-sm">
          <button id="browseFileBtnText" class="button button-ghost" type="button">파일 탐색기</button>
        </div>

        <textarea
          id="transcriptText"
          class="textarea editorbox-md topgap-sm"
          placeholder="여기에 원문 텍스트를 입력하거나 붙여넣기 해주세요."
        >${escapeHtml(meta.transcriptText || '')}</textarea>

        <div class="hint">텍스트 입력 유형은 원문을 직접 붙여넣거나 파일에서 불러와 사용할 수 있습니다.</div>
      </div>
    </section>

    <section class="card">
      <h3 class="mini-title">기본 정보 입력</h3>

      <div class="input-meta-grid">
        <div class="field">
          <label class="label" for="title">제목 <span style="color:#dc2626;">*</span></label>
          <input
            id="title"
            class="input"
            type="text"
            value="${escapeHtml(meta.title || '')}"
            placeholder="제목을 입력해 주세요."
          />
        </div>

        <div class="field">
          <label class="label" for="bibleText">본문 성구 <span style="color:#dc2626;">*</span></label>
          <input
            id="bibleText"
            class="input"
            type="text"
            value="${escapeHtml(meta.bibleText || '')}"
            placeholder="예: 시 1:1"
          />
        </div>

        <div class="field">
          <label class="label" for="hymn">찬송가</label>
          <input
            id="hymn"
            class="input"
            type="text"
            value="${escapeHtml(meta.hymn || '')}"
            placeholder="예: 1, 선택입력"
          />
        </div>

        <div class="field">
          <label class="label" for="preacher">설교자</label>
          <input
            id="preacher"
            class="input"
            type="text"
            value="${escapeHtml(meta.preacher || '')}"
            placeholder="선택 입력"
          />
        </div>

        <div class="field">
          <label class="label" for="churchName">교회명</label>
          <input
            id="churchName"
            class="input"
            type="text"
            value="${escapeHtml(meta.churchName || '')}"
            placeholder="선택 입력"
          />
        </div>
      </div>

      <div class="hint topgap">
        제목과 본문 성구는 필수입니다. 설교자와 교회명은 필요 시 입력합니다.
      </div>
    </section>
  `;
}

export function bindInputStepEvents() {
  bindInputTypeChange();

  bindMetaInput('sourceUrl', 'sourceUrl');
  bindMetaInput('transcriptText', 'transcriptText');
  bindMetaInput('title', 'title');
  bindMetaInput('bibleText', 'bibleText');
  bindMetaInput('hymn', 'hymn');
  bindMetaInput('preacher', 'preacher');
  bindMetaInput('churchName', 'churchName');

  const sourceUrlEl = document.getElementById('sourceUrl');
  if (sourceUrlEl) {
    sourceUrlEl.addEventListener('keydown', handleSourceEnter);
  }

  bindSourceActionEvents();
  syncInputTypeUI();
  syncInputReady();
}