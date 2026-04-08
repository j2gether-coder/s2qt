function escapeHtml(value) {
  return String(value ?? '')
    .replaceAll('&', '&amp;')
    .replaceAll('<', '&lt;')
    .replaceAll('>', '&gt;')
    .replaceAll('"', '&quot;')
    .replaceAll("'", '&#39;');
}

function resolveInitialReadonlyTitle(audienceId, basicInfo) {
  const metaTitle = (basicInfo?.title || '').trim();

  // 초기 렌더 단계에서는 아직 Step2 로드 데이터가 없으므로
  // 우선 메타 제목을 fallback으로 보여주고,
  // 실제 LLM 제목 반영은 bindQTStep2Events > loadStep2Data()에서 갱신
  return metaTitle || '-';
}

export function renderQTStep2(audienceId, appState) {
  const basicInfo = appState?.source?.basicInfo || {};
  const titleText = resolveInitialReadonlyTitle(audienceId, basicInfo);
  const bibleText = basicInfo.bibleText || '-';

  return `
    <section class="workspace-step-panel">
      <section class="card card-plain">
        <div class="step-badge">Step2. 검토 및 편집</div>
        <p class="body-note topgap-sm">초안을 검토하고 수정합니다.</p>
      </section>

      <section class="card card-soft-meta">
        <div class="meta-readonly-grid">
          <div class="meta-readonly-item">
            <div class="meta-readonly-label">제목</div>
            <div class="meta-readonly-value" id="qtReadonlyTitle">${escapeHtml(titleText)}</div>
          </div>

          <div class="meta-readonly-item">
            <div class="meta-readonly-label">본문 성구</div>
            <div class="meta-readonly-value" id="qtReadonlyBibleText">${escapeHtml(bibleText)}</div>
          </div>
        </div>
      </section>

      <section class="card editor-card">
        <div class="editor-card-head">
          <h3 class="mini-title">QT를 검토 및 편집</h3>
        </div>

        <div class="section-edit-group topgap">
          <div class="section-edit-card">
            <label class="form-label">말씀의 창</label>
            <input id="summaryTitle" type="text" />
            <textarea id="summaryBody" class="textarea-3rows"></textarea>
          </div>

          <div class="section-edit-card">
            <label class="form-label">오늘의 메시지 1</label>
            <input id="messageTitle1" type="text" />
            <textarea id="messageBody1" class="textarea-3rows"></textarea>
          </div>

          <div class="section-edit-card">
            <label class="form-label">오늘의 메시지 2</label>
            <input id="messageTitle2" type="text" />
            <textarea id="messageBody2" class="textarea-3rows"></textarea>
          </div>

          <div class="section-edit-card">
            <label class="form-label">오늘의 메시지 3</label>
            <input id="messageTitle3" type="text" />
            <textarea id="messageBody3" class="textarea-3rows"></textarea>
          </div>

          <div class="section-edit-card">
            <label class="form-label">깊은 묵상과 적용 1</label>
            <input id="reflectionItem1" type="text" />
          </div>

          <div class="section-edit-card">
            <label class="form-label">깊은 묵상과 적용 2</label>
            <input id="reflectionItem2" type="text" />
          </div>

          <div class="section-edit-card">
            <label class="form-label">깊은 묵상과 적용 3</label>
            <input id="reflectionItem3" type="text" />
          </div>

          <div class="section-edit-card">
            <label class="form-label">오늘의 기도</label>
            <input id="prayerTitle" type="text" />
            <textarea id="prayerBody" class="textarea-3rows"></textarea>
          </div>
        </div>
      </section>

      <section class="card">
        <div class="half-action-row">
          <button id="saveQtJsonBtn" class="button" type="button">저장</button>
          <button id="previewHtmlBtn" class="button-ghost" type="button">미리보기</button>
        </div>
      </section>

      <section class="step-bottom-bar double">
        <div class="step-bottom-left">
          <button id="backToStep1Btn" class="button-ghost" type="button">이전</button>
        </div>
        <div class="step-bottom-right">
          <button id="goStep3Btn" class="button" type="button" disabled>다음</button>
        </div>
      </section>
    </section>
  `;
}