import { LoadGuideDocument } from "../../../wailsjs/go/main/App";
import { setInlineMessage, clearInlineMessage } from "../../common/uiMessage";

const GUIDE_MESSAGE_ID = "settings-guide-message";

let guideTabState = {
  currentSection: "license", // license | guide
  loaded: false,
  loading: false,
  docs: {
    license: "",
    guide: "",
  },
};

const GUIDE_SECTIONS = [
  { id: "license", label: "라이선스" },
  { id: "guide", label: "사용 가이드" },
];

export function getCurrentGuideSection() {
  return guideTabState.currentSection;
}

export function setCurrentGuideSection(sectionId) {
  const exists = GUIDE_SECTIONS.some((section) => section.id === sectionId);
  guideTabState.currentSection = exists ? sectionId : "license";
}

function escapeHtml(value) {
  return String(value ?? "")
    .replaceAll("&", "&amp;")
    .replaceAll("<", "&lt;")
    .replaceAll(">", "&gt;")
    .replaceAll('"', "&quot;")
    .replaceAll("'", "&#39;");
}

function renderMarkdownToHtml(markdown) {
  const text = String(markdown || "").trim();
  if (!text) {
    return `<p>문서 내용이 없습니다.</p>`;
  }

  const lines = text.split(/\r?\n/);
  const html = [];
  let inList = false;
  let inOrderedList = false;

  function closeLists() {
    if (inList) {
      html.push("</ul>");
      inList = false;
    }
    if (inOrderedList) {
      html.push("</ol>");
      inOrderedList = false;
    }
  }

  for (const rawLine of lines) {
    const line = rawLine.trim();

    if (!line) {
      closeLists();
      continue;
    }

    if (line.startsWith("### ")) {
      closeLists();
      html.push(`<h3>${escapeHtml(line.slice(4))}</h3>`);
      continue;
    }

    if (line.startsWith("## ")) {
      closeLists();
      html.push(`<h2>${escapeHtml(line.slice(3))}</h2>`);
      continue;
    }

    if (line.startsWith("# ")) {
      closeLists();
      html.push(`<h1>${escapeHtml(line.slice(2))}</h1>`);
      continue;
    }

    if (/^\d+\.\s+/.test(line)) {
      if (!inOrderedList) {
        closeLists();
        html.push("<ol>");
        inOrderedList = true;
      }
      html.push(`<li>${escapeHtml(line.replace(/^\d+\.\s+/, ""))}</li>`);
      continue;
    }

    if (line.startsWith("- ")) {
      if (!inList) {
        closeLists();
        html.push("<ul>");
        inList = true;
      }
      html.push(`<li>${escapeHtml(line.slice(2))}</li>`);
      continue;
    }

    closeLists();
    html.push(`<p>${escapeHtml(line)}</p>`);
  }

  closeLists();
  return html.join("");
}

function renderGuideSectionTabs() {
  return `
    <div class="workspace-step-row settings-guide-subtab-row">
      ${GUIDE_SECTIONS.map(
        (section) => `
          <button
            type="button"
            class="step-tab ${guideTabState.currentSection === section.id ? "active" : ""}"
            data-guide-section="${section.id}"
          >
            ${section.label}
          </button>
        `
      ).join("")}
    </div>
  `;
}

function getGuideSectionDescription() {
  switch (guideTabState.currentSection) {
    case "guide":
      return "기본 사용 흐름과 주요 기능을 확인합니다.";
    case "license":
    default:
      return "라이선스와 사용 정책 관련 내용을 확인합니다.";
  }
}

function getCurrentGuideHtml() {
  const markdown = guideTabState.docs[guideTabState.currentSection] || "";

  if (guideTabState.loading) {
    return `<p>문서를 불러오는 중입니다.</p>`;
  }

  return renderMarkdownToHtml(markdown);
}

function renderGuideDocumentCard() {
  return `
    <section class="card">
      <h3 class="mini-title">사용 안내</h3>
      <p class="body-note topgap-sm">${getGuideSectionDescription()}</p>

      ${renderGuideSectionTabs()}

      <div class="guide-document-viewer topgap-sm">
        <div class="guide-document-content">
          ${getCurrentGuideHtml()}
        </div>
      </div>
    </section>
  `;
}

export function renderSettingsGuideTab() {
  return `
    <section class="settings-tab-panel settings-guide-tab">
      <div id="${GUIDE_MESSAGE_ID}" class="ui-inline-message hidden"></div>
      ${renderGuideDocumentCard()}
    </section>
  `;
}

async function rerenderGuideTab() {
  const { rerenderCurrentSettingsPanel } = await import("./appSettings");
  rerenderCurrentSettingsPanel();
}

async function loadGuideSection(sectionId) {
  guideTabState.loading = true;
  clearInlineMessage(GUIDE_MESSAGE_ID);
  rerenderGuideTab();

  try {
    const content = await LoadGuideDocument(sectionId);
    guideTabState.docs[sectionId] = String(content || "");
    guideTabState.loaded = true;
    guideTabState.loading = false;
    rerenderGuideTab();
  } catch (error) {
    console.error(error);
    guideTabState.loading = false;
    setInlineMessage(
      GUIDE_MESSAGE_ID,
      error?.message || "안내 문서를 불러오는 중 오류가 발생했습니다.",
      "error"
    );
    rerenderGuideTab();
  }
}

export async function bindSettingsGuideTabEvents() {
  const sectionButtons = document.querySelectorAll("[data-guide-section]");

  sectionButtons.forEach((button) => {
    button.addEventListener("click", async () => {
      const sectionId = button.dataset.guideSection || "license";
      setCurrentGuideSection(sectionId);

      if (!guideTabState.docs[sectionId]) {
        await loadGuideSection(sectionId);
        return;
      }

      rerenderGuideTab();
    });
  });

  if (!guideTabState.docs[guideTabState.currentSection] && !guideTabState.loading) {
    await loadGuideSection(guideTabState.currentSection);
  }
}