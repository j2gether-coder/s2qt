import {
  CountHistory,
  ListHistoryPaged,
  GetHistoryQTJSON,
  DeleteHistory,
  PrepareReworkFromHistory,
} from "../../../wailsjs/go/main/App";
import { appState, setAudienceStep, setBasicInfoField, setSelectedMenu } from "../../state/appState";
import { showToast, setInlineMessage, clearInlineMessage } from "../../common/uiMessage";
import { mountAppShell } from "../appShell";

const HISTORY_MESSAGE_ID = "history-workspace-message";

const DEFAULT_HISTORY_FILTERS = {
  keyword: "",
  audience: "all",
  sortKey: "createdAt",
  sortDir: "desc",
};

const DEFAULT_HISTORY_PAGE_SIZE = 10;

let historyState = {
  loaded: false,
  loading: false,
  searched: false,
  items: [],
  selectedIds: [],
  audienceMap: {},
  filters: { ...DEFAULT_HISTORY_FILTERS },
  currentPage: 1,
  pageSize: DEFAULT_HISTORY_PAGE_SIZE,
  totalCount: 0,
  totalPages: 1,
};

function safeValue(value) {
  return value == null ? "" : String(value);
}

function escapeHtml(value) {
  return String(value ?? "")
    .replaceAll("&", "&amp;")
    .replaceAll("<", "&lt;")
    .replaceAll(">", "&gt;")
    .replaceAll('"', "&quot;")
    .replaceAll("'", "&#39;");
}

function formatDisplay(value, fallback = "-") {
  const text = safeValue(value).trim();
  return text || fallback;
}

function formatDateOnly(value) {
  const text = formatDisplay(value, "-");
  if (text === "-") return text;
  return String(text).slice(0, 10);
}

function audienceLabel(audienceId) {
  switch (audienceId) {
    case "adult":
      return "장년";
    case "young_adult":
      return "청년";
    case "teen":
      return "중고등부";
    case "child":
      return "어린이";
    default:
      return audienceId || "-";
  }
}

function isSelected(historyId) {
  return historyState.selectedIds.includes(historyId);
}

function getAllAudienceIds() {
  return ["adult", "young_adult", "teen", "child"];
}

async function loadAudienceAvailability(historyId) {
  const audienceIds = getAllAudienceIds();
  const available = [];

  for (const audienceId of audienceIds) {
    try {
      const row = await GetHistoryQTJSON(historyId, audienceId);
      if (row && safeValue(row.qtResultJson).trim()) {
        available.push(audienceId);
      }
    } catch (error) {
      // 없는 경우는 무시
    }
  }

  return available;
}

function buildHistoryQuery() {
  return {
    keyword: safeValue(historyState.filters.keyword).trim(),
    audience: safeValue(historyState.filters.audience).trim() || "all",
    sortKey: safeValue(historyState.filters.sortKey).trim() || "createdAt",
    sortDir: safeValue(historyState.filters.sortDir).trim() || "desc",
    page: historyState.currentPage,
    pageSize: historyState.pageSize,
  };
}

async function loadHistoryData() {
  historyState.loading = true;

  try {
    const baseQuery = buildHistoryQuery();

    const totalCount = await CountHistory(baseQuery);
    historyState.totalCount = Number(totalCount || 0);
    historyState.totalPages = Math.max(1, Math.ceil(historyState.totalCount / historyState.pageSize));

    if (historyState.currentPage > historyState.totalPages) {
      historyState.currentPage = historyState.totalPages;
    }
    if (historyState.currentPage < 1) {
      historyState.currentPage = 1;
    }

    const pagedQuery = {
      ...baseQuery,
      page: historyState.currentPage,
      pageSize: historyState.pageSize,
    };

    const list = await ListHistoryPaged(pagedQuery);
    historyState.items = Array.isArray(list) ? list : [];
    historyState.audienceMap = {};

    for (const item of historyState.items) {
      historyState.audienceMap[item.id] = await loadAudienceAvailability(item.id);
    }

    historyState.loaded = true;
  } finally {
    historyState.loading = false;
  }
}

function resetHistoryResults() {
  historyState.loaded = false;
  historyState.searched = false;
  historyState.items = [];
  historyState.selectedIds = [];
  historyState.audienceMap = {};
  historyState.currentPage = 1;
  historyState.totalCount = 0;
  historyState.totalPages = 1;
}

function getAvailableAudienceLabels(historyId) {
  const audienceIds = historyState.audienceMap[historyId] || [];
  return audienceIds.map(audienceLabel);
}

