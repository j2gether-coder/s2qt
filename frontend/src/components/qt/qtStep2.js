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
  const bibleText = basicInfo.bibleText || '';
  const hymnText = basicInfo.hymn || '';

  return `
    <section class="workspace-step-panel">
      <section class="card card-plain">
        <div class="step-badge">Step2. 검토 및 편집</div>
        <p class="body-note topgap-sm">초안을 검토하고 수정합니다.</p>
        <div id="qt-step2-message" class="ui-inline-message hidden"></div>
      </section>

      <section class="card editor-card">
        <div class="editor-card-head">
          <h3 class="mini-title">QT를 검토 및 편집</h3>
        </div>

        <div class="section-edit-group topgap">
          <div class="section-edit-card">
            <label class="form-label">제목</label>
            <div class="hint topgap-sm">
              Step1에서 생성된 제목을 확인하고 필요 시 수정합니다.
            </div>
            <input
              id="title"
              type="text"
              class="topgap-sm"
              value="${escapeHtml(titleText)}"
              placeholder="제목을 입력해 주세요."
            />
          </div>

          <div class="section-edit-card">
            <label class="form-label">본문 성구</label>
            <div class="hint topgap-sm">
              Step1에서 생성된 본문 성구를 확인하고 필요 시 수정합니다.
            </div>
            <input
              id="bibleText"
              type="text"
              class="topgap-sm"
              value="${escapeHtml(bibleText)}"
              placeholder="예) 시편 1:1~2"
            />
          </div>

          <div class="section-edit-card">
            <label class="form-label">찬송</label>
            <div class="hint topgap-sm">
              Step1에서 생성되었거나 기본 정보에서 입력한 찬송을 확인하고 수정합니다.
            </div>
            <input
              id="hymn"
              type="text"
              class="topgap-sm"
              value="${escapeHtml(hymnText)}"
              placeholder="예) 488장 이 몸의 소망 무언가"
            />
          </div>

          <div class="section-edit-card">
            <label class="form-label">본문 텍스트</label>
            <div class="hint topgap-sm">
              AI가 생성한 본문 텍스트 초안입니다. 성경 본문과 대조하여 확인 후 저장해 주세요.
            </div>
            <div class="hint topgap-sm">
              본문이 5절을 초과하면 Step3 산출물에서는 첫 절과 마지막 절 중심으로 축약 표시될 수 있습니다.
            </div>
            <textarea
              id="biblePassageText"
              class="topgap-sm"
              rows="6"
              placeholder="예)
1절 복 있는 사람은 악인들의 꾀를 따르지 아니하며 죄인들의 길에 서지 아니하며 오만한 자들의 자리에 앉지 아니하고
2절 오직 여호와의 율법을 즐거워하여 그의 율법을 주야로 묵상하는도다"
            ></textarea>
          </div>

          <div class="section-edit-card">
            <label class="form-label">관련 성구</label>
            <div class="hint topgap-sm">
              본문 성구 외, 말씀 이해를 돕는 관련 성구를 입력합니다. 콤마(,)로 구분하여 입력하세요.
            </div>
            <input
              id="supportScriptures"
              type="text"
              class="topgap-sm"
              placeholder="콤마(,)로 구분하여 입력해 주세요. 예) 이사야 40:31, 로마서 8:28"
            />
          </div>

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