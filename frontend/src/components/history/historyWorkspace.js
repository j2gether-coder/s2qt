import {
  ListHistory,
  GetHistory,
  GetHistoryStep1,
  DeleteHistory,
} from "../../../wailsjs/go/main/App";
import { appState, setSelectedMenu, setAudienceStep } from "../../state/appState";
import { showToast, setInlineMessage, clearInlineMessage } from "../../common/uiMessage";
import { mountAppShell } from "../appShell";

const HISTORY_MESSAGE_ID = "history-workspace-message";

const AUDIENCE_OPTIONS = [
  { id: "adult", label: "장년" },
  { id: "young_adult", label: "청년" },
  { id: "teen", label: "중고등" },
  { id: "child", label: "어린이" },
];

let historyState = {
  loaded: false,
  items: [],
  availabilityMap: {},
  selectedIds: [],
  filters: {
    keyword: "",
    audience: "",
    sortBy: "createdAt",
    sortDir: "desc",
  },
};

function escapeHtml(value) {
  return String(value ?? "")
    .replaceAll("&", "&amp;")
    .replaceAll("<", "&lt;")
    .replaceAll(">", "&gt;")
    .replaceAll('"', "&quot;")
    .replaceAll("'", "&#39;");
}

function formatDisplay(value, fallback = "-") {
  const text = String(value ?? "").trim();
  return text ? text : fallback;
}

function formatDateTime(value) {
  return formatDisplay(value, "-");
}

function normalizeText(value) {
  return String(value ?? "").trim().toLowerCase();
}

function isSelected(historyId) {
  return historyState.selectedIds.includes(Number(historyId));
}

function getSelectedCount() {
  return historyState.selectedIds.length;
}

function getSortByOptionsHtml(selectedValue) {
  const options = [
    { value: "createdAt", label: "저장일" },
    { value: "title", label: "제목" },
    { value: "bibleText", label: "본문 성구" },
  ];

  return options
    .map(
      (option) => `
        <option value="${option.value}" ${selectedValue === option.value ? "selected" : ""}>
          ${option.label}
        </option>
      `
    )
    .join("");
}

function getSortDirOptionsHtml(selectedValue) {
  const options = [
    { value: "desc", label: "내림차순" },
    { value: "asc", label: "오름차순" },
  ];

  return options
    .map(
      (option) => `
        <option value="${option.value}" ${selectedValue === option.value ? "selected" : ""}>
          ${option.label}
        </option>
      `
    )
    .join("");
}

function getAudienceFilterOptionsHtml(selectedValue) {
  return `
    <option value="">전체</option>
    ${AUDIENCE_OPTIONS.map(
      (audience) => `
        <option value="${audience.id}" ${selectedValue === audience.id ? "selected" : ""}>
          ${audience.label}
        </option>
      `
    ).join("")}
  `;
}

async function loadHistoryList() {
  const items = await ListHistory();
  historyState.items = Array.isArray(items) ? items : [];
  historyState.loaded = true;
}

async function loadHistoryAvailability() {
  const availabilityMap = {};

  await Promise.all(
    historyState.items.map(async (item) => {
      const availability = {
        adult: false,
        young_adult: false,
        teen: false,
        child: false,
      };

      await Promise.all(
        AUDIENCE_OPTIONS.map(async (audience) => {
          try {
            const step1 = await GetHistoryStep1(item.id, audience.id);
            availability[audience.id] = !!(step1 && step1.step1ResultJson);
          } catch (error) {
            availability[audience.id] = false;
          }
        })
      );

      availabilityMap[item.id] = availability;
    })
  );

  historyState.availabilityMap = availabilityMap;
}

async function reloadHistoryWorkspace() {
  await loadHistoryList();
  await loadHistoryAvailability();

  const validIds = new Set(historyState.items.map((item) => Number(item.id)));
  historyState.selectedIds = historyState.selectedIds.filter((id) => validIds.has(id));
}

function getAvailableAudienceLabels(itemId) {
  const availability = historyState.availabilityMap[itemId] || {};

  return AUDIENCE_OPTIONS
    .filter((audience) => !!availability[audience.id])
    .map((audience) => audience.label);
}

