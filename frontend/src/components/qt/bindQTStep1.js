import {
  BuildQTPrompt,
  SaveManualLLMResult,
} from '../../../wailsjs/go/main/App';
import {
  appState,
  setAudienceStep,
  setAudienceStepStatus,
} from '../../state/appState';
import { mountAppShell } from '../appShell';
import { showToast, setInlineMessage, clearInlineMessage } from "../../common/uiMessage";

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
      clearInlineMessage("qt-step1-message");

      try {
        setAudienceStepStatus(audienceId, 'step1', 'running');

        const req = buildLLMRequest(audienceId);
        const prompt = await BuildQTPrompt(req);

        if (promptBox) {
          promptBox.value = prompt || '';
        }

        setAudienceStepStatus(audienceId, 'step1', 'idle');
        showToast('프롬프트를 생성했습니다.', 'success');
      } catch (error) {
        console.error(error);
        setAudienceStepStatus(audienceId, 'step1', 'error');
        setInlineMessage("qt-step1-message", error?.message || '프롬프트 생성 중 오류가 발생했습니다.', "error");
      }
    });
  }

  if (copyPromptBtn) {
    copyPromptBtn.addEventListener('click', async () => {
      clearInlineMessage("qt-step1-message");

      try {
        const text = promptBox?.value || '';
        if (!text.trim()) {
          setInlineMessage("qt-step1-message", '복사할 프롬프트가 없습니다.', "warning");
          return;
        }

        await navigator.clipboard.writeText(text);
        showToast('프롬프트를 복사했습니다.', 'success');
      } catch (error) {
        console.error(error);
        setInlineMessage("qt-step1-message", '프롬프트 복사 중 오류가 발생했습니다.', "error");
      }
    });
  }

  if (saveResultBtn) {
    saveResultBtn.addEventListener('click', async () => {
      clearInlineMessage("qt-step1-message");

      try {
        const jsonText = resultBox?.value || '';

        if (!jsonText.trim()) {
          setInlineMessage("qt-step1-message", '저장할 결과가 없습니다.', "warning");
          return;
        }

        setAudienceStepStatus(audienceId, 'step1', 'running');
        await SaveManualLLMResult(jsonText);
        setAudienceStepStatus(audienceId, 'step1', 'done');

        if (goStep2Btn) {
          goStep2Btn.disabled = false;
        }

        showToast('결과를 저장했습니다.', 'success');
      } catch (error) {
        console.error(error);
        setAudienceStepStatus(audienceId, 'step1', 'error');
        setInlineMessage("qt-step1-message", error?.message || '결과 저장 중 오류가 발생했습니다.', "error");
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