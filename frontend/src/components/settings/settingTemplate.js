const TEMPLATE_CATEGORIES = [
  { id: "all", label: "전체" },
  { id: "monthly", label: "월별" },
  { id: "seasonal", label: "계절별" },
  { id: "liturgical", label: "절기별" },
];

const PAGE_SIZE = 5;

const DUMMY_TEMPLATES = [
  {
    id: "tpl_0001",
    name: "3월 템플릿",
    fileName: "template_001.png",
    category: "monthly",
    previewImage: "var/template/tpl_0001/preview.png",
  },
  {
    id: "tpl_0002",
    name: "4월 템플릿",
    fileName: "template_002.png",
    category: "monthly",
    previewImage: "var/template/tpl_0002/preview.png",
  },
  {
    id: "tpl_0003",
    name: "5월 템플릿",
    fileName: "template_003.png",
    category: "monthly",
    previewImage: "var/template/tpl_0003/preview.png",
  },
  {
    id: "tpl_0004",
    name: "여름 템플릿",
    fileName: "template_004.png",
    category: "seasonal",
    previewImage: "var/template/tpl_0004/preview.png",
  },
  {
    id: "tpl_0005",
    name: "가을 템플릿",
    fileName: "template_005.png",
    category: "seasonal",
    previewImage: "var/template/tpl_0005/preview.png",
  },
  {
    id: "tpl_0006",
    name: "성탄 템플릿",
    fileName: "template_006.png",
    category: "liturgical",
    previewImage: "var/template/tpl_0006/preview.png",
  },
  {
    id: "tpl_0007",
    name: "부활절 템플릿",
    fileName: "template_007.png",
    category: "liturgical",
    previewImage: "var/template/tpl_0007/preview.png",
  },
];

let templateUiState = {
  enabled: false,
  selectedCategory: "all",
  selectedId: "",
  selectedPage: 1,
  templates: [...DUMMY_TEMPLATES],
};

function escapeHtml(value = "") {
  return String(value)
    .replaceAll("&", "&amp;")
    .replaceAll("<", "&lt;")
    .replaceAll(">", "&gt;")
    .replaceAll('"', "&quot;")
    .replaceAll("'", "&#39;");
}

function getTemplateById(templateId) {
  return templateUiState.templates.find((item) => item.id === templateId) || null;
}

function getSelectedTemplate() {
  return getTemplateById(templateUiState.selectedId);
}

function getFilteredTemplates() {
  if (templateUiState.selectedCategory === "all") {
    return templateUiState.templates;
  }

  return templateUiState.templates.filter(
    (item) => item.category === templateUiState.selectedCategory
  );
}

function getPagedTemplates() {
  const filtered = getFilteredTemplates();
  const totalPages = Math.max(1, Math.ceil(filtered.length / PAGE_SIZE));
  const currentPage = Math.min(templateUiState.selectedPage, totalPages);
  const startIndex = (currentPage - 1) * PAGE_SIZE;
  const items = filtered.slice(startIndex, startIndex + PAGE_SIZE);

  return {
    items,
    totalCount: filtered.length,
    totalPages,
    currentPage,
  };
}

function renderTemplateGuideCard() {
  return `
    <section class="card">
      <div class="settings-block">
        <h3 class="settings-block-title">템플릿 설정</h3>
        <div class="body-note topgap-sm">
          <p>PNG 산출물에 템플릿을 적용할 수 있습니다.</p>
          <p>템플릿을 선택하지 않으면 기존 PNG가 생성됩니다.</p>
          <p>템플릿은 환경설정에서 선택한 후 Step3 산출물 생성 시 반영됩니다.</p>
        </div>
      </div>

      <div class="settings-block topgap-md">
        <label class="settings-check-row">
          <input type="checkbox" id="templateEnabled" ${templateUiState.enabled ? "checked" : ""} />
          <span>템플릿 사용</span>
        </label>
      </div>
    </section>
  `;
}

function renderCategoryTabs() {
  return `
    <div class="settings-template-category-tabs">
      ${TEMPLATE_CATEGORIES.map(
        (item) => `
          <button
            type="button"
            class="step-tab ${templateUiState.selectedCategory === item.id ? "active" : ""}"
            data-template-category="${item.id}"
            ${templateUiState.enabled ? "" : "disabled"}
          >
            ${item.label}
          </button>
        `
      ).join("")}
    </div>
  `;
}

function renderTemplateTable() {
  const { items } = getPagedTemplates();

  if (!items.length) {
    return `
      <div class="settings-template-empty">
        선택한 분류에 템플릿이 없습니다.
      </div>
    `;
  }

  return `
    <div class="settings-template-table-wrap">
      <table class="settings-template-table">
        <colgroup>
          <col style="width: 88px;" />
          <col />
        </colgroup>
        <thead>
          <tr>
            <th>선택</th>
            <th>템플릿명</th>
          </tr>
        </thead>
        <tbody>
          ${items
            .map((item) => {
              const isSelected = templateUiState.selectedId === item.id;
              return `
                <tr class="${isSelected ? "is-selected" : ""}" data-template-row="${escapeHtml(item.id)}">
                  <td>
                    <label class="settings-template-radio-wrap">
                      <input
                        type="radio"
                        name="templateSelect"
                        value="${escapeHtml(item.id)}"
                        ${isSelected ? "checked" : ""}
                        ${templateUiState.enabled ? "" : "disabled"}
                      />
                    </label>
                  </td>
                  <td class="settings-template-table-name-cell">
                    ${escapeHtml(item.name)}
                  </td>
                </tr>
              `;
            })
            .join("")}
        </tbody>
      </table>
    </div>
  `;
}

