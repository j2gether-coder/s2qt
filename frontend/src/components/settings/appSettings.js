import { renderSettingsBasicTab, bindSettingsBasicTabEvents } from "./settingsBasicTab";
import { renderSettingsExtraTab, bindSettingsExtraTabEvents } from "./settingsExtraTab";
import { renderSettingsGuideTab, bindSettingsGuideTabEvents } from "./settingsGuideTab";
import { renderSettingsMenuTab, bindSettingsMenuTabEvents } from "./settingsMenuTab";

const SETTINGS_TABS = [
  { id: "basic", label: "기본 정보" },
  { id: "extra", label: "부가 기능" },
  { id: "menu", label: "메뉴 설정" },
  { id: "guide", label: "안내" },
];

let currentSettingsTab = "basic";

export function getCurrentSettingsTab() {
  return currentSettingsTab;
}

export function setCurrentSettingsTab(tabId) {
  const exists = SETTINGS_TABS.some((tab) => tab.id === tabId);
  currentSettingsTab = exists ? tabId : "basic";
}

function getSettingsIntroText() {
  switch (currentSettingsTab) {
    case "extra":
      return "AI와 이메일 전송 등 부가 기능을 설정합니다.";
    case "menu":
      return "사이드 메뉴에 표시할 이름과 표시 여부를 설정합니다.";
    case "guide":
      return "라이선스, 도움말, 사용 가이드를 확인합니다.";
    case "basic":
    default:
      return "보안 기준 정보와 교회/브랜드 기본값을 설정합니다.";
  }
}

function renderSettingsTabs() {
  return `
    <div class="workspace-step-row settings-tab-row">
      ${SETTINGS_TABS.map(
        (tab) => `
          <button
            type="button"
            class="step-tab ${currentSettingsTab === tab.id ? "active" : ""}"
            data-settings-tab="${tab.id}"
          >
            ${tab.label}
          </button>
        `
      ).join("")}
    </div>
  `;
}

function renderCurrentSettingsPanel() {
  switch (currentSettingsTab) {
    case "extra":
      return renderSettingsExtraTab();
    case "menu":
      return renderSettingsMenuTab();
    case "guide":
      return renderSettingsGuideTab();
    case "basic":
    default:
      return renderSettingsBasicTab();
  }
}

function renderSettingsHeaderCard() {
  return `
    <section class="card card-plain">
      <div class="step-badge">환경설정</div>
      <p class="body-note topgap-sm">${getSettingsIntroText()}</p>
    </section>
  `;
}

export function renderAppSettings() {
  return `
    <section class="workspace-step-panel settings-workspace-panel">
      ${renderSettingsHeaderCard()}

      <section class="card">
        ${renderSettingsTabs()}
      </section>

      <div class="workspace-content settings-content">
        ${renderCurrentSettingsPanel()}
      </div>
    </section>
  `;
}

function rerenderSettingsWorkspace() {
  const workspaceRoot = document.querySelector(".main-workspace");
  if (!workspaceRoot) return;

  workspaceRoot.innerHTML = renderAppSettings();
  bindAppSettingsEvents();
}

export function bindAppSettingsEvents() {
  const tabButtons = document.querySelectorAll("[data-settings-tab]");

  tabButtons.forEach((button) => {
    button.addEventListener("click", () => {
      const tabId = button.dataset.settingsTab || "basic";
      setCurrentSettingsTab(tabId);
      rerenderSettingsWorkspace();
    });
  });

  switch (currentSettingsTab) {
    case "extra":
      bindSettingsExtraTabEvents();
      break;
    case "menu":
      bindSettingsMenuTabEvents();
      break;
    case "guide":
      bindSettingsGuideTabEvents();
      break;
    case "basic":
    default:
      bindSettingsBasicTabEvents();
      break;
  }
}