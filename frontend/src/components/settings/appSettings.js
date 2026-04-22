import { renderSettingsBasicTab, bindSettingsBasicTabEvents } from "./settingsBasicTab";
import { renderSettingsExtraTab, bindSettingsExtraTabEvents } from "./settingsExtraTab";
import { renderSettingsGuideTab, bindSettingsGuideTabEvents } from "./settingsGuideTab";
import { renderSettingsMenuTab, bindSettingsMenuTabEvents } from "./settingsMenuTab";
import { renderSettingTemplateTab, bindSettingTemplateTabEvents } from "./settingTemplate";

const SETTINGS_TABS = [
  { id: "basic", label: "기본 정보" },
  { id: "extra", label: "부가 기능" },
  { id: "template", label: "템플릿" },
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
    case "template":
      return "산출물에 적용할 템플릿 사용 여부와 선택 상태를 설정합니다.";
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
    case "template":
      return renderSettingTemplateTab();
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
    <div class="workspace-header-row">
      <h2 class="workspace-header-title">환경 설정</h2>
    </div>
    <p class="workspace-meta-note">${getSettingsIntroText()}</p>
  `;
}

export function renderAppSettings() {
  return `
    <section class="workspace-step-panel settings-workspace-panel">
      ${renderSettingsHeaderCard()}

      ${renderSettingsTabs()}

      <div
        class="workspace-content settings-content"
        id="settingsContentRoot"
        data-settings-tab="${currentSettingsTab}"
      >
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

export function bindCurrentSettingsPanelEvents() {
  switch (currentSettingsTab) {
    case "extra":
      bindSettingsExtraTabEvents();
      break;
    case "template":
      bindSettingTemplateTabEvents();
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

export function rerenderCurrentSettingsPanel() {
  const contentRoot = document.querySelector("#settingsContentRoot");
  if (!contentRoot) return;

  contentRoot.setAttribute("data-settings-tab", currentSettingsTab);
  contentRoot.innerHTML = renderCurrentSettingsPanel();
  bindCurrentSettingsPanelEvents();
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

  bindCurrentSettingsPanelEvents();
}