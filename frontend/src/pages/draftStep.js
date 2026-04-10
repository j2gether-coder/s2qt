import { appState, updateDraft, updateStatus } from '../state/appState';
import { refreshShellStatus } from '../components/appShell';

function escapeHtml(value) {
  return String(value ?? '')
    .replaceAll('&', '&amp;')
    .replaceAll('<', '&lt;')
    .replaceAll('>', '&gt;')
    .replaceAll('"', '&quot;')
    .replaceAll("'", '&#39;');
}

function getInputTypeLabel(inputType) {
  switch (inputType) {
    case 'v':
      return '동영상';
    case 'a':
      return '음성';
    case 't':
      return '텍스트';
    default:
      return '-';
  }
}

function hasApiKey() {
  return !!(
    appState.apiKey?.trim() ||
    appState.settings?.apiKey?.trim()
  );
}

function buildDraftReady() {
  const ready = !!(
    appState.draft.promptText?.trim() &&
    appState.draft.draftHtml?.trim()
  );

  updateStatus({ draftReady: ready });
  refreshShellStatus();

  const statusEl = document.getElementById('draftStepStatus');
  if (statusEl) {
    statusEl.textContent = ready
      ? 'LLM 결과가 준비되었습니다. 다음 단계로 진행할 수 있습니다.'
      : '프롬프트를 생성하고 LLM 결과를 입력해 주세요.';
  }
}

function renderMetaSummary() {
  const { meta } = appState;

  return `
    <div class="fileitem"><strong>제목:</strong> ${escapeHtml(meta.title || '-')}</div>
    <div class="fileitem"><strong>본문 성구:</strong> ${escapeHtml(meta.bibleText || '-')}</div>
    <div class="fileitem"><strong>찬송가:</strong> ${escapeHtml(meta.hymn || '-')}</div>
    <div class="fileitem"><strong>설교자:</strong> ${escapeHtml(meta.preacher || '-')}</div>
    <div class="fileitem"><strong>교회명:</strong> ${escapeHtml(meta.churchName || '-')}</div>
    <div class="fileitem"><strong>입력 유형:</strong> ${escapeHtml(getInputTypeLabel(meta.inputType || ''))}</div>
  `;
}

async function handleBuildPrompt() {
  if (!appState.meta.transcriptText?.trim()) {
    const statusEl = document.getElementById('draftStepStatus');
    if (statusEl) {
      statusEl.textContent = '전사문이 없습니다. Step1에서 원본 처리 후 다시 시도해 주세요.';
    }
    return;
  }

  try {
    const prompt = await window.go.main.App.BuildQTPromptPreview({
      title: appState.meta.title || '',
      bibleText: appState.meta.bibleText || '',
      hymn: appState.meta.hymn || '',
      preacher: appState.meta.preacher || '',
      churchName: appState.meta.churchName || '',
      sermonDate: appState.meta.sermonDate || '',
      sourceUrl: appState.meta.sourceUrl || '',
      transcript: appState.meta.transcriptText || ''
    });

    updateDraft({ promptText: prompt });

    const promptEl = document.getElementById('promptPreview');
    if (promptEl) {
      promptEl.value = prompt;
    }

    buildDraftReady();
  } catch (error) {
    const statusEl = document.getElementById('draftStepStatus');
    if (statusEl) {
      statusEl.textContent = `프롬프트 생성 중 오류가 발생했습니다: ${error?.message || error}`;
    }
  }
}

function handleCopyPrompt() {
  const prompt = appState.draft.promptText?.trim();
  const resultGuideEl = document.getElementById('llmResultGuide');

  if (!prompt) {
    if (resultGuideEl) {
      resultGuideEl.textContent = '복사할 프롬프트가 없습니다. 먼저 프롬프트를 생성해 주세요.';
    }
    return;
  }

  navigator.clipboard.writeText(prompt)
    .then(() => {
      if (resultGuideEl) {
        resultGuideEl.textContent = '프롬프트를 클립보드에 복사했습니다. 외부 LLM에 붙여넣고 결과를 아래에 입력해 주세요.';
      }
    })
    .catch(() => {
      if (resultGuideEl) {
        resultGuideEl.textContent = '프롬프트 복사에 실패했습니다.';
      }
    });
}

