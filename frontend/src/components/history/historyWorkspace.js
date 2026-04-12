import {
  ListHistory,
  GetHistory,
  GetHistoryQTJSON,
  DeleteHistory,
} from "../../../wailsjs/go/main/App";
import { appState, setAudienceStep, setBasicInfoField, setSelectedMenu } from "../../state/appState";
import { showToast, setInlineMessage, clearInlineMessage } from "../../common/uiMessage";
import { mountAppShell } from "../appShell";

const HISTORY_MESSAGE_ID = "history-workspace-message";

let historyState = {
  loaded: false,
  loading: false,
  items: [],
  selectedIds: [],
  audienceMap: {},
  filters: {
    keyword: "",
    audience: "all",
    sortKey: "createdAt",
    sortDir: "desc",
  },
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

async function loadHistoryData() {
  historyState.loading = true;

  const list = await ListHistory();
  historyState.items = Array.isArray(list) ? list : [];
  historyState.audienceMap = {};

  for (const item of historyState.items) {
    historyState.audienceMap[item.id] = await loadAudienceAvailability(item.id);
  }

  historyState.loaded = true;
  historyState.loading = false;
}

function getAvailableAudienceLabels(historyId) {
  const audienceIds = historyState.audienceMap[historyId] || [];
  return audienceIds.map(audienceLabel);
}

function getFilteredItems() {
  const keyword = safeValue(historyState.filters.keyword).trim().toLowerCase();
  const audienceFilter = historyState.filters.audience;
  const sortKey = historyState.filters.sortKey;
  const sortDir = historyState.filters.sortDir;

  let items = [...historyState.items];

  if (keyword) {
    items = items.filter((item) => {
      const title = safeValue(item.title).toLowerCase();
      const bibleText = safeValue(item.bibleText).toLowerCase();
      return title.includes(keyword) || bibleText.includes(keyword);
    });
  }

  if (audienceFilter !== "all") {
    items = items.filter((item) => {
      const audienceIds = historyState.audienceMap[item.id] || [];
      return audienceIds.includes(audienceFilter);
    });
  }

  items.sort((a, b) => {
    const aValue = safeValue(a?.[sortKey]);
    const bValue = safeValue(b?.[sortKey]);

    const compared =
      sortKey === "createdAt"
        ? aValue.localeCompare(bValue)
        : aValue.localeCompare(bValue, "ko");

    return sortDir === "asc" ? compared : -compared;
  });

  return items;
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
    </section>
  `;
}

function renderActionCard() {
  const selectedCount = historyState.selectedIds.length;
  const canRework = selectedCount === 1;
  const canDelete = selectedCount >= 1;

  return `
    <section class="card">
      <div class="half-action-row">
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
    </section>
  `;
}

function renderTableRows(items) {
  if (!items.length) {
    return `
      <tr>
        <td colspan="5">
          <div class="history-empty-box">저장된 작업이 없습니다.</div>
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

function renderTableCard() {
  const items = getFilteredItems();
  const totalCount = historyState.items.length;
  const visibleCount = items.length;
  const allChecked = items.length > 0 && items.every((item) => isSelected(item.id));

  return `
    <section class="card">
      <h3 class="mini-title">작업 목록 
          <span class="history-count-text">(${visibleCount}/${totalCount})</span>
      </h3>

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
                />
              </th>
              <th>제목</th>
              <th>본문 성구</th>
              <th>연령대</th>
              <th>저장일</th>
            </tr>
          </thead>
          <tbody>
            ${renderTableRows(items)}
          </tbody>
        </table>
      </div>
    </section>
  `;
}

export function renderHistoryWorkspace() {
  return `
    <section class="workspace-step-panel">
      <section class="card card-plain">
        <div class="step-badge">작업 내역</div>
        <p class="body-note topgap-sm">저장된 작업을 검색하고 다시 이어서 작업할 수 있습니다.</p>
        <div id="${HISTORY_MESSAGE_ID}" class="ui-inline-message hidden"></div>
      </section>

      ${renderSearchCard()}
      ${renderActionCard()}
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

async function rerenderHistoryWorkspace() {
  const workspaceRoot = document.querySelector(".main-workspace");
  if (!workspaceRoot) return;

  const { renderMainWorkspace, bindMainWorkspaceEvents } = await import("../mainWorkspace");
  workspaceRoot.innerHTML = renderMainWorkspace();
  bindMainWorkspaceEvents();
}

async function handleReworkSelected() {
  clearInlineMessage(HISTORY_MESSAGE_ID);

  if (historyState.selectedIds.length !== 1) {
    setInlineMessage(HISTORY_MESSAGE_ID, "재작업은 1건만 선택할 수 있습니다.", "warning");
    return;
  }

  const historyId = historyState.selectedIds[0];

  try {
    const master = await GetHistory(historyId);
    const audienceIds = historyState.audienceMap[historyId] || [];
    const audienceId = audienceIds[0];

    if (!audienceId) {
      setInlineMessage(HISTORY_MESSAGE_ID, "재작업할 연령대 정보를 찾을 수 없습니다.", "warning");
      return;
    }

    const qtRow = await GetHistoryQTJSON(historyId, audienceId);

    setBasicInfoField("title", master.title || "");
    setBasicInfoField("bibleText", master.bibleText || "");
    setBasicInfoField("hymn", master.hymn || "");
    setBasicInfoField("preacher", master.preacher || "");
    setBasicInfoField("churchName", master.churchName || "");
    setBasicInfoField("sermonDate", master.sermonDate || "");

    appState.historySelected = {
      historyId,
      audienceId,
      step1ResultJson: qtRow.qtResultJson || "",
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
    showToast("선택한 작업을 삭제했습니다.", "success");
    await rerenderHistoryWorkspace();
  } catch (error) {
    console.error(error);
    setInlineMessage(HISTORY_MESSAGE_ID, error?.message || "작업 삭제 중 오류가 발생했습니다.", "error");
  }
}

function bindFilterEvents() {
  const keywordInput = document.getElementById("history-keyword-input");
  const audienceFilter = document.getElementById("history-audience-filter");
  const sortKey = document.getElementById("history-sort-key");
  const sortDir = document.getElementById("history-sort-dir");

  [keywordInput, audienceFilter, sortKey, sortDir].forEach((el) => {
    if (!el) return;
    el.addEventListener("input", async () => {
      applySearchFiltersFromDom();
      await rerenderHistoryWorkspace();
    });
    el.addEventListener("change", async () => {
      applySearchFiltersFromDom();
      await rerenderHistoryWorkspace();
    });
  });
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
  try {
    if (!historyState.loaded && !historyState.loading) {
      await loadHistoryData();
      await rerenderHistoryWorkspace();
      return;
    }
  } catch (error) {
    console.error(error);
    setInlineMessage(HISTORY_MESSAGE_ID, error?.message || "작업 내역을 불러오는 중 오류가 발생했습니다.", "error");
    return;
  }

  bindFilterEvents();
  bindSelectionEvents();
  bindActionEvents();
}