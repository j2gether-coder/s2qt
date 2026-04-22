import { showToast, setInlineMessage, clearInlineMessage } from "../../common/uiMessage";

const TEMPLATE_MESSAGE_ID = "template-settings-message";

const TEMPLATE_CATEGORIES = [
  { id: "all", label: "전체" },
  { id: "seasonal", label: "계절별" },
  { id: "liturgical", label: "절기별" },
];

const PAGE_SIZE = 5;
const TEMPLATE_NO_IMAGE_PATH = "var/template/no_image.png";

let templateLoadPromise = null;

let templateUiState = {
  enabled: false,
  selectedCategory: "all",
  searchText: "",
  selectedId: "",
  selectedPage: 1,
  templates: [],
  noImageDataURI: "",
  isLoading: false,
  isRefreshing: false,
  hasLoaded: false,
  isSearchComposing: false,
};

function getAppBindings() {
  return window?.go?.main?.App || null;
}

function escapeHtml(value = "") {
  return String(value)
    .replaceAll("&", "&amp;")
    .replaceAll("<", "&lt;")
    .replaceAll(">", "&gt;")
    .replaceAll('"', "&quot;")
    .replaceAll("'", "&#39;");
}

function normalizeCategory(value) {
  const v = String(value || "").trim();
  const found = TEMPLATE_CATEGORIES.find((item) => item.id === v);
  return found ? found.id : "all";
}

function getTemplateById(templateId) {
  return templateUiState.templates.find((item) => item.id === templateId) || null;
}

function getSelectedTemplate() {
  return getTemplateById(templateUiState.selectedId);
}

function isValidSelectedTemplate() {
  return !!getSelectedTemplate();
}

function ensureSelectedTemplateIsValid() {
  if (!templateUiState.selectedId) return;
  if (!isValidSelectedTemplate()) {
    templateUiState.selectedId = "";
  }
}