function getFilteredHistoryItems() {
  const keyword = normalizeText(historyState.filters.keyword);
  const audienceFilter = historyState.filters.audience;
  const sortBy = historyState.filters.sortBy;
  const sortDir = historyState.filters.sortDir;

  const filtered = historyState.items.filter((item) => {
    const haystack = [
      item.title,
      item.bibleText,
      item.preacher,
      item.churchName,
      item.hymn,
      item.sermonDate,
    ]
      .map((v) => normalizeText(v))
      .join(" ");

    const keywordMatched = !keyword || haystack.includes(keyword);

    const availability = historyState.availabilityMap[item.id] || {};
    const audienceMatched = !audienceFilter || !!availability[audienceFilter];

    return keywordMatched && audienceMatched;
  });

  filtered.sort((a, b) => {
    let av = a?.[sortBy] ?? "";
    let bv = b?.[sortBy] ?? "";

    av = String(av ?? "");
    bv = String(bv ?? "");

    const compared = av.localeCompare(bv, "ko", { numeric: true, sensitivity: "base" });
    return sortDir === "asc" ? compared : -compared;
  });

  return filtered;
}

function renderHistoryHeaderCard() {
  return `
    <section class="card card-plain">
      <div class="step-badge">작업 내역</div>
      <p class="body-note topgap-sm">
        저장된 기본정보와 Step1 결과를 조회하고 재작업하거나 삭제할 수 있습니다.
      </p>
    </section>
  `;
}

function renderSearchConditionCard() {
  return `
    <section class="card">
      <h3 class="mini-title">검색 조건</h3>
      <p class="body-note topgap-sm">검색어와 정렬 조건을 선택해 작업 내역을 조회합니다.</p>

      <div class="form-grid two-column-grid topgap-sm">
        <div class="form-field">
          <label class="form-label">검색어</label>
          <input
            type="text"
            id="history-keyword-input"
            class="input"
            value="${escapeHtml(historyState.filters.keyword)}"
            placeholder="제목 / 본문 성구"
          />
        </div>

        <div class="form-field">
          <label class="form-label">연령대</label>
          <select id="history-audience-filter" class="input">
            ${getAudienceFilterOptionsHtml(historyState.filters.audience)}
          </select>
        </div>

        <div class="form-field">
          <label class="form-label">정렬 기준</label>
          <select id="history-sort-by" class="input">
            ${getSortByOptionsHtml(historyState.filters.sortBy)}
          </select>
        </div>

        <div class="form-field">
          <label class="form-label">정렬 방향</label>
          <select id="history-sort-dir" class="input">
            ${getSortDirOptionsHtml(historyState.filters.sortDir)}
          </select>
        </div>
      </div>

      <div class="half-action-row topgap-sm">
        <button type="button" class="button" id="history-search-btn">검색</button>
        <button type="button" class="button-ghost" id="history-reset-btn">초기화</button>
      </div>
    </section>
  `;
}

function renderActionCard() {
  const selectedCount = getSelectedCount();
  const canRework = selectedCount === 1;
  const canDelete = selectedCount >= 1;

  return `
    <section class="card">
      <h3 class="mini-title">작업</h3>
      <p class="body-note topgap-sm">선택한 작업에 대해 재작업 또는 삭제를 수행합니다.</p>

      <div class="mode-strip topgap-sm">
        <span class="mode-label">선택 건수</span>
        <span class="mode-value">${selectedCount}건</span>
      </div>

      <div class="half-action-row topgap-sm">
        <button
          type="button"
          class="button"
          id="history-rework-btn"
          ${canRework ? "" : "disabled"}
        >
          재작업
        </button>

        <button
          type="button"
          class="danger-button"
          id="history-delete-btn"
          ${canDelete ? "" : "disabled"}
        >
          삭제
        </button>
      </div>
    </section>
  `;
}

function renderHistoryTableRows() {
  const items = getFilteredHistoryItems();

  if (!items.length) {
    return `
      <tr>
        <td colspan="5">
          <div class="history-empty-box">검색 결과가 없습니다.</div>
        </td>
      </tr>
    `;
  }

  return items
    .map((item) => {
      const availableAudienceLabels = getAvailableAudienceLabels(item.id);

      return `
        <tr>
          <td class="history-table-check">
            <input
              type="checkbox"
              data-history-select-id="${item.id}"
              ${isSelected(item.id) ? "checked" : ""}
            />
          </td>
          <td title="${escapeHtml(formatDisplay(item.title))}">
            ${escapeHtml(formatDisplay(item.title))}
          </td>
          <td title="${escapeHtml(formatDisplay(item.bibleText))}">
            ${escapeHtml(formatDisplay(item.bibleText))}
          </td>
          <td>${escapeHtml(formatDateTime(item.createdAt))}</td>
          <td>${escapeHtml(availableAudienceLabels.join(", ") || "-")}</td>
        </tr>
      `;
    })
    .join("");
}