function getFilteredItems() {
  return Array.isArray(historyState.items) ? historyState.items : [];
}

function renderSearchCard() {
  return `
    <section class="card">
      <h3 class="mini-title">검색 조건</h3>

      <div class="form-grid two-column-grid topgap-sm">
        <div class="form-field">
          <label class="form-label">검색어</label>
          <input
            type="text"
            id="history-keyword-input"
            class="input"
            value="${escapeHtml(historyState.filters.keyword)}"
            placeholder="제목 또는 본문 성구"
          />
        </div>

        <div class="form-field">
          <label class="form-label">연령대</label>
          <select id="history-audience-filter" class="input">
            <option value="all" ${historyState.filters.audience === "all" ? "selected" : ""}>전체</option>
            <option value="adult" ${historyState.filters.audience === "adult" ? "selected" : ""}>장년</option>
            <option value="young_adult" ${historyState.filters.audience === "young_adult" ? "selected" : ""}>청년</option>
            <option value="teen" ${historyState.filters.audience === "teen" ? "selected" : ""}>중고등부</option>
            <option value="child" ${historyState.filters.audience === "child" ? "selected" : ""}>어린이</option>
          </select>
        </div>

        <div class="form-field">
          <label class="form-label">정렬 기준</label>
          <select id="history-sort-key" class="input">
            <option value="createdAt" ${historyState.filters.sortKey === "createdAt" ? "selected" : ""}>저장일</option>
            <option value="title" ${historyState.filters.sortKey === "title" ? "selected" : ""}>제목</option>
            <option value="bibleText" ${historyState.filters.sortKey === "bibleText" ? "selected" : ""}>본문 성구</option>
          </select>
        </div>

        <div class="form-field">
          <label class="form-label">정렬 방향</label>
          <select id="history-sort-dir" class="input">
            <option value="desc" ${historyState.filters.sortDir === "desc" ? "selected" : ""}>내림차순</option>
            <option value="asc" ${historyState.filters.sortDir === "asc" ? "selected" : ""}>오름차순</option>
          </select>
        </div>
      </div>

      <div class="half-action-row topgap-sm">
        <button
          type="button"
          class="button"
          id="history-search-btn"
        >
          조회
        </button>

        <button
          type="button"
          class="button-ghost"
          id="history-search-reset-btn"
        >
          초기화
        </button>
      </div>
    </section>
  `;
}

function renderTableBodyContent(items) {
  if (!historyState.searched) {
    return `
      <tr>
        <td colspan="5">
          <div class="history-empty-box">조회 버튼을 눌러 작업 목록을 불러오세요.</div>
        </td>
      </tr>
    `;
  }

  if (!items.length) {
    return `
      <tr>
        <td colspan="5">
          <div class="history-empty-box">조회된 작업이 없습니다.</div>
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

          <td class="history-col-title" title="${escapeHtml(formatDisplay(item.title))}">
            ${escapeHtml(formatDisplay(item.title))}
          </td>

          <td class="history-col-bible" title="${escapeHtml(formatDisplay(item.bibleText))}">
            ${escapeHtml(formatDisplay(item.bibleText))}
          </td>

          <td class="history-col-audience" title="${escapeHtml(availableAudienceLabels.join(", ") || "-")}">
            ${escapeHtml(availableAudienceLabels.join(", ") || "-")}
          </td>

          <td class="history-col-date">
            ${escapeHtml(formatDateOnly(item.createdAt))}
          </td>
        </tr>
      `;
    })
    .join("");
}

function renderHistoryPagination() {
  const canGoPrev = historyState.searched && historyState.currentPage > 1 && !historyState.loading;
  const canGoNext =
    historyState.searched &&
    historyState.currentPage < historyState.totalPages &&
    !historyState.loading;

  return `
    <div class="history-pagination topgap-sm">
      <button
        type="button"
        class="history-page-btn"
        id="history-prev-page-btn"
        ${canGoPrev ? "" : "disabled"}
      >
        이전
      </button>

      <div class="history-page-status">
        ${historyState.currentPage} / ${historyState.totalPages}
        <span class="history-page-total">(${historyState.totalCount}개)</span>
      </div>

      <button
        type="button"
        class="history-page-btn"
        id="history-next-page-btn"
        ${canGoNext ? "" : "disabled"}
      >
        다음
      </button>
    </div>
  `;
}