function handleRunLlm() {
  const resultGuideEl = document.getElementById('llmResultGuide');

  if (!appState.draft.promptText?.trim()) {
    if (resultGuideEl) {
      resultGuideEl.textContent = '먼저 프롬프트를 생성해 주세요.';
    }
    return;
  }

  if (resultGuideEl) {
    resultGuideEl.textContent = 'LLM API 연동은 추후 연결 예정입니다. 현재는 수동 입력 방식으로 진행해 주세요.';
  }
}

export function renderDraftStep() {
  const apiEnabled = hasApiKey();

  return `
    <section class="card">
      <div class="hint">
        Step1 정보를 바탕으로 프롬프트를 생성하고, LLM 응답 결과를 확보합니다.
      </div>
      <div class="status" id="draftStepStatus">
        ${
          appState.status.draftReady
            ? 'LLM 결과가 준비되었습니다. 다음 단계로 진행할 수 있습니다.'
            : '프롬프트를 생성하고 LLM 결과를 입력해 주세요.'
        }
      </div>
    </section>

    <section class="card">
      <h3 class="mini-title">입력 정보 요약</h3>
      ${renderMetaSummary()}
    </section>

    <section class="card">
      <h3 class="mini-title">프롬프트 생성</h3>

      <div class="row topgap-sm">
        <button id="buildPromptBtn" class="button" type="button">프롬프트 생성</button>
      </div>

      <label class="label topgap" for="promptPreview">프롬프트 결과</label>
      <textarea
        id="promptPreview"
        class="textarea promptbox"
        placeholder="여기에 프롬프트 생성 결과가 표시됩니다."
      >${escapeHtml(appState.draft.promptText || '')}</textarea>

      <div class="row topgap-sm">
        <button id="copyPromptBtn" class="button button-ghost" type="button">프롬프트 복사</button>
      </div>
    </section>

    <section class="card">
      <h3 class="mini-title">${apiEnabled ? 'LLM API 연동' : 'LLM 결과 입력'}</h3>

      ${
        apiEnabled
          ? `
            <div class="row topgap-sm">
              <button id="runLlmBtn" class="button button-success" type="button">LLM 호출</button>
            </div>
          `
          : ''
      }

      <div class="hint topgap" id="llmResultGuide">
        ${
          apiEnabled
            ? 'API KEY가 설정되어 있습니다. 추후 LLM API 연동 시 이 단계에서 결과를 자동으로 받아오게 됩니다.'
            : 'API KEY가 없어 Manual 방식으로 진행합니다. 위에서 프롬프트를 복사해 외부 LLM에 전달한 뒤 결과를 아래에 붙여넣어 주세요.'
        }
      </div>

      <label class="label topgap" for="draftHtmlPreview">LLM 응답 결과</label>
      <textarea
        id="draftHtmlPreview"
        class="textarea editorbox-lg"
        placeholder="${
          apiEnabled
            ? '여기에 LLM API 응답 결과가 표시됩니다.'
            : '여기에 ChatGPT 등에서 받은 결과를 붙여넣어 주세요.'
        }"
      >${escapeHtml(appState.draft.draftHtml || '')}</textarea>
    </section>
  `;
}

export function bindDraftStepEvents() {
  const buildPromptBtn = document.getElementById('buildPromptBtn');
  const copyPromptBtn = document.getElementById('copyPromptBtn');
  const runLlmBtn = document.getElementById('runLlmBtn');
  const promptPreview = document.getElementById('promptPreview');
  const draftHtmlPreview = document.getElementById('draftHtmlPreview');

  if (buildPromptBtn) {
    buildPromptBtn.addEventListener('click', handleBuildPrompt);
  }

  if (copyPromptBtn) {
    copyPromptBtn.addEventListener('click', handleCopyPrompt);
  }

  if (runLlmBtn) {
    runLlmBtn.addEventListener('click', handleRunLlm);
  }

  if (promptPreview) {
    promptPreview.addEventListener('input', (event) => {
      updateDraft({ promptText: event.target.value });
      buildDraftReady();
    });
  }

  if (draftHtmlPreview) {
    draftHtmlPreview.addEventListener('input', (event) => {
      updateDraft({ draftHtml: event.target.value });
      buildDraftReady();
    });
  }

  buildDraftReady();
}