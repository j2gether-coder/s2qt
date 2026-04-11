import { setInlineMessage } from "../../common/uiMessage";

const GUIDE_MESSAGE_ID = "settings-guide-message";

let guideTabState = {
  currentSection: "license", // license | help | guide
};

const GUIDE_SECTIONS = [
  { id: "license", label: "라이선스" },
  { id: "help", label: "도움말" },
  { id: "guide", label: "사용 가이드" },
];

const DEFAULT_GUIDE_CONTENT = {
  license: `
<h3>라이선스</h3>
<p>S2QT는 현재 로컬형 중심으로 구성된 QT 문서 생성 도구입니다.</p>
<p>현재 버전은 프리웨어 성격으로 운영하며, 사용 정책은 향후 변경될 수 있습니다.</p>
<ul>
  <li>기본 기능은 로컬에서 사용할 수 있습니다.</li>
  <li>부가 기능은 추후 확장될 수 있습니다.</li>
  <li>라이선스 및 사용 정책은 버전 변경 시 함께 안내됩니다.</li>
</ul>
  `.trim(),

  help: `
<h3>도움말</h3>
<p>환경설정은 다음 순서로 진행하는 것을 권장합니다.</p>
<ol>
  <li>기본 이메일 설정</li>
  <li>PIN 설정</li>
  <li>부가 기능 설정(AI / SMTP)</li>
  <li>교회/브랜드 설정 확인</li>
</ol>

<p>PIN은 민감정보 보호에 사용되며, 같은 실행 세션에서는 한 번 확인 후 재입력이 생략될 수 있습니다.</p>
  `.trim(),

  guide: `
<h3>사용 가이드</h3>
<ol>
  <li><strong>QT 준비</strong>: 자료 입력과 기본정보 확인</li>
  <li><strong>Step1</strong>: 수동 / 원격 / 로컬 AI 방식에 따라 초안 생성</li>
  <li><strong>Step2</strong>: 검토 및 편집</li>
  <li><strong>Step3</strong>: 문서 생성 및 필요 시 부가 기능 활용</li>
</ol>

<p>작업 내역에서는 저장된 기본정보와 Step1 결과를 불러와 Step2부터 이어서 작업할 수 있습니다.</p>
  `.trim(),
};

export function getCurrentGuideSection() {
  return guideTabState.currentSection;
}

export function setCurrentGuideSection(sectionId) {
  const exists = GUIDE_SECTIONS.some((section) => section.id === sectionId);
  guideTabState.currentSection = exists ? sectionId : "license";
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

function getGuideHtml() {
  return DEFAULT_GUIDE_CONTENT[guideTabState.currentSection] || "<p>내용이 없습니다.</p>";
}

function getGuideSectionDescription() {
  switch (guideTabState.currentSection) {
    case "help":
      return "환경설정과 사용 흐름에 대한 도움말을 확인합니다.";
    case "guide":
      return "S2QT 사용 흐름과 작업 내역 활용 방법을 확인합니다.";
    case "license":
    default:
      return "라이선스와 사용 정책 관련 내용을 확인합니다.";
  }
}

function renderGuideDocumentCard() {
  return `
    <section class="card">
      <h3 class="mini-title">안내 문서</h3>
      <p class="body-note topgap-sm">${getGuideSectionDescription()}</p>

      ${renderGuideSectionTabs()}

      <div class="guide-document-viewer topgap-sm">
        <div class="guide-document-content">
          ${getGuideHtml()}
        </div>
      </div>
    </section>
  `;
}

function renderGuideInfoCard() {
  return `
    <section class="card card-plain">
      <div class="mini-title">안내</div>
      <p class="body-note topgap-sm">
        라이선스, 도움말, 사용 가이드를 확인할 수 있습니다.
      </p>
    </section>
  `;
}

export function renderSettingsGuideTab() {
  return `
    <section class="settings-tab-panel settings-guide-tab">
      <div id="${GUIDE_MESSAGE_ID}" class="ui-inline-message hidden"></div>
      ${renderGuideInfoCard()}
      ${renderGuideDocumentCard()}
    </section>
  `;
}

function rerenderGuideTab() {
  const workspaceRoot = document.querySelector(".main-workspace");
  if (!workspaceRoot) return;

  import("./appSettings").then(({ renderAppSettings, bindAppSettingsEvents }) => {
    workspaceRoot.innerHTML = renderAppSettings();
    bindAppSettingsEvents();
  });
}

export function bindSettingsGuideTabEvents() {
  const sectionButtons = document.querySelectorAll("[data-guide-section]");

  sectionButtons.forEach((button) => {
    button.addEventListener("click", () => {
      const sectionId = button.dataset.guideSection || "license";
      setCurrentGuideSection(sectionId);
      rerenderGuideTab();
    });
  });

  try {
    // 추후 Go 파일 로딩 연결 위치
  } catch (error) {
    console.error(error);
    setInlineMessage(
      GUIDE_MESSAGE_ID,
      error?.message || "안내 문서를 불러오는 중 오류가 발생했습니다.",
      "error"
    );
  }
}