function renderTableCard() {
  const items = getFilteredItems();
  const totalCount = historyState.searched ? historyState.totalCount : 0;
  const visibleCount = historyState.searched ? items.length : 0;
  const selectedCount = historyState.selectedIds.length;
  const canRework = historyState.searched && selectedCount === 1;
  const canDelete = historyState.searched && selectedCount >= 1;
  const allChecked =
    historyState.searched &&
    items.length > 0 &&
    items.every((item) => isSelected(item.id));

  return `
    <section class="card">
      <div class="card-inline-head">
        <div class="card-inline-head-left">
          <h3 class="mini-title">작업 목록 
            <span class="history-count-text">(${visibleCount}/${totalCount})</span>
          </h3>
        </div>
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
          class="button-ghost"
          id="history-delete-btn"
          ${canDelete ? "" : "disabled"}
        >
          삭제
        </button>
      </div>

      <div class="history-table-wrap topgap-sm">
        <table class="history-table">
          <colgroup>
            <col style="width: 44px;" />
            <col />
            <col style="width: 100px;" />
            <col style="width: 80px;" />
            <col style="width: 100px;" />
          </colgroup>
          <thead>
            <tr>
              <th class="history-table-check">
                <input
                  type="checkbox"
                  id="history-select-all"
                  ${allChecked ? "checked" : ""}
                  ${historyState.searched && items.length > 0 ? "" : "disabled"}
                />
              </th>
              <th>제목</th>
              <th>본문 성구</th>
              <th>연령대</th>
              <th>저장일</th>
            </tr>
          </thead>
          <tbody>
            ${renderTableBodyContent(items)}
          </tbody>
        </table>
      </div>

      ${renderHistoryPagination()}
    </section>
  `;
}

export function renderHistoryWorkspace() {
  return `
    <section class="workspace-step-panel">
      <div class="workspace-header-row">
        <h2 class="workspace-header-title">작업 내역</h2>
      </div>
      <p class="workspace-meta-note">저장된 작업을 검색하고 다시 이어서 작업할 수 있습니다.</p>

      <div id="${HISTORY_MESSAGE_ID}" class="ui-inline-message hidden"></div>

      ${renderSearchCard()}
      ${renderTableCard()}
    </section>
  `;
}

function applySearchFiltersFromDom() {
  historyState.filters.keyword = document.getElementById("history-keyword-input")?.value || "";
  historyState.filters.audience = document.getElementById("history-audience-filter")?.value || "all";
  historyState.filters.sortKey = document.getElementById("history-sort-key")?.value || "createdAt";
  historyState.filters.sortDir = document.getElementById("history-sort-dir")?.value || "desc";
}

function resetSearchFilters() {
  historyState.filters = { ...DEFAULT_HISTORY_FILTERS };
}

async function rerenderHistoryWorkspace() {
  const workspaceRoot = document.querySelector(".main-workspace");
  if (!workspaceRoot) return;

  const { renderMainWorkspace, bindMainWorkspaceEvents } = await import("../mainWorkspace");
  workspaceRoot.innerHTML = renderMainWorkspace();
  bindMainWorkspaceEvents();
}

async function handleSearchSubmit() {
  clearInlineMessage(HISTORY_MESSAGE_ID);

  try {
    applySearchFiltersFromDom();
    historyState.currentPage = 1;
    historyState.selectedIds = [];
    await loadHistoryData();
    historyState.searched = true;
    await rerenderHistoryWorkspace();
  } catch (error) {
    console.error(error);
    setInlineMessage(HISTORY_MESSAGE_ID, error?.message || "작업 내역 조회 중 오류가 발생했습니다.", "error");
  }
}

async function handleSearchReset() {
  clearInlineMessage(HISTORY_MESSAGE_ID);
  resetSearchFilters();
  resetHistoryResults();
  await rerenderHistoryWorkspace();
}

async function handleReworkSelected() {
  clearInlineMessage(HISTORY_MESSAGE_ID);

  if (historyState.selectedIds.length !== 1) {
    setInlineMessage(HISTORY_MESSAGE_ID, "재작업은 1건만 선택할 수 있습니다.", "warning");
    return;
  }

  const historyId = historyState.selectedIds[0];

  try {
    const audienceIds = historyState.audienceMap[historyId] || [];

    if (!audienceIds.length) {
      setInlineMessage(HISTORY_MESSAGE_ID, "재작업할 연령대 정보를 찾을 수 없습니다.", "warning");
      return;
    }

    if (audienceIds.length !== 1) {
      setInlineMessage(HISTORY_MESSAGE_ID, "재작업할 연령대 정보가 올바르지 않습니다.", "error");
      return;
    }

    const audienceId = audienceIds[0];
    const result = await PrepareReworkFromHistory(historyId, audienceId);

    setBasicInfoField("title", result.title || "");
    setBasicInfoField("bibleText", result.bibleText || "");
    setBasicInfoField("hymn", result.hymn || "");
    setBasicInfoField("preacher", result.preacher || "");
    setBasicInfoField("churchName", result.churchName || "");
    setBasicInfoField("sermonDate", result.sermonDate || "");

    appState.historySelected = {
      historyId,
      audienceId,
      tempJsonPath: result.tempJsonPath || "",
    };

    setSelectedMenu(audienceId);
    setAudienceStep(audienceId, "step2");
    showToast("작업을 불러왔습니다.", "success");
    mountAppShell("app");
  } catch (error) {
    console.error(error);
    setInlineMessage(HISTORY_MESSAGE_ID, error?.message || "재작업 준비 중 오류가 발생했습니다.", "error");
  }
}

