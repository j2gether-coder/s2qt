import {
  LoadAppSettingsByGroup,
  SaveAppSettings,
  IsPinEnabled,
  GetPinLength,
  SetupPin,
  ChangePin,
} from "../../../wailsjs/go/main/App";
import { showToast, setInlineMessage, clearInlineMessage } from "../../common/uiMessage";

const BASIC_MESSAGE_ID = "settings-basic-message";

let basicSettingsState = {
  loaded: false,

  // user
  userEmail: "",

  // pin
  pinEnabled: false,
  pinLength: 6,
  pinMode: "idle", // idle | setup | change

  // church/brand
  churchName: "",
  logoPath: "",
  homepageUrl: "",
  footerText: "",
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

async function loadBasicSettings() {
  const churchItems = ensureArray(await LoadAppSettingsByGroup("church").catch(() => []));
  const userItems = ensureArray(await LoadAppSettingsByGroup("user").catch(() => []));

  const churchMap = new Map(churchItems.map((item) => [item.key, item]));
  const userMap = new Map(userItems.map((item) => [item.key, item]));

  let pinEnabled = false;
  let pinLength = 6;

  try {
    pinEnabled = !!(await IsPinEnabled());
  } catch (error) {
    console.error(error);
    pinEnabled = false;
  }

  try {
    pinLength = await GetPinLength();
    if (pinLength !== 4 && pinLength !== 6) {
      pinLength = 6;
    }
  } catch (error) {
    console.error(error);
    pinLength = 6;
  }

  basicSettingsState = {
    loaded: true,
    userEmail: safeValue(userMap.get("user.email")?.value || ""),
    pinEnabled,
    pinLength,
    pinMode: "idle",

    churchName: safeValue(churchMap.get("church.name")?.value || ""),
    logoPath: safeValue(churchMap.get("church.logo_path")?.value || ""),
    homepageUrl: safeValue(churchMap.get("church.homepage_url")?.value || ""),
    footerText: safeValue(churchMap.get("church.default_footer_text")?.value || ""),
  };
}

function rerenderBasicTab() {
  const workspaceRoot = document.querySelector(".main-workspace");
  if (!workspaceRoot) return;

  import("./appSettings").then(({ renderAppSettings, bindAppSettingsEvents }) => {
    workspaceRoot.innerHTML = renderAppSettings();
    bindAppSettingsEvents();
  });
}

function isProbablyEmail(value) {
  if (!value.trim()) return false;
  return /^[^\s@]+@[^\s@]+\.[^\s@]+$/.test(value.trim());
}

function isProbablyUrl(value) {
  if (!value.trim()) return true;
  return /^https?:\/\/.+/i.test(value.trim());
}

function renderBasicSecurityNoticeCard() {
  return `
    <section class="card card-plain">
      <div class="mini-title">보안 안내</div>
      <p class="body-note topgap-sm">
        이메일과 PIN은 저장된 민감정보(API KEY, 비밀번호 등)를 보호하는 데 사용됩니다.
      </p>
    </section>
  `;
}

function renderEmailCard() {
  return `
    <section class="card">
      <h3 class="mini-title">기본 이메일</h3>
      <p class="body-note topgap-sm">보안 기준 이메일을 설정합니다.</p>

      <div class="topgap-sm" id="${BASIC_MESSAGE_ID}" class="ui-inline-message hidden"></div>

      <div class="form-field topgap-sm">
        <label class="form-label">기본 이메일</label>
        <input
          type="text"
          id="user-email-input"
          class="input"
          value="${escapeHtml(basicSettingsState.userEmail)}"
          placeholder="예: user@example.com"
        />
      </div>

      <div class="row single-action-row topgap-sm">
        <button type="button" class="button" id="save-user-email-btn">이메일 저장</button>
      </div>
    </section>
  `;
}

function renderPinStatusText() {
  return basicSettingsState.pinEnabled ? "PIN 설정됨" : "PIN 미설정";
}

function renderPinActionArea() {
  if (basicSettingsState.pinMode === "setup") {
    return `
      <div class="form-grid two-column-grid topgap-sm">
        <div class="form-field">
          <label class="form-label">새 PIN</label>
          <input
            type="password"
            id="pin-new-input"
            class="input"
            placeholder="${basicSettingsState.pinLength}자리 숫자"
          />
        </div>

        <div class="form-field">
          <label class="form-label">PIN 확인</label>
          <input
            type="password"
            id="pin-confirm-input"
            class="input"
            placeholder="한 번 더 입력"
          />
        </div>
      </div>

      <div class="row topgap-sm">
        <button type="button" class="button-ghost" id="cancel-pin-edit-btn">취소</button>
        <button type="button" class="button" id="save-pin-setup-btn">PIN 등록</button>
      </div>
    `;
  }

  if (basicSettingsState.pinMode === "change") {
    return `
      <div class="form-grid two-column-grid topgap-sm">
        <div class="form-field">
          <label class="form-label">현재 PIN</label>
          <input
            type="password"
            id="pin-current-input"
            class="input"
            placeholder="현재 PIN 입력"
          />
        </div>

        <div class="form-field">
          <label class="form-label">새 PIN</label>
          <input
            type="password"
            id="pin-new-input"
            class="input"
            placeholder="${basicSettingsState.pinLength}자리 숫자"
          />
        </div>

        <div class="form-field">
          <label class="form-label">새 PIN 확인</label>
          <input
            type="password"
            id="pin-confirm-input"
            class="input"
            placeholder="한 번 더 입력"
          />
        </div>
      </div>

      <div class="row topgap-sm">
        <button type="button" class="button-ghost" id="cancel-pin-edit-btn">취소</button>
        <button type="button" class="button" id="save-pin-change-btn">PIN 변경</button>
      </div>
    `;
  }

  return `
    <div class="row single-action-row topgap-sm">
      ${
        basicSettingsState.pinEnabled
          ? `<button type="button" class="button-ghost" id="open-pin-change-btn">PIN 변경</button>`
          : `<button type="button" class="button" id="open-pin-setup-btn">PIN 등록</button>`
      }
    </div>
  `;
}

function renderPinCard() {
  return `
    <section class="card">
      <h3 class="mini-title">PIN 설정</h3>
      <p class="body-note topgap-sm">민감정보 보호에 사용할 PIN을 관리합니다.</p>

      <div class="mode-strip topgap-sm">
        <span class="mode-label">보안 상태</span>
        <span class="mode-value">${renderPinStatusText()}</span>
      </div>

      ${renderPinActionArea()}
    </section>
  `;
}

function renderChurchCard() {
  return `
    <section class="card">
      <h3 class="mini-title">교회/브랜드 설정</h3>
      <p class="body-note topgap-sm">문서에 사용할 기본 정보를 설정합니다.</p>

      <div class="form-grid two-column-grid topgap-sm">
        <div class="form-field">
          <label class="form-label">교회명 또는 브랜드명</label>
          <input
            type="text"
            id="church-name-input"
            class="input"
            value="${escapeHtml(basicSettingsState.churchName)}"
            placeholder="교회/브랜드명을 입력해 주세요."
          />
        </div>

        <div class="form-field">
          <label class="form-label">홈페이지 URL</label>
          <input
            type="text"
            id="church-homepage-url-input"
            class="input"
            value="${escapeHtml(basicSettingsState.homepageUrl)}"
            placeholder="예: https://example.org"
          />
        </div>
      </div>

      <div class="form-field topgap-sm">
        <label class="form-label">로고 경로</label>
        <div class="inline-action-row logo-row">
          <input
            type="text"
            id="church-logo-path-input"
            class="input"
            value="${escapeHtml(basicSettingsState.logoPath)}"
            placeholder="로고 파일 경로를 입력해 주세요."
          />
          <button type="button" class="button-ghost" id="select-logo-btn">파일 탐색기</button>
          <button type="button" class="button-ghost" id="preview-logo-btn">로고 미리 보기</button>
        </div>
      </div>

      <div class="form-field topgap-sm">
        <label class="form-label">기본 footer 문구</label>
        <textarea
          id="church-footer-text-input"
          class="input"
          rows="5"
          placeholder="문서 하단 공통 문구를 입력해 주세요."
        >${escapeHtml(basicSettingsState.footerText)}</textarea>
      </div>

      <div class="row single-action-row topgap-sm">
        <button type="button" class="button" id="save-church-settings-btn">교회/브랜드 설정 저장</button>
      </div>
    </section>
  `;
}

function renderBasicLoadingState() {
  return `
    <section class="card card-plain">
      <div class="mini-title">기본 정보</div>
      <p class="body-note topgap-sm">기본 정보를 불러오는 중입니다.</p>
    </section>
  `;
}

export function renderSettingsBasicTab() {
  if (!basicSettingsState.loaded) {
    return `
      <section class="settings-tab-panel settings-basic-tab">
        <div id="${BASIC_MESSAGE_ID}" class="ui-inline-message hidden"></div>
        ${renderBasicLoadingState()}
      </section>
    `;
  }

  return `
    <section class="settings-tab-panel settings-basic-tab">
      <div id="${BASIC_MESSAGE_ID}" class="ui-inline-message hidden"></div>
      ${renderBasicSecurityNoticeCard()}
      ${renderEmailCard()}
      ${renderPinCard()}
      ${renderChurchCard()}
    </section>
  `;
}

async function handleSaveUserEmail() {
  clearInlineMessage(BASIC_MESSAGE_ID);

  const email = document.getElementById("user-email-input")?.value || "";
  if (!isProbablyEmail(email)) {
    setInlineMessage(
      BASIC_MESSAGE_ID,
      "올바른 이메일 형식으로 입력해 주세요.",
      "warning"
    );
    return;
  }

  try {
    await SaveAppSettings([
      {
        key: "user.email",
        value: email.trim(),
        valueType: "text",
        isSecret: false,
        group: "user",
      },
    ]);

    basicSettingsState.userEmail = email.trim();
    showToast("기본 이메일이 저장되었습니다.", "success");
  } catch (error) {
    console.error(error);
    setInlineMessage(
      BASIC_MESSAGE_ID,
      error?.message || "이메일 저장 중 오류가 발생했습니다.",
      "error"
    );
  }
}

function handleOpenPinSetup() {
  clearInlineMessage(BASIC_MESSAGE_ID);
  basicSettingsState.pinMode = "setup";
  rerenderBasicTab();
}

function handleOpenPinChange() {
  clearInlineMessage(BASIC_MESSAGE_ID);
  basicSettingsState.pinMode = "change";
  rerenderBasicTab();
}

function handleCancelPinEdit() {
  clearInlineMessage(BASIC_MESSAGE_ID);
  basicSettingsState.pinMode = "idle";
  rerenderBasicTab();
}

async function handleSavePinSetup() {
  clearInlineMessage(BASIC_MESSAGE_ID);

  const email = document.getElementById("user-email-input")?.value || "";
  const newPin = document.getElementById("pin-new-input")?.value || "";
  const confirmPin = document.getElementById("pin-confirm-input")?.value || "";

  if (!isProbablyEmail(email)) {
    setInlineMessage(
      BASIC_MESSAGE_ID,
      "PIN 등록 전에 기본 이메일을 먼저 올바르게 입력해 주세요.",
      "warning"
    );
    return;
  }

  if (!newPin.trim() || !confirmPin.trim()) {
    setInlineMessage(BASIC_MESSAGE_ID, "PIN을 모두 입력해 주세요.", "warning");
    return;
  }

  if (newPin !== confirmPin) {
    setInlineMessage(BASIC_MESSAGE_ID, "PIN과 확인 PIN이 일치하지 않습니다.", "warning");
    return;
  }

  try {
    await SaveAppSettings([
      {
        key: "user.email",
        value: email.trim(),
        valueType: "text",
        isSecret: false,
        group: "user",
      },
    ]);

    await SetupPin(newPin);

    basicSettingsState.userEmail = email.trim();
    basicSettingsState.pinEnabled = true;
    basicSettingsState.pinMode = "idle";

    try {
      basicSettingsState.pinLength = await GetPinLength();
    } catch (error) {
      console.error(error);
      basicSettingsState.pinLength = 6;
    }

    showToast("PIN이 등록되었습니다.", "success");
    rerenderBasicTab();
  } catch (error) {
    console.error(error);
    setInlineMessage(
      BASIC_MESSAGE_ID,
      error?.message || "PIN 등록 중 오류가 발생했습니다.",
      "error"
    );
  }
}

async function handleSavePinChange() {
  clearInlineMessage(BASIC_MESSAGE_ID);

  const currentPin = document.getElementById("pin-current-input")?.value || "";
  const newPin = document.getElementById("pin-new-input")?.value || "";
  const confirmPin = document.getElementById("pin-confirm-input")?.value || "";

  if (!currentPin.trim() || !newPin.trim() || !confirmPin.trim()) {
    setInlineMessage(BASIC_MESSAGE_ID, "PIN 항목을 모두 입력해 주세요.", "warning");
    return;
  }

  if (newPin !== confirmPin) {
    setInlineMessage(BASIC_MESSAGE_ID, "새 PIN과 확인 PIN이 일치하지 않습니다.", "warning");
    return;
  }

  try {
    await ChangePin(currentPin, newPin);
    basicSettingsState.pinMode = "idle";
    basicSettingsState.pinEnabled = true;
    showToast("PIN이 변경되었습니다.", "success");
    rerenderBasicTab();
  } catch (error) {
    console.error(error);
    setInlineMessage(
      BASIC_MESSAGE_ID,
      error?.message || "PIN 변경 중 오류가 발생했습니다.",
      "error"
    );
  }
}

function handleSelectLogo() {
  clearInlineMessage(BASIC_MESSAGE_ID);
  setInlineMessage(
    BASIC_MESSAGE_ID,
    "파일 탐색기 기능은 다음 단계에서 연결합니다.",
    "info"
  );
}

function handlePreviewLogo() {
  clearInlineMessage(BASIC_MESSAGE_ID);

  const logoPath = document.getElementById("church-logo-path-input")?.value || "";
  if (!logoPath.trim()) {
    setInlineMessage(BASIC_MESSAGE_ID, "로고 경로를 입력해 주세요.", "warning");
    return;
  }

  setInlineMessage(
    BASIC_MESSAGE_ID,
    "로고 미리 보기 기능은 다음 단계에서 연결합니다.",
    "info"
  );
}

async function handleSaveChurchSettings() {
  clearInlineMessage(BASIC_MESSAGE_ID);

  const homepageUrl = document.getElementById("church-homepage-url-input")?.value || "";
  if (!isProbablyUrl(homepageUrl)) {
    setInlineMessage(
      BASIC_MESSAGE_ID,
      "홈페이지 URL은 http:// 또는 https:// 형식으로 입력해 주세요.",
      "warning"
    );
    return;
  }

  try {
    const items = [
      {
        key: "church.name",
        value: document.getElementById("church-name-input")?.value || "",
        valueType: "text",
        isSecret: false,
        group: "church",
      },
      {
        key: "church.logo_path",
        value: document.getElementById("church-logo-path-input")?.value || "",
        valueType: "text",
        isSecret: false,
        group: "church",
      },
      {
        key: "church.homepage_url",
        value: homepageUrl.trim(),
        valueType: "url",
        isSecret: false,
        group: "church",
      },
      {
        key: "church.default_footer_text",
        value: document.getElementById("church-footer-text-input")?.value || "",
        valueType: "multiline",
        isSecret: false,
        group: "church",
      },
    ];

    await SaveAppSettings(items);

    basicSettingsState.churchName = document.getElementById("church-name-input")?.value || "";
    basicSettingsState.logoPath = document.getElementById("church-logo-path-input")?.value || "";
    basicSettingsState.homepageUrl = homepageUrl.trim();
    basicSettingsState.footerText = document.getElementById("church-footer-text-input")?.value || "";

    showToast("교회/브랜드 설정이 저장되었습니다.", "success");
  } catch (error) {
    console.error(error);
    setInlineMessage(
      BASIC_MESSAGE_ID,
      error?.message || "교회/브랜드 설정 저장 중 오류가 발생했습니다.",
      "error"
    );
  }
}

export async function bindSettingsBasicTabEvents() {
  try {
    if (!basicSettingsState.loaded) {
      await loadBasicSettings();
      rerenderBasicTab();
      return;
    }
  } catch (error) {
    console.error(error);
    setInlineMessage(
      BASIC_MESSAGE_ID,
      error?.message || "기본 정보 불러오기 중 오류가 발생했습니다.",
      "error"
    );
    return;
  }

  const saveUserEmailBtn = document.getElementById("save-user-email-btn");
  if (saveUserEmailBtn) {
    saveUserEmailBtn.addEventListener("click", async () => {
      await handleSaveUserEmail();
    });
  }

  const openPinSetupBtn = document.getElementById("open-pin-setup-btn");
  if (openPinSetupBtn) {
    openPinSetupBtn.addEventListener("click", () => {
      handleOpenPinSetup();
    });
  }

  const openPinChangeBtn = document.getElementById("open-pin-change-btn");
  if (openPinChangeBtn) {
    openPinChangeBtn.addEventListener("click", () => {
      handleOpenPinChange();
    });
  }

  const cancelPinEditBtn = document.getElementById("cancel-pin-edit-btn");
  if (cancelPinEditBtn) {
    cancelPinEditBtn.addEventListener("click", () => {
      handleCancelPinEdit();
    });
  }

  const savePinSetupBtn = document.getElementById("save-pin-setup-btn");
  if (savePinSetupBtn) {
    savePinSetupBtn.addEventListener("click", async () => {
      await handleSavePinSetup();
    });
  }

  const savePinChangeBtn = document.getElementById("save-pin-change-btn");
  if (savePinChangeBtn) {
    savePinChangeBtn.addEventListener("click", async () => {
      await handleSavePinChange();
    });
  }

  const selectLogoBtn = document.getElementById("select-logo-btn");
  if (selectLogoBtn) {
    selectLogoBtn.addEventListener("click", () => {
      handleSelectLogo();
    });
  }

  const previewLogoBtn = document.getElementById("preview-logo-btn");
  if (previewLogoBtn) {
    previewLogoBtn.addEventListener("click", () => {
      handlePreviewLogo();
    });
  }

  const saveChurchSettingsBtn = document.getElementById("save-church-settings-btn");
  if (saveChurchSettingsBtn) {
    saveChurchSettingsBtn.addEventListener("click", async () => {
      await handleSaveChurchSettings();
    });
  }
}