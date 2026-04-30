import {
  LoadAppSettingsByGroup,
  SaveAppSettings,
} from "../../../wailsjs/go/main/App";
import { showToast, setInlineMessage, clearInlineMessage } from "../../common/uiMessage";
import { mountAppShell } from "../appShell";
import { updateMenuConfigItem } from "../../state/appState";
import { rerenderSideNavMenu } from "../sideNav";

const MENU_MESSAGE_ID = "settings-menu-message";

const MENU_ITEMS = [
  { id: "adult", baseLabel: "장년", defaultLabel: "장년 QT" },
  { id: "young_adult", baseLabel: "청년", defaultLabel: "청년 QT" },
  { id: "teen", baseLabel: "중고등부", defaultLabel: "중고등부 QT" },
  { id: "child", baseLabel: "어린이", defaultLabel: "어린이 QT" },
];

let menuSettingsState = {
  loaded: false,
  items: MENU_ITEMS.map((item) => ({
    ...item,
    visible: true,
    label: item.defaultLabel,
  })),
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

function ensureArray(value) {
  return Array.isArray(value) ? value : [];
}

function boolFromValue(value, fallback = true) {
  const text = String(value ?? "").trim().toLowerCase();
  if (text === "true") return true;
  if (text === "false") return false;
  return fallback;
}

async function loadMenuSettings() {
  const menuItems = ensureArray(await LoadAppSettingsByGroup("menu").catch(() => []));
  const menuMap = new Map(menuItems.map((item) => [item.key, item]));

  menuSettingsState = {
    loaded: true,
    items: MENU_ITEMS.map((item) => {
      const visibleKey = `menu.${item.id}.visible`;
      const labelKey = `menu.${item.id}.label`;

      return {
        ...item,
        visible: boolFromValue(menuMap.get(visibleKey)?.value, true),
        label: safeValue(menuMap.get(labelKey)?.value || item.defaultLabel),
      };
    }),
  };
}

function rerenderMenuTab() {
  const contentRoot = document.querySelector("#settingsContentRoot");
  if (!contentRoot) return;
  contentRoot.innerHTML = renderSettingsMenuTab();
  bindSettingsMenuTabEvents();
}

function rerenderSideNav() {
  rerenderSideNavMenu(() => mountAppShell("app"));
}

function syncAppStateMenuConfig(items) {
  items.forEach((item) => {
    updateMenuConfigItem(item.id, {
      visible: item.visible,
      label: item.label,
    });
  });
}

function getCurrentMenuItem(itemId) {
  return (
    menuSettingsState.items.find((item) => item.id === itemId) ||
    MENU_ITEMS.find((item) => item.id === itemId) || {
      id: itemId,
      baseLabel: "",
      defaultLabel: "",
      visible: true,
      label: "",
    }
  );
}

function renderMenuInfoCard() {
  return `
    <section class="card card-plain">
      <div class="mini-title">메뉴 설정</div>
      <p class="body-note topgap-sm">
        사이드 메뉴에 표시할 이름과 표시 여부를 설정합니다.
      </p>
    </section>
  `;
}

function renderMenuTableRows() {
  return MENU_ITEMS.map((item) => {
    const current = getCurrentMenuItem(item.id);

    return `
      <tr>
        <td class="menu-settings-check">
          <input
            type="checkbox"
            id="menu-visible-${item.id}"
            ${current.visible ? "checked" : ""}
          />
        </td>
        <td>
          <div class="menu-base-label">
            <span>${escapeHtml(item.baseLabel)}</span>
            <span class="menu-base-key">(${escapeHtml(item.id)})</span>
          </div>
        </td>
        <td>
          <input
            type="text"
            id="menu-label-${item.id}"
            class="input"
            value="${escapeHtml(current.label)}"
            placeholder="${escapeHtml(item.defaultLabel)}"
          />
        </td>
      </tr>
    `;
  }).join("");
}

function renderMenuSettingsCard() {
  return `
    <section class="card">
      <h3 class="mini-title">사이드 메뉴 설정</h3>
      <p class="body-note topgap-sm">
        내부 연령대 분류는 유지하고, 화면에 표시할 메뉴 이름만 변경합니다.
      </p>

      <div class="menu-settings-table-wrap topgap-sm">
        <table class="menu-settings-table">
          <thead>
            <tr>
              <th class="menu-settings-check">사용</th>
              <th>기본 메뉴</th>
              <th>표시명</th>
            </tr>
          </thead>
          <tbody>
            ${renderMenuTableRows()}
          </tbody>
        </table>
      </div>

      <div class="half-action-row topgap-sm">
        <button type="button" class="button" id="save-menu-settings-btn">
          메뉴 설정 저장
        </button>
        <button type="button" class="button-ghost" id="refresh-menu-settings-btn">
          메뉴 새로 고침
        </button>
      </div>
    </section>
  `;
}

function renderMenuLoadingState() {
  return `
    <section class="card card-plain">
      <div class="mini-title">메뉴 설정</div>
      <p class="body-note topgap-sm">메뉴 설정을 불러오는 중입니다.</p>
    </section>
  `;
}

export function renderSettingsMenuTab() {
  if (!menuSettingsState.loaded) {
    return `
      <section class="settings-tab-panel settings-menu-tab">
        <div id="${MENU_MESSAGE_ID}" class="ui-inline-message hidden"></div>
        ${renderMenuLoadingState()}
      </section>
    `;
  }

  return `
    <section class="settings-tab-panel settings-menu-tab">
      <div id="${MENU_MESSAGE_ID}" class="ui-inline-message hidden"></div>
      ${renderMenuInfoCard()}
      ${renderMenuSettingsCard()}
    </section>
  `;
}

async function handleSaveMenuSettings() {
  clearInlineMessage(MENU_MESSAGE_ID);

  const nextItems = MENU_ITEMS.map((item) => {
    const visible = !!document.getElementById(`menu-visible-${item.id}`)?.checked;
    const labelInput = document.getElementById(`menu-label-${item.id}`)?.value || "";
    const label = safeValue(labelInput).trim() || item.defaultLabel;

    return {
      ...item,
      visible,
      label,
    };
  });

  const hasVisibleItem = nextItems.some((item) => item.visible);
  if (!hasVisibleItem) {
    setInlineMessage(
      MENU_MESSAGE_ID,
      "사이드 메뉴는 최소 1개 이상 표시되어야 합니다.",
      "warning"
    );
    return;
  }

  try {
    const saveItems = [];

    nextItems.forEach((item) => {
      saveItems.push(
        {
          key: `menu.${item.id}.visible`,
          value: item.visible ? "true" : "false",
          valueType: "boolean",
          isSecret: false,
          group: "menu",
        },
        {
          key: `menu.${item.id}.label`,
          value: item.label,
          valueType: "text",
          isSecret: false,
          group: "menu",
        }
      );
    });

    await SaveAppSettings(saveItems);
    menuSettingsState.items = nextItems;
    syncAppStateMenuConfig(nextItems);

    nextItems.forEach((item) => {
      const labelInput = document.getElementById(`menu-label-${item.id}`);
      if (labelInput && labelInput.value !== item.label) {
        labelInput.value = item.label;
      }
    });

    showToast("메뉴 설정이 저장되었습니다.", "success");
    rerenderSideNav();
  } catch (error) {
    console.error(error);
    setInlineMessage(
      MENU_MESSAGE_ID,
      error?.message || "메뉴 설정 저장 중 오류가 발생했습니다.",
      "error"
    );
  }
}

async function handleRefreshMenu() {
  clearInlineMessage(MENU_MESSAGE_ID);

  try {
    await loadMenuSettings();
    syncAppStateMenuConfig(menuSettingsState.items);

    rerenderSideNav();
    rerenderMenuTab();
    showToast("메뉴를 새로 고쳤습니다.", "success");
  } catch (error) {
    console.error(error);
    setInlineMessage(
      MENU_MESSAGE_ID,
      error?.message || "메뉴 새로 고침 중 오류가 발생했습니다.",
      "error"
    );
  }
}

export async function bindSettingsMenuTabEvents() {
  try {
    if (!menuSettingsState.loaded) {
      await loadMenuSettings();
      rerenderMenuTab();
      return;
    }
  } catch (error) {
    console.error(error);
    setInlineMessage(
      MENU_MESSAGE_ID,
      error?.message || "메뉴 설정 불러오기 중 오류가 발생했습니다.",
      "error"
    );
    return;
  }

  const saveButton = document.getElementById("save-menu-settings-btn");
  if (saveButton) {
    saveButton.addEventListener("click", async () => {
      await handleSaveMenuSettings();
    });
  }

  const refreshButton = document.getElementById("refresh-menu-settings-btn");
  if (refreshButton) {
    refreshButton.addEventListener("click", () => {
      handleRefreshMenu();
    });
  }
}