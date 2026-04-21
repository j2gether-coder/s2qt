import {
  LoadQTStep2Data,
  SaveQTStep2Data,
  PreviewQTStep2HTML,
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

function getBasicInfo() {
  return appState?.source?.basicInfo || {};
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
    hymn: (meta.hymn || '').trim(),
    preacher: (meta.preacher || '').trim(),
    churchName: (meta.churchName || '').trim(),
    sermonDate: (meta.sermonDate || '').trim(),
    sourceURL: (meta.sourceURL || '').trim(),
  };
}

function buildStep2Payload(audienceId) {
  const basicInfo = getBasicInfo();
  const loadedMeta = getLoadedStep2Meta();

  const metaTitle = basicInfo.title || '';
  const llmTitle = getLoadedStep2TitleCandidate();
  const resolvedTitle = resolveQtTitleByAudience(audienceId, metaTitle, llmTitle);

  return {
    audience: audienceId,

    title: resolvedTitle,
    bibleText: (basicInfo.bibleText || '').trim(),
    hymn: (basicInfo.hymn || loadedMeta.hymn || '').trim(),
    preacher: (basicInfo.preacher || loadedMeta.preacher || '').trim(),
    churchName: (basicInfo.churchName || loadedMeta.churchName || '').trim(),
    sermonDate: (basicInfo.sermonDate || loadedMeta.sermonDate || '').trim(),
    sourceURL: (appState?.source?.sourceRef?.url || loadedMeta.sourceURL || '').trim(),

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

  return {
    title: basicInfo.title || '',
    bibleText: basicInfo.bibleText || '',
    hymn: (basicInfo.hymn || loadedMeta.hymn || '').trim(),
    preacher: (basicInfo.preacher || loadedMeta.preacher || '').trim(),
    churchName: (basicInfo.churchName || loadedMeta.churchName || '').trim(),
    sermonDate: (basicInfo.sermonDate || loadedMeta.sermonDate || '').trim(),
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

  const readonlyTitleEl = document.getElementById('qtReadonlyTitle');
  if (readonlyTitleEl) {
    readonlyTitleEl.textContent = resolvedTitle || '-';
  }

  const readonlyBibleTextEl = document.getElementById('qtReadonlyBibleText');
  if (readonlyBibleTextEl) {
    readonlyBibleTextEl.textContent = basicInfo.bibleText || '-';
  }

  setLoadedStep2Meta({
    hymn: data?.hymn || '',
    preacher: data?.preacher || '',
    churchName: data?.churchName || '',
    sermonDate: data?.sermonDate || '',
    sourceURL: data?.sourceURL || '',
  });

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

export async function bindQTStep2Events(audienceId) {
  clearInlineMessage("qt-step2-message");

  try {
    await loadStep2Data(audienceId);
  } catch (error) {
    console.error(error);
    setInlineMessage(
      "qt-step2-message",
      error?.message || 'Step2 데이터 불러오기 중 오류가 발생했습니다.',
      "error"
    );
  }

  const saveBtn = document.getElementById('saveQtJsonBtn');
  const previewBtn = document.getElementById('previewHtmlBtn');
  const backBtn = document.getElementById('backToStep1Btn');
  const nextBtn = document.getElementById('goStep3Btn');

  if (saveBtn) {
    saveBtn.addEventListener('click', async () => {
      clearInlineMessage("qt-step2-message");

      try {
        setAudienceStepStatus(audienceId, 'step2', 'running');

        const step2Payload = buildStep2Payload(audienceId);
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
        const req = buildStep2Payload(audienceId);
        await PreviewQTStep2HTML(req);
        await OpenTempHTMLPreview();

        if (nextBtn) {
          nextBtn.disabled = false;
        }

        showToast('미리보기를 생성했습니다.', 'success');
      } catch (error) {
        console.error(error);
        setAudienceStepStatus(audienceId, 'step2', 'error');
        setInlineMessage(
          "qt-step2-message",
          error?.message || '미리보기 생성 중 오류가 발생했습니다.',
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
