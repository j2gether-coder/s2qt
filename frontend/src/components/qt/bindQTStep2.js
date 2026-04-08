import {
  LoadQTStep2Data,
  SaveQTStep2Data,
  PreviewQTStep2HTML,
  OpenTempHTMLPreview,
} from '../../../wailsjs/go/main/App';

import {
  appState,
  setAudienceStep,
} from '../../state/appState';

import { mountAppShell } from '../appShell';

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

function buildStep2Payload(audienceId) {
  const basicInfo = getBasicInfo();
  const metaTitle = basicInfo.title || '';
  const llmTitle = getLoadedStep2TitleCandidate();
  const resolvedTitle = resolveQtTitleByAudience(audienceId, metaTitle, llmTitle);

  return {
    audience: audienceId,

    // audience 규칙에 따라 최종 제목 결정
    title: resolvedTitle,
    bibleText: basicInfo.bibleText || '',
    hymn: basicInfo.hymn || '',
    preacher: basicInfo.preacher || '',
    churchName: basicInfo.churchName || '',
    sermonDate: basicInfo.sermonDate || '',
    sourceURL: appState?.source?.sourceRef?.url || '',

    // Step2 편집값
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
  try {
    await loadStep2Data(audienceId);
  } catch (error) {
    console.error(error);
    window.alert(error?.message || 'Step2 데이터 불러오기 중 오류가 발생했습니다.');
  }

  const saveBtn = document.getElementById('saveQtJsonBtn');
  const previewBtn = document.getElementById('previewHtmlBtn');
  const backBtn = document.getElementById('backToStep1Btn');
  const nextBtn = document.getElementById('goStep3Btn');

  if (saveBtn) {
    saveBtn.addEventListener('click', async () => {
      try {
        const req = buildStep2Payload(audienceId);
        await SaveQTStep2Data(req);

        if (nextBtn) {
          nextBtn.disabled = false;
        }
      } catch (error) {
        console.error(error);
        window.alert(error?.message || 'Step2 저장 중 오류가 발생했습니다.');
      }
    });
  }

  if (previewBtn) {
    previewBtn.addEventListener('click', async () => {
      try {
        const req = buildStep2Payload(audienceId);
        await PreviewQTStep2HTML(req);
        await OpenTempHTMLPreview();

        if (nextBtn) {
          nextBtn.disabled = false;
        }
      } catch (error) {
        console.error(error);
        window.alert(error?.message || '미리보기 생성 중 오류가 발생했습니다.');
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