function renderHistoryTableCard() {
  const count = getFilteredHistoryItems().length;
  const allFilteredIds = getFilteredHistoryItems().map((item) => Number(item.id));
  const allChecked =
    allFilteredIds.length > 0 &&
    allFilteredIds.every((id) => historyState.selectedIds.includes(id));

  return `
    <section class="card">
      <h3 class="mini-title">작업 내역</h3>
      <p class="body-note topgap-sm">검색 조건에 맞는 작업 내역을 표로 확인합니다.</p>

      <div class="mode-strip topgap-sm">
        <span class="mode-label">조회 건수</span>
        <span class="mode-value">${count}건</span>
      </div>

      <div class="history-table-wrap topgap-sm">
        <table class="history-table">
          <thead>
            <tr>
              <th class="history-table-check">
                <input
                  type="checkbox"
                  id="history-select-all"
                  ${allChecked ? "checked" : ""}
                />
              </th>
              <th>제목</th>
              <th>본문 성구</th>
              <th>저장일</th>
              <th>연령대</th>
            </tr>
          </thead>
          <tbody>
            ${renderHistoryTableRows()}
          </tbody>
        </table>
      </div>
    </section>
  `;
}

function renderHistoryLoadingState() {
  return `
    <section class="card card-plain">
      <div class="mini-title">작업 내역</div>
      <p class="body-note topgap-sm">작업 내역을 불러오는 중입니다.</p>
    </section>
  `;
}

export function renderHistoryWorkspace() {
  if (!historyState.loaded) {
    return `
      <section class="workspace-step-panel">
        ${renderHistoryHeaderCard()}
        <div id="${HISTORY_MESSAGE_ID}" class="ui-inline-message hidden"></div>
        ${renderHistoryLoadingState()}
      </section>
    `;
  }

  return `
    <section class="workspace-step-panel">
      ${renderHistoryHeaderCard()}
      <div id="${HISTORY_MESSAGE_ID}" class="ui-inline-message hidden"></div>
      ${renderSearchConditionCard()}
      ${renderActionCard()}
      ${renderHistoryTableCard()}
    </section>
  `;
}

async function rerenderHistoryWorkspace() {
  const root = document.querySelector(".main-workspace");
  if (!root) return;

  try {
    await reloadHistoryWorkspace();
    root.innerHTML = renderHistoryWorkspace();
    bindHistoryWorkspaceEvents();
  } catch (error) {
    console.error(error);
    root.innerHTML = renderHistoryWorkspace();
    setInlineMessage(
      HISTORY_MESSAGE_ID,
      error?.message || "작업 내역을 불러오는 중 오류가 발생했습니다.",
      "error"
    );
  }
}

function updateSelection(historyId, checked) {
  const id = Number(historyId);

  if (checked) {
    if (!historyState.selectedIds.includes(id)) {
      historyState.selectedIds.push(id);
    }
  } else {
    historyState.selectedIds = historyState.selectedIds.filter((itemId) => itemId !== id);
  }
}

function handleSelectAll(checked) {
  const filteredIds = getFilteredHistoryItems().map((item) => Number(item.id));

  if (checked) {
    const merged = new Set([...historyState.selectedIds, ...filteredIds]);
    historyState.selectedIds = Array.from(merged);
  } else {
    historyState.selectedIds = historyState.selectedIds.filter((id) => !filteredIds.includes(id));
  }

  const root = document.querySelector(".main-workspace");
  if (!root) return;

  root.innerHTML = renderHistoryWorkspace();
  bindHistoryWorkspaceEvents();
}

function handleSearch() {
  clearInlineMessage(HISTORY_MESSAGE_ID);

  historyState.filters.keyword =
    document.getElementById("history-keyword-input")?.value || "";
  historyState.filters.audience =
    document.getElementById("history-audience-filter")?.value || "";
  historyState.filters.sortBy =
    document.getElementById("history-sort-by")?.value || "createdAt";
  historyState.filters.sortDir =
    document.getElementById("history-sort-dir")?.value || "desc";

  const root = document.querySelector(".main-workspace");
  if (!root) return;

  root.innerHTML = renderHistoryWorkspace();
  bindHistoryWorkspaceEvents();
}

function handleReset() {
  clearInlineMessage(HISTORY_MESSAGE_ID);

  historyState.filters.keyword = "";
  historyState.filters.audience = "";
  historyState.filters.sortBy = "createdAt";
  historyState.filters.sortDir = "desc";
  historyState.selectedIds = [];

  const root = document.querySelector(".main-workspace");
  if (!root) return;

  root.innerHTML = renderHistoryWorkspace();
  bindHistoryWorkspaceEvents();
}

