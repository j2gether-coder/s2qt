import {
  LoadQTStep2Data,
  SaveQTStep2Data,
  OpenTempHTMLPreview,
  SaveHistory,
} from '../../../wailsjs/go/main/App';
import {
  appState,
  setAudienceStep,
  setAudienceStepStatus,
} from '../../state/appState';
import { mountAppShell } from '../appShell';
import { showToast, setInlineMessage, clearInlineMessage } from "../../common/uiMessage";

function setValue(id, value) {
  const el = document.getElementById(id);
  if (el) {
    el.value = value || '';
  }
}

function getValue(id) {
  return document.getElementById(id)?.value || '';
}

function toSupportScripturesArray(text) {
  return String(text || '')
    .split(/[,\r\n]+/)
    .map((v) => v.trim())
    .filter(Boolean);
}

function toSupportScripturesText(items) {
  if (!Array.isArray(items)) return '';
  return items
    .map((item) => String(item || '').trim())
    .filter(Boolean)
    .join(', ');
}

function syncReadonlyStep2Meta(title, bibleText, hymn) {
  const readonlyTitleEl = document.getElementById('qtReadonlyTitle');
  if (readonlyTitleEl) {
    readonlyTitleEl.textContent = (title || '').trim() || '-';
  }

  const readonlyBibleTextEl = document.getElementById('qtReadonlyBibleText');
  if (readonlyBibleTextEl) {
    readonlyBibleTextEl.textContent = (bibleText || '').trim() || '-';
  }

  const readonlyHymnEl = document.getElementById('qtReadonlyHymn');
  if (readonlyHymnEl) {
    readonlyHymnEl.textContent = (hymn || '').trim() || '-';
  }
}

function toSupportScripturesTextarea(value) {
  if (!Array.isArray(value)) {
    return '';
  }

  return value
    .map((v) => String(v || '').trim())
    .filter(Boolean)
    .join(', ');
}

function getBasicInfo() {
  return appState?.source?.basicInfo || {};
}

function ensureQTTitlePrefix(title) {
  const v = (title || '').trim();

  if (!v) {
    return '[QT]';
  }

  if (v.startsWith('[QT]')) {
    return v;
  }

  return `[QT] ${v}`;
}

function resolveQtTitleByAudience(audienceId, metaTitle, llmTitle) {
  const safeMetaTitle = (metaTitle || '').trim();
  const safeLlmTitle = (llmTitle || '').trim();

  if (audienceId === 'adult') {
    return safeMetaTitle || safeLlmTitle || '';
  }

  if (['young_adult', 'teen', 'child'].includes(audienceId)) {
    return safeLlmTitle || safeMetaTitle || '';
  }

  return safeMetaTitle || safeLlmTitle || '';
}

function getLoadedStep2TitleCandidate() {
  return appState?.qtStep2LoadedTitle || '';
}

function setLoadedStep2TitleCandidate(value) {
  appState.qtStep2LoadedTitle = (value || '').trim();
}

function getLoadedStep2Meta() {
  return appState?.qtStep2LoadedMeta || {};
}

function setLoadedStep2Meta(meta = {}) {
  appState.qtStep2LoadedMeta = {
    title: (meta.title || '').trim(),
    bibleText: (meta.bibleText || '').trim(),
    biblePassageText: meta.biblePassageText || '',
    hymn: (meta.hymn || '').trim(),
    preacher: (meta.preacher || '').trim(),
    churchName: (meta.churchName || '').trim(),
    sermonDate: (meta.sermonDate || '').trim(),
    sourceURL: (meta.sourceURL || '').trim(),
    supportScriptures: Array.isArray(meta.supportScriptures) ? meta.supportScriptures : [],
  };
}