async function handleDeleteSelected() {
  clearInlineMessage(HISTORY_MESSAGE_ID);

  if (!historyState.selectedIds.length) {
    setInlineMessage(HISTORY_MESSAGE_ID, "삭제할 작업을 선택해 주세요.", "warning");
    return;
  }

  try {
    for (const historyId of historyState.selectedIds) {
      await DeleteHistory(historyId);
    }

    historyState.selectedIds = [];
    await loadHistoryData();
    historyState.searched = true;
    showToast("선택한 작업을 삭제했습니다.", "success");
    await rerenderHistoryWorkspace();
  } catch (error) {
    console.error(error);
    setInlineMessage(HISTORY_MESSAGE_ID, error?.message || "작업 삭제 중 오류가 발생했습니다.", "error");
  }
}

async function handlePageMove(direction) {
  if (!historyState.searched || historyState.loading) return;

  const nextPage =
    direction === "prev"
      ? historyState.currentPage - 1
      : historyState.currentPage + 1;

  if (nextPage < 1 || nextPage > historyState.totalPages) {
    return;
  }

  clearInlineMessage(HISTORY_MESSAGE_ID);

  try {
    historyState.currentPage = nextPage;
    historyState.selectedIds = [];
    await loadHistoryData();
    historyState.searched = true;
    await rerenderHistoryWorkspace();
  } catch (error) {
    console.error(error);
    setInlineMessage(HISTORY_MESSAGE_ID, error?.message || "페이지 이동 중 오류가 발생했습니다.", "error");
  }
}

function bindPaginationEvents() {
  const prevBtn = document.getElementById("history-prev-page-btn");
  const nextBtn = document.getElementById("history-next-page-btn");

  if (prevBtn) {
    prevBtn.addEventListener("click", async () => {
      await handlePageMove("prev");
    });
  }

  if (nextBtn) {
    nextBtn.addEventListener("click", async () => {
      await handlePageMove("next");
    });
  }
}

function bindFilterEvents() {
  const keywordInput = document.getElementById("history-keyword-input");
  const searchBtn = document.getElementById("history-search-btn");
  const resetBtn = document.getElementById("history-search-reset-btn");

  if (keywordInput) {
    keywordInput.addEventListener("keydown", async (event) => {
      if (event.key === "Enter") {
        event.preventDefault();
        await handleSearchSubmit();
      }
    });
  }

  if (searchBtn) {
    searchBtn.addEventListener("click", async () => {
      await handleSearchSubmit();
    });
  }

  if (resetBtn) {
    resetBtn.addEventListener("click", async () => {
      await handleSearchReset();
    });
  }
}

function bindSelectionEvents() {
  const selectAll = document.getElementById("history-select-all");
  if (selectAll) {
    selectAll.addEventListener("change", async () => {
      const items = getFilteredItems();

      if (selectAll.checked) {
        historyState.selectedIds = items.map((item) => item.id);
      } else {
        historyState.selectedIds = [];
      }

      await rerenderHistoryWorkspace();
    });
  }

  const rowChecks = document.querySelectorAll("[data-history-select-id]");
  rowChecks.forEach((checkbox) => {
    checkbox.addEventListener("change", async () => {
      const id = Number(checkbox.dataset.historySelectId);
      if (!id) return;

      if (checkbox.checked) {
        if (!historyState.selectedIds.includes(id)) {
          historyState.selectedIds.push(id);
        }
      } else {
        historyState.selectedIds = historyState.selectedIds.filter((item) => item !== id);
      }

      await rerenderHistoryWorkspace();
    });
  });
}

function bindActionEvents() {
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

export async function bindHistoryWorkspaceEvents() {
  bindFilterEvents();
  bindSelectionEvents();
  bindActionEvents();
  bindPaginationEvents();
}