async function handleDeleteSelected() {
  clearInlineMessage(HISTORY_MESSAGE_ID);

  if (historyState.selectedIds.length === 0) {
    setInlineMessage(HISTORY_MESSAGE_ID, "삭제할 작업을 선택해 주세요.", "warning");
    return;
  }

  try {
    for (const id of historyState.selectedIds) {
      await DeleteHistory(Number(id));
    }

    historyState.selectedIds = [];
    showToast("선택한 작업 내역이 삭제되었습니다.", "success");
    await rerenderHistoryWorkspace();
  } catch (error) {
    console.error(error);
    setInlineMessage(
      HISTORY_MESSAGE_ID,
      error?.message || "작업 내역 삭제 중 오류가 발생했습니다.",
      "error"
    );
  }
}

async function handleReworkSelected() {
  clearInlineMessage(HISTORY_MESSAGE_ID);

  if (historyState.selectedIds.length !== 1) {
    setInlineMessage(HISTORY_MESSAGE_ID, "재작업은 1건만 선택해 주세요.", "warning");
    return;
  }

  const historyId = Number(historyState.selectedIds[0]);
  const availability = historyState.availabilityMap[historyId] || {};
  const selectedAudience = AUDIENCE_OPTIONS.find((audience) => availability[audience.id]);

  if (!selectedAudience) {
    setInlineMessage(HISTORY_MESSAGE_ID, "재작업 가능한 연령대가 없습니다.", "warning");
    return;
  }

  try {
    const history = await GetHistory(historyId);
    const step1 = await GetHistoryStep1(historyId, selectedAudience.id);

    if (!history || !step1 || !step1.step1ResultJson) {
      setInlineMessage(
        HISTORY_MESSAGE_ID,
        "선택한 작업의 Step1 결과를 찾을 수 없습니다.",
        "warning"
      );
      return;
    }

    if (appState.source?.basicInfo) {
      appState.source.basicInfo.title = history.title || "";
      appState.source.basicInfo.bibleText = history.bibleText || "";
      appState.source.basicInfo.hymn = history.hymn || "";
      appState.source.basicInfo.preacher = history.preacher || "";
      appState.source.basicInfo.churchName = history.churchName || "";
      appState.source.basicInfo.sermonDate = history.sermonDate || "";
    }

    appState.historySelected = {
      historyId,
      audienceId: selectedAudience.id,
      step1ResultJson: step1.step1ResultJson,
    };

    setSelectedMenu(selectedAudience.id);
    setAudienceStep(selectedAudience.id, "step2");
    mountAppShell("app");
  } catch (error) {
    console.error(error);
    setInlineMessage(
      HISTORY_MESSAGE_ID,
      error?.message || "재작업 준비 중 오류가 발생했습니다.",
      "error"
    );
  }
}

export async function bindHistoryWorkspaceEvents() {
  try {
    if (!historyState.loaded) {
      await rerenderHistoryWorkspace();
      return;
    }
  } catch (error) {
    console.error(error);
    setInlineMessage(
      HISTORY_MESSAGE_ID,
      error?.message || "작업 내역 초기화 중 오류가 발생했습니다.",
      "error"
    );
    return;
  }

  const searchBtn = document.getElementById("history-search-btn");
  if (searchBtn) {
    searchBtn.addEventListener("click", () => {
      handleSearch();
    });
  }

  const resetBtn = document.getElementById("history-reset-btn");
  if (resetBtn) {
    resetBtn.addEventListener("click", () => {
      handleReset();
    });
  }

  const keywordInput = document.getElementById("history-keyword-input");
  if (keywordInput) {
    keywordInput.addEventListener("keydown", (event) => {
      if (event.key === "Enter") {
        event.preventDefault();
        handleSearch();
      }
    });
  }

  const selectAll = document.getElementById("history-select-all");
  if (selectAll) {
    selectAll.addEventListener("change", () => {
      handleSelectAll(selectAll.checked);
    });
  }

  const rowCheckboxes = document.querySelectorAll("[data-history-select-id]");
  rowCheckboxes.forEach((checkbox) => {
    checkbox.addEventListener("change", () => {
      updateSelection(checkbox.dataset.historySelectId, checkbox.checked);

      const root = document.querySelector(".main-workspace");
      if (!root) return;

      root.innerHTML = renderHistoryWorkspace();
      bindHistoryWorkspaceEvents();
    });
  });

  const reworkBtn = document.getElementById("history-rework-btn");
  if (reworkBtn) {
    reworkBtn.addEventListener("click", async () => {
      await handleReworkSelected();
    });
  }

  const deleteBtn = document.getElementById("history-delete-btn");
  if (deleteBtn) {
    deleteBtn.addEventListener("click", async () => {
      await handleDeleteSelected();
    });
  }
}