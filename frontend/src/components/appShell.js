import { LoadAppSettingsByGroup } from "../../wailsjs/go/main/App";
import {
  appState,
  setMenuConfig,
  getDefaultMenuConfig,
} from "../state/appState";
import { renderSideNav, bindSideNavEvents } from "./sideNav";
import { renderMainWorkspace, bindMainWorkspaceEvents } from "./mainWorkspace";

function renderHeader() {
  return `
    <header class="app-header">
      <div class="app-header-left">
        <h1 class="app-title">QT 문서 만들기</h1>
        <div class="app-subtitle">
          ${
            appState.source.basicInfo.title
              ? `현재 제목: ${appState.source.basicInfo.title}`
              : "말씀을 묵상으로, 묵상을 삶으로"
          }
        </div>
      </div>
    </header>
  `;
}

function renderLoadingShell() {
  return `
    <div class="app-shell">
      ${renderHeader()}
      <div class="app-body">
        <aside class="side-nav">
          <nav class="side-nav-menu">
            <button class="side-nav-item active" type="button" disabled>
              메뉴 불러오는 중
            </button>
          </nav>
        </aside>

        <main class="main-workspace">
          <section class="workspace-step-panel">
            <section class="card card-plain">
              <div class="mini-title">앱 초기화</div>
              <p class="body-note topgap-sm">메뉴 설정을 불러오는 중입니다.</p>
            </section>
          </section>
        </main>
      </div>
    </div>
  `;
}

async function loadMenuSettingsToAppState() {
  try {
    const items = await LoadAppSettingsByGroup("menu").catch(() => []);
    const list = Array.isArray(items) ? items : [];
    const map = new Map(list.map((item) => [item.key, item.value]));

    const defaultConfig = getDefaultMenuConfig();

    const nextConfig = {};
    Object.keys(defaultConfig).forEach((menuId) => {
      const visibleKey = `menu.${menuId}.visible`;
      const labelKey = `menu.${menuId}.label`;

      const visibleRaw = String(map.get(visibleKey) ?? defaultConfig[menuId].visible).toLowerCase();
      const visible = visibleRaw !== "false";

      const label =
        String(map.get(labelKey) ?? "").trim() || defaultConfig[menuId].label;

      nextConfig[menuId] = {
        visible,
        label,
      };
    });

    setMenuConfig(nextConfig);
  } catch (error) {
    console.error(error);
    setMenuConfig(getDefaultMenuConfig());
  }
}

export function renderAppShell() {
  return `
    <div class="app-shell">
      ${renderHeader()}
      <div class="app-body">
        ${renderSideNav()}
        ${renderMainWorkspace()}
      </div>
    </div>
  `;
}

export async function mountAppShell(rootId = "app") {
  const root = document.getElementById(rootId);
  if (!root) return;

  root.innerHTML = renderLoadingShell();

  await loadMenuSettingsToAppState();

  root.innerHTML = renderAppShell();

  window.scrollTo({ top: 0, left: 0, behavior: "auto" });

  const appBody = document.querySelector(".app-body");
  if (appBody) {
    appBody.scrollTop = 0;
  }

  const mainWorkspace = document.querySelector(".main-workspace");
  if (mainWorkspace) {
    mainWorkspace.scrollTop = 0;
  }

  bindSideNavEvents(() => {
    mountAppShell(rootId);
  });

  bindMainWorkspaceEvents();
}