function buildStep2Payload(audienceId) {
  const basicInfo = getBasicInfo();
  const loadedMeta = getLoadedStep2Meta();

  const inputTitle = getValue('title').trim();
  const loadedTitle = getLoadedStep2TitleCandidate();
  const fallbackTitle = basicInfo.title || '';

  const resolvedTitle = resolveQtTitleByAudience(
    audienceId,
    inputTitle || fallbackTitle,
    inputTitle || loadedTitle
  );

  const finalTitle = ensureQTTitlePrefix(resolvedTitle);

  return {
    audience: audienceId,

    title: finalTitle,
    bibleText: getValue('bibleText').trim() || (basicInfo.bibleText || loadedMeta.bibleText || '').trim(),
    bible_passage_text: getValue('biblePassageText'),
    hymn: getValue('hymn').trim() || (basicInfo.hymn || loadedMeta.hymn || '').trim(),
    preacher: (basicInfo.preacher || loadedMeta.preacher || '').trim(),
    churchName: (basicInfo.churchName || loadedMeta.churchName || '').trim(),
    sermonDate: (basicInfo.sermonDate || loadedMeta.sermonDate || '').trim(),
    sourceURL: (appState?.source?.sourceRef?.url || loadedMeta.sourceURL || '').trim(),
    support_scriptures: toSupportScripturesArray(getValue('supportScriptures')),

    summaryTitle: getValue('summaryTitle'),
    summaryBody: getValue('summaryBody'),

    messageTitle1: getValue('messageTitle1'),
    messageBody1: getValue('messageBody1'),

    messageTitle2: getValue('messageTitle2'),
    messageBody2: getValue('messageBody2'),

    messageTitle3: getValue('messageTitle3'),
    messageBody3: getValue('messageBody3'),

    reflectionItem1: getValue('reflectionItem1'),
    reflectionItem2: getValue('reflectionItem2'),
    reflectionItem3: getValue('reflectionItem3'),

    prayerTitle: getValue('prayerTitle'),
    prayerBody: getValue('prayerBody'),
  };
}

function buildHistoryPayload(audienceId, step2Payload) {
  const basicInfo = getBasicInfo();
  const loadedMeta = getLoadedStep2Meta();

  const historyTitle = ensureQTTitlePrefix(
    step2Payload?.title || basicInfo.title || getLoadedStep2TitleCandidate() || ''
  );

  return {
    title: historyTitle,
    bibleText: (step2Payload?.bibleText || basicInfo.bibleText || loadedMeta.bibleText || '').trim(),
    hymn: (step2Payload?.hymn || basicInfo.hymn || loadedMeta.hymn || '').trim(),
    preacher: (step2Payload?.preacher || basicInfo.preacher || loadedMeta.preacher || '').trim(),
    churchName: (step2Payload?.churchName || basicInfo.churchName || loadedMeta.churchName || '').trim(),
    sermonDate: (step2Payload?.sermonDate || basicInfo.sermonDate || loadedMeta.sermonDate || '').trim(),
    audience: audienceId,
    qtResultJson: JSON.stringify(step2Payload),
  };
}

async function loadStep2Data(audienceId) {
  const data = await LoadQTStep2Data();
  const basicInfo = getBasicInfo();
  const loadedTitle =
    data?.title ||
    data?.meta?.title ||
    '';

  setLoadedStep2TitleCandidate(loadedTitle);

  const resolvedTitle = resolveQtTitleByAudience(
    audienceId,
    basicInfo.title || '',
    loadedTitle
  );

  const finalTitle = ensureQTTitlePrefix(resolvedTitle || loadedTitle || '');

  const finalBibleText = (data?.bibleText || basicInfo.bibleText || '').trim();
  const finalHymn = (data?.hymn || basicInfo.hymn || '').trim();

  syncReadonlyStep2Meta(finalTitle, finalBibleText, finalHymn);

  setLoadedStep2Meta({
    title: data?.title || '',
    bibleText: data?.bibleText || '',
    biblePassageText: data?.bible_passage_text || '',
    hymn: data?.hymn || '',
    preacher: data?.preacher || '',
    churchName: data?.churchName || '',
    sermonDate: data?.sermonDate || '',
    sourceURL: data?.sourceURL || '',
    supportScriptures: data?.support_scriptures || [],
  });

  setValue('title', finalTitle);
  setValue('bibleText', finalBibleText);
  setValue('hymn', finalHymn);
  setValue('biblePassageText', data?.bible_passage_text || '');
  setValue('supportScriptures', toSupportScripturesText(data?.support_scriptures));

  setValue('summaryTitle', data?.summaryTitle);
  setValue('summaryBody', data?.summaryBody);

  setValue('messageTitle1', data?.messageTitle1);
  setValue('messageBody1', data?.messageBody1);

  setValue('messageTitle2', data?.messageTitle2);
  setValue('messageBody2', data?.messageBody2);

  setValue('messageTitle3', data?.messageTitle3);
  setValue('messageBody3', data?.messageBody3);

  setValue('reflectionItem1', data?.reflectionItem1);
  setValue('reflectionItem2', data?.reflectionItem2);
  setValue('reflectionItem3', data?.reflectionItem3);

  setValue('prayerTitle', data?.prayerTitle);
  setValue('prayerBody', data?.prayerBody);
}

