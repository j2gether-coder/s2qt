import {
  BuildQTPromptPreview,
  SaveMarkdownAndMakePDF,
  GetQTPromptTemplate
} from '../../wailsjs/go/main/App';

export async function buildPrompt(meta) {
  return await BuildQTPromptPreview(meta);
}

export async function buildPromptTemplate() {
  return await GetQTPromptTemplate();
}

/**
 * 현재는 백엔드에 "초안 HTML 생성" 전용 함수가 없으므로
 * 우선 프론트에서 받은 HTML을 그대로 반환하거나,
 * 추후 Go 함수가 생기면 이 함수만 교체합니다.
 */
export async function buildDraftHtmlFromPrompt(promptText, fallbackHtml = '') {
  if (fallbackHtml?.trim()) {
    return fallbackHtml;
  }

  return `
<div class="qt-wrap">
  <div class="qt-main">
    <h1 class="qt-title">[QT] 초안</h1>

    <div class="qt-subbox">
      본문: <br>
      찬송:
    </div>

    <h2 class="qt-section-title">🌿 말씀의 창: 본문 요약</h2>
    <div class="qt-body">
      <p>여기에 초안이 생성됩니다.</p>
    </div>

    <h2 class="qt-section-title">✨ 오늘의 메시지</h2>
    <div class="qt-body">
      <p>${promptText ? '프롬프트 기반 초안 준비 완료' : '프롬프트를 먼저 생성해 주세요.'}</p>
    </div>

    <h2 class="qt-section-title">🔍 깊은 묵상과 적용</h2>
    <div class="qt-box qt-reflection">
      <ul class="qt-list">
        <li>적용 항목</li>
      </ul>
    </div>

    <div class="qt-box qt-prayer">
      <div class="qt-prayer-title">🙏 오늘의 기도</div>
      <div class="qt-body">
        <p>기도문 초안</p>
      </div>
    </div>
  </div>

  <div class="qt-footer">
    <div class="qt-footer-line"></div>
    <div class="qt-footer-text">말씀을 묵상으로, 묵상을 삶으로</div>
  </div>
</div>
  `.trim();
}

export async function savePdfFromHtml(html) {
  return await SaveMarkdownAndMakePDF(html);
}