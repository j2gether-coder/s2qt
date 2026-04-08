import {
  BuildQTPrompt,
  SaveManualLLMResult,
} from '../../../wailsjs/go/main/App';

import {
  appState,
  setAudienceStep,
} from '../../state/appState';

import { mountAppShell } from '../appShell';

function buildLLMRequest(audienceId) {
  const info = appState?.source?.basicInfo || {};
  const sourceRef = appState?.source?.sourceRef || {};

  return {
    audience: audienceId,
    title: info.title || '',
    bibleText: info.bibleText || '',
    hymn: info.hymn || '',
    preacher: info.preacher || '',
    churchName: info.churchName || '',
    sermonDate: info.sermonDate || '',
    sourceURL: sourceRef.url || '',
  };
}

export function bindQTStep1Events(audienceId) {
  const buildPromptBtn = document.getElementById('buildPromptBtn');
  const promptBox = document.getElementById('qtPromptPreview');
  const copyPromptBtn = document.getElementById('copyPromptBtn');
  const resultBox = document.getElementById('qtResultText');
  const saveResultBtn = document.getElementById('saveResultBtn');
  const goStep2Btn = document.getElementById('goStep2Btn');

  if (buildPromptBtn) {
    buildPromptBtn.addEventListener('click', async () => {
      try {
        const req = buildLLMRequest(audienceId);
        const prompt = await BuildQTPrompt(req);

        if (promptBox) {
          promptBox.value = prompt || '';
        }
      } catch (error) {
        console.error(error);
        window.alert(error?.message || '프롬프트 생성 중 오류가 발생했습니다.');
      }
    });
  }

  if (copyPromptBtn) {
    copyPromptBtn.addEventListener('click', async () => {
      try {
        const text = promptBox?.value || '';
        if (!text.trim()) {
          window.alert('복사할 프롬프트가 없습니다.');
          return;
        }

        await navigator.clipboard.writeText(text);
      } catch (error) {
        console.error(error);
        window.alert('프롬프트 복사 중 오류가 발생했습니다.');
      }
    });
  }

  if (saveResultBtn) {
    saveResultBtn.addEventListener('click', async () => {
      try {
        const jsonText = resultBox?.value || '';

        if (!jsonText.trim()) {
          window.alert('저장할 결과가 없습니다.');
          return;
        }

        await SaveManualLLMResult(jsonText);

        if (goStep2Btn) {
          goStep2Btn.disabled = false;
        }
      } catch (error) {
        console.error(error);
        window.alert(error?.message || '결과 저장 중 오류가 발생했습니다.');
      }
    });
  }

  if (goStep2Btn) {
    goStep2Btn.addEventListener('click', () => {
      if (goStep2Btn.disabled) return;

      setAudienceStep(audienceId, 'step2');
      mountAppShell('app');
    });
  }
}