function bindStep2MetaPreviewSync() {
  const titleEl = document.getElementById('title');
  const bibleTextEl = document.getElementById('bibleText');
  const hymnEl = document.getElementById('hymn');

  const sync = () => {
    syncReadonlyStep2Meta(
      getValue('title'),
      getValue('bibleText'),
      getValue('hymn')
    );
  };

  if (titleEl) titleEl.addEventListener('input', sync);
  if (bibleTextEl) bibleTextEl.addEventListener('input', sync);
  if (hymnEl) hymnEl.addEventListener('input', sync);
}

export async function bindQTStep2Events(audienceId) {
  clearInlineMessage("qt-step2-message");

  const saveBtn = document.getElementById('saveQtJsonBtn');
  const previewBtn = document.getElementById('previewHtmlBtn');
  const backBtn = document.getElementById('backToStep1Btn');
  const nextBtn = document.getElementById('goStep3Btn');

  if (previewBtn) {
    previewBtn.disabled = true;
  }

  try {
    await loadStep2Data(audienceId);

    if (previewBtn) {
      previewBtn.disabled = false;
    }
  } catch (error) {
    console.error(error);
    setInlineMessage(
      "qt-step2-message",
      'Step2 데이터를 불러오는 중 오류가 발생했습니다.',
      "error"
    );
  }

  if (saveBtn) {
    saveBtn.addEventListener('click', async () => {
      clearInlineMessage("qt-step2-message");

      try {
        setAudienceStepStatus(audienceId, 'step2', 'running');

        const step2Payload = buildStep2Payload(audienceId);

        // SaveQTStep2Data 내부에서 temp.json + temp.html 처리
        await SaveQTStep2Data(step2Payload);

        const historyReq = buildHistoryPayload(audienceId, step2Payload);
        const historyId = await SaveHistory(historyReq);

        appState.historySelected = {
          historyId,
          audienceId,
          step1ResultJson: historyReq.qtResultJson || '',
        };

        setAudienceStepStatus(audienceId, 'step2', 'done');

        if (nextBtn) {
          nextBtn.disabled = false;
        }

        if (previewBtn) {
          previewBtn.disabled = false;
        }

        showToast('Step2 내용을 저장했습니다.', 'success');
      } catch (error) {
        console.error(error);
        setAudienceStepStatus(audienceId, 'step2', 'error');
        setInlineMessage(
          "qt-step2-message",
          error?.message || 'Step2 저장 중 오류가 발생했습니다.',
          "error"
        );
      }
    });
  }

  if (previewBtn) {
    previewBtn.addEventListener('click', async () => {
      clearInlineMessage("qt-step2-message");

      try {
        await OpenTempHTMLPreview();
        showToast('저장된 미리보기를 열었습니다.', 'success');
      } catch (error) {
        console.error(error);
        setInlineMessage(
          "qt-step2-message",
          '미리보기를 열 수 없습니다. 먼저 저장해 주세요.',
          "error"
        );
      }
    });
  }

  if (backBtn) {
    backBtn.addEventListener('click', () => {
      setAudienceStep(audienceId, 'step1');
      mountAppShell('app');
    });
  }

  if (nextBtn) {
    nextBtn.addEventListener('click', () => {
      if (nextBtn.disabled) return;

      setAudienceStep(audienceId, 'step3');
      mountAppShell('app');
    });
  }
}