function renderPagination() {
  const { totalCount, totalPages, currentPage } = getPagedTemplates();

  return `
    <div class="settings-template-pagination">
      <button
        type="button"
        class="settings-template-page-btn"
        data-template-page="prev"
        ${templateUiState.enabled && currentPage > 1 ? "" : "disabled"}
      >
        이전
      </button>

      <div class="settings-template-page-status">
        ${currentPage} / ${totalPages}
        <span class="settings-template-page-total">(${totalCount}개)</span>
      </div>

      <button
        type="button"
        class="settings-template-page-btn"
        data-template-page="next"
        ${templateUiState.enabled && currentPage < totalPages ? "" : "disabled"}
      >
        다음
      </button>
    </div>
  `;
}

function renderTemplatePreviewPanel() {
  const selected = getSelectedTemplate();

  if (!templateUiState.enabled) {
    return `
      <div class="settings-template-preview-empty">
        템플릿 사용이 꺼져 있습니다.
      </div>
    `;
  }

  if (!selected) {
    return `
      <div class="settings-template-preview-empty">
        선택된 템플릿이 없습니다.
      </div>
    `;
  }

  return `
    <div class="settings-template-preview-panel">
      <div class="settings-template-preview-large">
        <img src="${escapeHtml(selected.previewImage)}" alt="${escapeHtml(selected.name)} 미리보기" />
      </div>

      <div class="settings-template-preview-detail">
        <div class="settings-template-preview-title">${escapeHtml(selected.name)}</div>
        <div class="settings-template-preview-sub">${escapeHtml(selected.fileName)}</div>
      </div>
    </div>
  `;
}

function renderTemplateSelectionCard() {
  return `
    <section class="card topgap-md">
      <div class="settings-block">
        <h3 class="settings-block-title">템플릿 선택</h3>
      </div>

      <div class="settings-block topgap-md">
        ${renderCategoryTabs()}
      </div>

      <div class="settings-block topgap-md">
        <div class="settings-template-picker-layout">
          <div class="settings-template-picker-list">
            <div class="settings-field-label">템플릿 목록</div>
            ${renderTemplateTable()}
            ${renderPagination()}
          </div>

          <div class="settings-template-picker-preview">
            <div class="settings-field-label">미리보기</div>
            ${renderTemplatePreviewPanel()}
          </div>
        </div>
      </div>
    </section>
  `;
}

export function renderSettingTemplateTab() {
  return `
    <section class="settings-template-panel">
      ${renderTemplateGuideCard()}
      ${renderTemplateSelectionCard()}
    </section>
  `;
}

function rerenderTemplatePanelOnly() {
  const settingsContent = document.querySelector(".settings-content");
  if (!settingsContent) return;

  settingsContent.innerHTML = renderSettingTemplateTab();
  bindSettingTemplateTabEvents();
}

function selectTemplate(templateId) {
  const found = getTemplateById(templateId);
  if (!found) return;

  templateUiState.selectedId = found.id;
  rerenderTemplatePanelOnly();
}

export function bindSettingTemplateTabEvents() {
  const enabledEl = document.querySelector("#templateEnabled");
  const categoryButtons = document.querySelectorAll("[data-template-category]");
  const pageButtons = document.querySelectorAll("[data-template-page]");
  const rowButtons = document.querySelectorAll("[data-template-row]");
  const radioButtons = document.querySelectorAll('input[name="templateSelect"]');

  if (enabledEl) {
    enabledEl.addEventListener("change", () => {
      templateUiState.enabled = enabledEl.checked;

      if (!templateUiState.enabled) {
        templateUiState.selectedId = "";
      }

      templateUiState.selectedPage = 1;
      rerenderTemplatePanelOnly();
    });
  }

  categoryButtons.forEach((button) => {
    button.addEventListener("click", () => {
      if (!templateUiState.enabled) return;

      templateUiState.selectedCategory = button.dataset.templateCategory || "all";
      templateUiState.selectedPage = 1;
      rerenderTemplatePanelOnly();
    });
  });

  pageButtons.forEach((button) => {
    button.addEventListener("click", () => {
      if (!templateUiState.enabled) return;

      const action = button.dataset.templatePage;
      const { currentPage, totalPages } = getPagedTemplates();

      if (action === "prev" && currentPage > 1) {
        templateUiState.selectedPage = currentPage - 1;
      } else if (action === "next" && currentPage < totalPages) {
        templateUiState.selectedPage = currentPage + 1;
      }

      rerenderTemplatePanelOnly();
    });
  });

  rowButtons.forEach((row) => {
    row.addEventListener("click", (event) => {
      if (!templateUiState.enabled) return;

      const input = row.querySelector('input[name="templateSelect"]');
      if (!input) return;

      if (event.target.tagName !== "INPUT") {
        input.checked = true;
      }

      selectTemplate(row.dataset.templateRow || "");
    });
  });

  radioButtons.forEach((radio) => {
    radio.addEventListener("change", () => {
      if (!templateUiState.enabled) return;
      selectTemplate(radio.value || "");
    });
  });
}