function getFilteredTemplates() {
  const q = String(templateUiState.searchText || "").trim().toLowerCase();

  return templateUiState.templates.filter((item) => {
    if (
      templateUiState.selectedCategory !== "all" &&
      item.category !== templateUiState.selectedCategory
    ) {
      return false;
    }

    if (!q) {
      return true;
    }

    const target = [
      item.name || "",
      item.description || "",
      ...(Array.isArray(item.tags) ? item.tags : []),
      ...(Array.isArray(item.searchTerms) ? item.searchTerms : []),
    ]
      .join(" ")
      .toLowerCase();

    return target.includes(q);
  });
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

function getTemplatePreviewDataURI(item) {
  return item?.previewDataURI || templateUiState.noImageDataURI || "";
}

function getCurrentPreviewTitle() {
  const selected = getSelectedTemplate();
  if (selected) {
    return selected.name || selected.id;
  }
  return "NO IMAGE";
}

function getCurrentPreviewSubText() {
  const selected = getSelectedTemplate();
  if (!selected) {
    return "선택된 템플릿이 없습니다.";
  }

  return "";
}

async function fetchImageDataURI(path) {
  const app = getAppBindings();
  if (!app?.LoadImageAsDataURI) {
    return "";
  }

  const targetPath = String(path || "").trim();
  if (!targetPath) {
    return "";
  }

  try {
    return await app.LoadImageAsDataURI(targetPath);
  } catch (error) {
    console.error("image data uri load failed:", targetPath, error);
    return "";
  }
}

async function ensureNoImagePreviewLoaded() {
  if (templateUiState.noImageDataURI) {
    return false;
  }

  const dataURI = await fetchImageDataURI(TEMPLATE_NO_IMAGE_PATH);
  if (!dataURI) {
    return false;
  }

  templateUiState.noImageDataURI = dataURI;
  return true;
}

async function loadSelectedPreviewImageIfNeeded() {
  let changed = false;

  const loadedNoImage = await ensureNoImagePreviewLoaded();
  if (loadedNoImage) {
    changed = true;
  }

  const app = getAppBindings();
  const selected = getSelectedTemplate();

  if (!selected) {
    if (changed) {
      rerenderTemplatePickerCardOnly();
    }
    return;
  }

  if (!selected.previewDataURI) {
    try {
      if (!app?.GetTemplatePreview) {
        throw new Error("GetTemplatePreview binding is not available");
      }

      const previewPath = await app.GetTemplatePreview(selected.id);
      const dataURI = await fetchImageDataURI(previewPath);

      if (dataURI) {
        const target = getTemplateById(selected.id);
        if (target) {
          target.previewDataURI = dataURI;
          changed = true;
        }
      }
    } catch (error) {
      console.error("template preview load failed", error);
    }
  }

  if (changed) {
    rerenderTemplatePickerCardOnly();
  }
}

function renderTemplateGuideCard() {
  return `
    <section class="card">
      <div class="settings-block">
        <h3 class="settings-block-title">템플릿 설정</h3>
        <div class="body-note topgap-sm">
          <p>PDF와 PNG 산출물에 템플릿을 적용할 수 있습니다.</p>
          <p>템플릿은 환경설정에서 선택한 후 Step3 산출물 생성 시 반영됩니다.</p>
          <p>템플릿 파일은 var/template 아래에 배치되며, 현재 목록은 실제 파일 기준으로 표시됩니다.</p>
        </div>
        <div id="${TEMPLATE_MESSAGE_ID}" class="ui-inline-message hidden"></div>
      </div>

      <div class="settings-block topgap-md">
        <label class="settings-check-row">
          <input
            type="checkbox"
            id="templateEnabled"
            ${templateUiState.enabled ? "checked" : ""}
            ${templateUiState.isLoading ? "disabled" : ""}
          />
          <span>템플릿 사용</span>
        </label>
      </div>
    </section>
  `;
}

function renderCategoryTabs() {
  return `
    <div class="radio-inline-group" role="radiogroup" aria-label="템플릿 분류">
      ${TEMPLATE_CATEGORIES.map((item) => `
        <label class="radio-inline-item">
          <input
            type="radio"
            name="templateCategory"
            value="${item.id}"
            ${templateUiState.selectedCategory === item.id ? "checked" : ""}
            ${templateUiState.isLoading ? "disabled" : ""}
          />
          <span>${item.label}</span>
        </label>
      `).join("")}
    </div>
  `;
}

function renderTemplateEmptyMessage() {
  if (templateUiState.isLoading) {
    return "템플릿 목록을 불러오는 중입니다.";
  }

  if (!templateUiState.templates.length) {
    return "등록된 템플릿이 없습니다. var/template에 템플릿을 배치한 후 새로 고침을 눌러 주세요.";
  }

  return "선택한 분류에 템플릿이 없습니다.";
}

function renderTemplateTable() {
  const { items } = getPagedTemplates();

  if (!items.length) {
    return `
      <div class="settings-template-empty">
        ${escapeHtml(renderTemplateEmptyMessage())}
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
              const disabled = templateUiState.isLoading || !item.isValid;

              return `
                <tr class="${isSelected ? "is-selected" : ""} ${!item.isValid ? "is-disabled" : ""}" data-template-row="${escapeHtml(item.id)}">
                  <td>
                    <label class="settings-template-radio-wrap">
                      <input
                        type="radio"
                        name="templateSelect"
                        value="${escapeHtml(item.id)}"
                        ${isSelected ? "checked" : ""}
                        ${disabled ? "disabled" : ""}
                      />
                    </label>
                  </td>
                  <td class="settings-template-table-name-cell">
                    ${escapeHtml(item.name || item.id)}
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
        ${currentPage > 1 && !templateUiState.isLoading ? "" : "disabled"}
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
        ${currentPage < totalPages && !templateUiState.isLoading ? "" : "disabled"}
      >
        다음
      </button>
    </div>
  `;
}

function renderTemplatePreviewPanel() {
  const selected = getSelectedTemplate();
  const previewDataURI = selected ? getTemplatePreviewDataURI(selected) : templateUiState.noImageDataURI;

  const previewContent = previewDataURI
    ? `<img src="${previewDataURI}" alt="${escapeHtml(getCurrentPreviewTitle())} 미리보기" />`
    : `<div class="settings-template-preview-empty">no_image.png를 불러오는 중입니다.</div>`;

  const previewSubText = getCurrentPreviewSubText();

  return `
    <div class="settings-template-preview-panel">
      <div class="settings-template-preview-large">
        ${previewContent}
      </div>
      ${
        previewSubText
          ? `<div class="settings-template-preview-sub">${escapeHtml(previewSubText)}</div>`
          : ``
      }
    </div>
  `;
}

function renderTemplateFilterCard() {
  return `
    <section class="card topgap-md">
      <div class="settings-block">
        <h3 class="settings-block-title">템플릿 선택</h3>
      </div>

      <div class="settings-block topgap-md">
        <div class="settings-template-filter-row">
          <div class="settings-template-filter-label">분류</div>
          <div class="settings-template-filter-control">
            ${renderCategoryTabs()}
          </div>
        </div>
      </div>

      <div class="settings-block topgap-md">
        <div class="settings-template-search-block">
          <input
            type="text"
            id="templateSearchText"
            class="input settings-template-search-input"
            placeholder="검색어를 입력하세요"
            value="${escapeHtml(templateUiState.searchText || "")}"
            ${templateUiState.isLoading ? "disabled" : ""}
          />
          <div class="settings-template-search-help">
            예) 편지지, 봄, seasonal, liturgical, 부활절, 성탄절 등
          </div>
        </div>
      </div>
    </section>
  `;
}

function renderTemplatePickerCard() {
  const refreshLabel = templateUiState.isRefreshing ? "⏳ 새로 고침 중" : "🔄 새로 고침";

  return `
    <section class="card topgap-md" id="templatePickerCard">
      <div class="settings-block">
        <h3 class="settings-block-title">템플릿 목록 / 미리보기</h3>
      </div>

      <div class="settings-block topgap-md">
        <div class="settings-template-picker-layout">
          <div class="settings-template-picker-list">
            <div style="display:grid; grid-template-columns: 1fr 1fr; gap:12px; align-items:center; margin-bottom:8px;">
              <div class="settings-field-label" style="margin-bottom:0;">템플릿 목록</div>
              <button
                type="button"
                class="settings-template-page-btn"
                id="templateRefreshBtn"
                title="목록 새로고침"
                aria-label="목록 새로고침"
                ${templateUiState.isLoading || templateUiState.isRefreshing ? "disabled" : ""}
              >
                ${refreshLabel}
              </button>
            </div>

            ${renderTemplateTable()}
            ${renderPagination()}
          </div>

          <div class="settings-template-picker-preview">
            <div style="display:flex; align-items:center; justify-content:space-between; gap:12px; margin-bottom:8px;">
              <div class="settings-field-label" style="margin-bottom:0;">미리보기</div>
              <div style="font-size:13px; font-weight:400; color:#374151; line-height:1.4; text-align:right;">
                ${escapeHtml(getCurrentPreviewTitle())}
              </div>
            </div>
            ${renderTemplatePreviewPanel()}
          </div>
        </div>
      </div>
    </section>
  `;
}

export function renderSettingTemplateTab() {
  return `
    <section class="settings-template-panel" id="settingsTemplateRoot">
      ${renderTemplateGuideCard()}
      ${renderTemplateFilterCard()}
      ${renderTemplatePickerCard()}
    </section>
  `;
}

function rerenderTemplatePanelOnly() {
  const root = document.querySelector("#settingsTemplateRoot");
  if (!root) return;

  root.outerHTML = renderSettingTemplateTab();
  bindSettingTemplateTabEvents();
}

async function persistTemplateSettings() {
  const app = getAppBindings();
  if (!app?.SaveTemplateSettings) {
    throw new Error("SaveTemplateSettings binding is not available");
  }

  ensureSelectedTemplateIsValid();

  await app.SaveTemplateSettings({
    enabled: !!templateUiState.enabled,
    selectedCategory: templateUiState.selectedCategory,
    selectedId: templateUiState.selectedId,
  });
}

function mapTemplateSettings(settings) {
  templateUiState.enabled = !!settings?.enabled;
  templateUiState.selectedCategory = normalizeCategory(settings?.selectedCategory);
  templateUiState.selectedId = String(settings?.selectedId || "").trim();
}

function mapTemplates(items) {
  templateUiState.templates = Array.isArray(items)
    ? items.map((item) => ({
        id: String(item?.id || "").trim(),
        name: String(item?.name || item?.id || "").trim(),
        category: normalizeCategory(item?.category),
        description: String(item?.description || "").trim(),
        tags: Array.isArray(item?.tags)
          ? item.tags.map((v) => String(v || "").trim()).filter(Boolean)
          : [],
        searchTerms: Array.isArray(item?.searchTerms)
          ? item.searchTerms.map((v) => String(v || "").trim()).filter(Boolean)
          : [],
        hasPdfAsset: !!item?.hasPdfAsset,
        hasPngAsset: !!item?.hasPngAsset,
        isValid: item?.isValid !== false,
        previewDataURI: "",
      }))
    : [];

  ensureSelectedTemplateIsValid();
}

async function loadTemplateState() {
  const app = getAppBindings();
  if (!app?.LoadTemplateSettings || !app?.ListTemplates) {
    throw new Error("Template bindings are not available");
  }

  const [settings, templates] = await Promise.all([
    app.LoadTemplateSettings(),
    app.ListTemplates(),
  ]);

  mapTemplateSettings(settings || {});
  mapTemplates(templates || []);
  templateUiState.selectedPage = 1;
}

async function ensureTemplateTabInitialized(force = false) {
  if (templateUiState.hasLoaded && !force) {
    void loadSelectedPreviewImageIfNeeded();
    return;
  }

  if (templateLoadPromise && !force) {
    await templateLoadPromise;
    return;
  }

  templateUiState.isLoading = true;
  clearInlineMessage(TEMPLATE_MESSAGE_ID);
  rerenderTemplatePanelOnly();

  templateLoadPromise = (async () => {
    let initSucceeded = false;

    try {
      await loadTemplateState();
      templateUiState.hasLoaded = true;
      initSucceeded = true;
    } catch (error) {
      console.error("template settings init failed", error);
      templateUiState.hasLoaded = false;
      setInlineMessage(
        TEMPLATE_MESSAGE_ID,
        "템플릿 설정 정보를 불러오지 못했습니다.",
        "error"
      );
    } finally {
      templateUiState.isLoading = false;
      rerenderTemplatePanelOnly();

      if (initSucceeded) {
        void loadSelectedPreviewImageIfNeeded();
      }

      templateLoadPromise = null;
    }
  })();

  await templateLoadPromise;
}

function ensureTemplateTabInitializedIfNeeded() {
  if (!templateUiState.hasLoaded && !templateLoadPromise) {
    void ensureTemplateTabInitialized();
  }
}

async function refreshTemplateList() {
  const app = getAppBindings();
  if (!app?.RefreshTemplates) {
    setInlineMessage(
      TEMPLATE_MESSAGE_ID,
      "템플릿 목록 새로고침 기능을 사용할 수 없습니다.",
      "error"
    );
    rerenderTemplatePanelOnly();
    return;
  }

  templateUiState.isRefreshing = true;
  clearInlineMessage(TEMPLATE_MESSAGE_ID);
  rerenderTemplatePanelOnly();

  try {
    const items = await app.RefreshTemplates();
    mapTemplates(items || []);
    templateUiState.selectedPage = 1;

    ensureSelectedTemplateIsValid();
    await persistTemplateSettings();

    showToast("템플릿 목록을 새로고침했습니다.", "success");
  } catch (error) {
    console.error("template refresh failed", error);
    setInlineMessage(
      TEMPLATE_MESSAGE_ID,
      "템플릿 목록을 새로고침하지 못했습니다.",
      "error"
    );
  } finally {
    templateUiState.isRefreshing = false;
    rerenderTemplatePanelOnly();
    void loadSelectedPreviewImageIfNeeded();
  }
}

async function updateTemplateEnabled(enabled) {
  const wasEnabled = templateUiState.enabled;
  templateUiState.enabled = !!enabled;

  if (wasEnabled && !templateUiState.enabled) {
    templateUiState.selectedId = "";
  }

  clearInlineMessage(TEMPLATE_MESSAGE_ID);

  try {
    await persistTemplateSettings();
  } catch (error) {
    console.error("template enabled save failed", error);
    setInlineMessage(
      TEMPLATE_MESSAGE_ID,
      "템플릿 사용 설정 저장에 실패했습니다.",
      "error"
    );
  }

  rerenderTemplatePanelOnly();

  if (templateUiState.enabled) {
    void loadSelectedPreviewImageIfNeeded();
  }
}

async function updateTemplateCategory(category) {
  templateUiState.selectedCategory = normalizeCategory(category);
  templateUiState.selectedPage = 1;

  clearInlineMessage(TEMPLATE_MESSAGE_ID);

  try {
    await persistTemplateSettings();
  } catch (error) {
    console.error("template category save failed", error);
    setInlineMessage(
      TEMPLATE_MESSAGE_ID,
      "템플릿 분류 저장에 실패했습니다.",
      "error"
    );
  }

  rerenderTemplatePanelOnly();
}

async function selectTemplate(templateId) {
  const found = getTemplateById(templateId);
  if (!found || !found.isValid) return;

  templateUiState.selectedId = found.id;
  clearInlineMessage(TEMPLATE_MESSAGE_ID);

  try {
    await persistTemplateSettings();
  } catch (error) {
    console.error("template select save failed", error);
    setInlineMessage(
      TEMPLATE_MESSAGE_ID,
      "선택한 템플릿 저장에 실패했습니다.",
      "error"
    );
  }

  rerenderTemplatePanelOnly();
  void loadSelectedPreviewImageIfNeeded();
}

function rerenderTemplatePickerCardOnly() {
  const pickerCard = document.querySelector("#templatePickerCard");
  if (!pickerCard) {
    rerenderTemplatePanelOnly();
    return;
  }

  pickerCard.outerHTML = renderTemplatePickerCard();
  bindTemplatePickerEvents();
}

function applyTemplateSearchText(value) {
  templateUiState.searchText = String(value || "");
  templateUiState.selectedPage = 1;
  rerenderTemplatePickerCardOnly();
}

function bindTemplateFilterEvents() {
  const enabledEl = document.querySelector("#templateEnabled");
  const categoryRadios = document.querySelectorAll('input[name="templateCategory"]');
  const searchInput = document.querySelector("#templateSearchText");

  if (enabledEl) {
    enabledEl.addEventListener("change", () => {
      void updateTemplateEnabled(enabledEl.checked);
    });
  }

  categoryRadios.forEach((radio) => {
    radio.addEventListener("change", () => {
      if (radio.disabled) return;
      void updateTemplateCategory(radio.value || "all");
    });
  });

  if (searchInput) {
    searchInput.addEventListener("compositionstart", () => {
      templateUiState.isSearchComposing = true;
    });

    searchInput.addEventListener("compositionend", (event) => {
      templateUiState.isSearchComposing = false;
      applyTemplateSearchText(event.target?.value || "");
    });

    searchInput.addEventListener("input", (event) => {
      if (templateUiState.isSearchComposing || event.isComposing) return;
      applyTemplateSearchText(event.target?.value || "");
    });
  }
}

function bindTemplatePickerEvents() {
  const pageButtons = document.querySelectorAll("[data-template-page]");
  const rowButtons = document.querySelectorAll("[data-template-row]");
  const radioButtons = document.querySelectorAll('input[name="templateSelect"]');
  const refreshBtn = document.querySelector("#templateRefreshBtn");

  pageButtons.forEach((button) => {
    button.addEventListener("click", () => {
      if (templateUiState.isLoading) return;

      const action = button.dataset.templatePage;
      const { currentPage, totalPages } = getPagedTemplates();

      if (action === "prev" && currentPage > 1) {
        templateUiState.selectedPage = currentPage - 1;
      } else if (action === "next" && currentPage < totalPages) {
        templateUiState.selectedPage = currentPage + 1;
      }

      rerenderTemplatePickerCardOnly();
    });
  });

  rowButtons.forEach((row) => {
    row.addEventListener("click", (event) => {
      const input = row.querySelector('input[name="templateSelect"]');
      if (!input || input.disabled) return;

      if (event.target?.tagName !== "INPUT") {
        input.checked = true;
      }

      void selectTemplate(row.dataset.templateRow || "");
    });
  });

  radioButtons.forEach((radio) => {
    radio.addEventListener("change", () => {
      if (radio.disabled) return;
      void selectTemplate(radio.value || "");
    });
  });

  if (refreshBtn) {
    refreshBtn.addEventListener("click", () => {
      void refreshTemplateList();
    });
  }
}

export function bindSettingTemplateTabEvents() {
  ensureTemplateTabInitializedIfNeeded();
  bindTemplateFilterEvents();
  bindTemplatePickerEvents();
}
