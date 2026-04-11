import {
  LoadAppSettingsByGroup,
  SaveAppSettings,
  SaveSecretSettingWithPin,
  HasSecretValue,
} from "../../../wailsjs/go/main/App";
import { showToast, setInlineMessage, clearInlineMessage } from "../../common/uiMessage";
import { requirePinIfNeeded } from "../security/securityState";

const EXTRA_MESSAGE_ID = "settings-extra-message";

let extraSettingsState = {
  loaded: false,

  // user
  userEmail: "",

  // AI
  aiMode: "manual", // manual | remote | local
  hasApiKey: false,
  isEditingApiKey: false,

  // SMTP
  smtpEnabled: false,
  smtpHost: "",
  smtpPort: "587",
  smtpUsername: "",
  smtpSecurity: "tls",
  hasSmtpPassword: false,
  isEditingSmtpPassword: false,
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

function boolFromValue(value) {
  return String(value).toLowerCase() === "true";
}

function ensureArray(value) {
  return Array.isArray(value) ? value : [];
}

async function loadExtraSettings() {
  const userItems = ensureArray(await LoadAppSettingsByGroup("user").catch(() => []));
  const llmItems = ensureArray(await LoadAppSettingsByGroup("llm").catch(() => []));
  const smtpItems = ensureArray(await LoadAppSettingsByGroup("smtp").catch(() => []));

  const userMap = new Map(userItems.map((item) => [item.key, item]));
  const llmMap = new Map(llmItems.map((item) => [item.key, item]));
  const smtpMap = new Map(smtpItems.map((item) => [item.key, item]));

  extraSettingsState = {
    loaded: true,

    userEmail: safeValue(userMap.get("user.email")?.value || ""),

    aiMode: safeValue(
      llmMap.get("llm.mode")?.value ||
      llmMap.get("ai.mode")?.value ||
      "manual"
    ),

    hasApiKey: await HasSecretValue("llm.api_key").catch(() => false),
    isEditingApiKey: false,

    smtpEnabled: boolFromValue(smtpMap.get("smtp.enabled")?.value || "false"),
    smtpHost: safeValue(smtpMap.get("smtp.host")?.value || ""),
    smtpPort: safeValue(smtpMap.get("smtp.port")?.value || "587"),
    smtpUsername: safeValue(smtpMap.get("smtp.username")?.value || ""),
    smtpSecurity: safeValue(smtpMap.get("smtp.security")?.value || "tls"),
    hasSmtpPassword: await HasSecretValue("smtp.password").catch(() => false),
    isEditingSmtpPassword: false,
  };
}

function rerenderExtraTab() {
  const workspaceRoot = document.querySelector(".main-workspace");
  if (!workspaceRoot) return;

  import("./appSettings").then(({ renderAppSettings, bindAppSettingsEvents }) => {
    workspaceRoot.innerHTML = renderAppSettings();
    bindAppSettingsEvents();
  });
}

function getAiModeLabel(mode) {
  switch (mode) {
    case "remote":
      return "원격";
    case "local":
      return "로컬";
    case "manual":
    default:
      return "수동";
  }
}

function getSmtpSecurityOptionsHtml(selectedValue) {
  const options = [
    { value: "none", label: "없음" },
    { value: "ssl", label: "SSL" },
    { value: "tls", label: "TLS" },
    { value: "starttls", label: "STARTTLS" },
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

function renderRemoteApiKeyArea() {
  if (extraSettingsState.isEditingApiKey) {
    return `
      <div class="form-field topgap-sm">
        <label class="form-label">원격 AI API KEY</label>
        <input
          type="password"
          id="remote-api-key-input"
          class="input"
          placeholder="API KEY를 입력해 주세요."
        />
      </div>

      <div class="row topgap-sm">
        <button type="button" class="button-ghost" id="cancel-remote-api-key-edit-btn">취소</button>
        <button type="button" class="button" id="save-remote-api-key-btn">API KEY 저장</button>
      </div>
    `;
  }

  return `
    <div class="form-field topgap-sm">
      <label class="form-label">원격 AI API KEY</label>
      <div class="masked-secret-row">
        <div class="masked-secret-value">
          ${extraSettingsState.hasApiKey ? "등록됨" : "미등록"}
        </div>
        <button type="button" class="button-ghost" id="edit-remote-api-key-btn">
          ${extraSettingsState.hasApiKey ? "수정" : "등록"}
        </button>
      </div>
    </div>
  `;
}

function renderAiModeDetail() {
  const aiMode = extraSettingsState.aiMode;

  if (aiMode === "manual") {
    return `
      <div class="hint topgap-sm">
        프롬프트를 복사해서 사용하는 방식입니다.
      </div>
    `;
  }

  if (aiMode === "remote") {
    return `${renderRemoteApiKeyArea()}`;
  }

  if (aiMode === "local") {
    return `
      <div class="hint topgap-sm">
        로컬 AI 방식은 추후 확장 예정입니다.
      </div>
    `;
  }

  return "";
}

function renderAiCard() {
  return `
    <section class="card">
      <h3 class="mini-title">AI 기능</h3>
      <p class="body-note topgap-sm">AI 사용 방식을 설정합니다.</p>

      <div class="mode-strip topgap-sm">
        <span class="mode-label">현재 모드</span>
        <span class="mode-value">${getAiModeLabel(extraSettingsState.aiMode)}</span>
      </div>

      <div class="form-field topgap-sm">
        <label class="form-label">AI 사용 방식</label>
        <div class="radio-inline-group">
          <label class="radio-inline-item">
            <input
              type="radio"
              name="ai-mode"
              value="manual"
              ${extraSettingsState.aiMode === "manual" ? "checked" : ""}
            />
            <span>수동</span>
          </label>

          <label class="radio-inline-item">
            <input
              type="radio"
              name="ai-mode"
              value="remote"
              ${extraSettingsState.aiMode === "remote" ? "checked" : ""}
            />
            <span>원격</span>
          </label>

          <label class="radio-inline-item">
            <input
              type="radio"
              name="ai-mode"
              value="local"
              ${extraSettingsState.aiMode === "local" ? "checked" : ""}
            />
            <span>로컬</span>
          </label>
        </div>
      </div>

      ${renderAiModeDetail()}

      <div class="row single-action-row topgap-sm">
        <button type="button" class="button" id="save-ai-mode-btn">AI 방식 저장</button>
      </div>
    </section>
  `;
}

function renderSmtpPasswordArea() {
  if (extraSettingsState.isEditingSmtpPassword) {
    return `
      <div class="form-field">
        <label class="form-label">SMTP 앱 비밀번호</label>
        <input
          type="password"
          id="smtp-password-input"
          class="input"
          placeholder="SMTP 앱 비밀번호를 입력해 주세요."
        />
        <div class="hint topgap-sm">
          일반 이메일 비밀번호가 아니라 앱 비밀번호를 입력합니다.
        </div>
      </div>

      <div class="row topgap-sm">
        <button type="button" class="button-ghost" id="cancel-smtp-password-edit-btn">취소</button>
        <button type="button" class="button" id="save-smtp-password-btn">비밀번호 저장</button>
      </div>
    `;
  }

  return `
    <div class="form-field">
      <label class="form-label">SMTP 앱 비밀번호</label>
      <div class="masked-secret-row">
        <div class="masked-secret-value">
          ${extraSettingsState.hasSmtpPassword ? "등록됨" : "미등록"}
        </div>
        <button type="button" class="button-ghost" id="edit-smtp-password-btn">
          ${extraSettingsState.hasSmtpPassword ? "수정" : "등록"}
        </button>
      </div>
    </div>
  `;
}

function renderSmtpCard() {
  const hasUserEmail = !!extraSettingsState.userEmail.trim();

  return `
    <section class="card">
      <h3 class="mini-title">SMTP</h3>
      <p class="body-note topgap-sm">Step3에서 산출물을 이메일로 보낼 수 있습니다.</p>

      <div class="form-field topgap-sm">
        <label class="checkbox-row">
          <input
            type="checkbox"
            id="smtp-enabled-checkbox"
            ${extraSettingsState.smtpEnabled ? "checked" : ""}
          />
          <span>SMTP 사용 여부</span>
        </label>
        <div class="hint topgap-sm">
          사용하지 않으면 설정하지 않아도 됩니다.
        </div>
      </div>

      <div class="form-field topgap-sm">
        <label class="form-label">발신 이메일</label>
        ${
          hasUserEmail
            ? `
              <div class="mode-strip">
                <span class="mode-value">${escapeHtml(extraSettingsState.userEmail)}</span>
              </div>
            `
            : `
              <div class="hint">
                기본 정보에서 이메일 주소를 입력해 주세요.
              </div>
            `
        }
      </div>

      <div class="form-grid two-column-grid topgap-sm">
        <div class="form-field">
          <label class="form-label">SMTP 서버</label>
          <input
            type="text"
            id="smtp-host-input"
            class="input"
            value="${escapeHtml(extraSettingsState.smtpHost)}"
            placeholder="예: smtp.gmail.com"
          />
        </div>

        <div class="form-field">
          <label class="form-label">포트</label>
          <input
            type="number"
            id="smtp-port-input"
            class="input"
            value="${escapeHtml(extraSettingsState.smtpPort)}"
            placeholder="587"
          />
        </div>

        <div class="form-field">
          <label class="form-label">보안 방식</label>
          <select id="smtp-security-select" class="input">
            ${getSmtpSecurityOptionsHtml(extraSettingsState.smtpSecurity)}
          </select>
        </div>

        <div class="form-field">
          <label class="form-label">사용자명</label>
          <input
            type="text"
            id="smtp-username-input"
            class="input"
            value="${escapeHtml(extraSettingsState.smtpUsername)}"
            placeholder="SMTP 사용자명을 입력해 주세요."
          />
        </div>
      </div>

      <div class="topgap-sm">
        ${renderSmtpPasswordArea()}
      </div>

      <div class="row single-action-row topgap-sm">
        <button type="button" class="button" id="save-smtp-settings-btn">SMTP 설정 저장</button>
      </div>
    </section>
  `;
}

function renderExtraLoadingState() {
  return `
    <section class="card card-plain">
      <div class="mini-title">부가 기능</div>
      <p class="body-note topgap-sm">AI와 SMTP 설정을 불러오는 중입니다.</p>
    </section>
  `;
}

export function renderSettingsExtraTab() {
  if (!extraSettingsState.loaded) {
    return `
      <section class="settings-tab-panel settings-extra-tab">
        <div id="${EXTRA_MESSAGE_ID}" class="ui-inline-message hidden"></div>
        ${renderExtraLoadingState()}
      </section>
    `;
  }

  return `
    <section class="settings-tab-panel settings-extra-tab">
      <div id="${EXTRA_MESSAGE_ID}" class="ui-inline-message hidden"></div>
      ${renderAiCard()}
      ${renderSmtpCard()}
    </section>
  `;
}

async function handleSaveAiMode() {
  clearInlineMessage(EXTRA_MESSAGE_ID);

  try {
    const aiMode =
      document.querySelector('input[name="ai-mode"]:checked')?.value || "manual";

    await SaveAppSettings([
      {
        key: "ai.mode",
        value: aiMode,
        valueType: "text",
        isSecret: false,
        group: "llm",
      },
      {
        key: "llm.mode",
        value: aiMode,
        valueType: "text",
        isSecret: false,
        group: "llm",
      },
    ]);

    extraSettingsState.aiMode = aiMode;
    showToast("AI 사용 방식이 저장되었습니다.", "success");
    rerenderExtraTab();
  } catch (error) {
    console.error(error);
    setInlineMessage(
      EXTRA_MESSAGE_ID,
      error?.message || "AI 방식 저장 중 오류가 발생했습니다.",
      "error"
    );
  }
}

async function handleEnterApiKeyEditMode() {
  clearInlineMessage(EXTRA_MESSAGE_ID);

  await requirePinIfNeeded({
    reason: "edit_api_key",
    message: "API KEY 수정을 위해 PIN을 입력해 주세요.",
    onSuccess: async () => {
      extraSettingsState.isEditingApiKey = true;
      rerenderExtraTab();
    },
  });
}

async function handleSaveApiKey() {
  clearInlineMessage(EXTRA_MESSAGE_ID);

  const input = document.getElementById("remote-api-key-input");
  const apiKey = input?.value || "";

  if (!apiKey.trim()) {
    setInlineMessage(EXTRA_MESSAGE_ID, "API KEY를 입력해 주세요.", "warning");
    return;
  }

  await requirePinIfNeeded({
    reason: "edit_api_key",
    message: "API KEY 저장을 위해 PIN을 입력해 주세요.",
    onSuccess: async () => {
      try {
        const pinValue = window.__lastVerifiedPin || "";
        if (!pinValue) {
          setInlineMessage(EXTRA_MESSAGE_ID, "PIN 확인 정보가 없습니다. 다시 시도해 주세요.", "error");
          return;
        }

        await SaveSecretSettingWithPin("llm.api_key", apiKey, "password", "llm", pinValue);

        extraSettingsState.hasApiKey = true;
        extraSettingsState.isEditingApiKey = false;

        showToast("API KEY가 저장되었습니다.", "success");
        rerenderExtraTab();
      } catch (error) {
        console.error(error);
        setInlineMessage(
          EXTRA_MESSAGE_ID,
          error?.message || "API KEY 저장 중 오류가 발생했습니다.",
          "error"
        );
      }
    },
  });
}

function handleCancelApiKeyEdit() {
  clearInlineMessage(EXTRA_MESSAGE_ID);
  extraSettingsState.isEditingApiKey = false;
  rerenderExtraTab();
}

async function handleSaveSmtpSettings() {
  clearInlineMessage(EXTRA_MESSAGE_ID);

  try {
    const items = [
      {
        key: "smtp.enabled",
        value: document.getElementById("smtp-enabled-checkbox")?.checked ? "true" : "false",
        valueType: "boolean",
        isSecret: false,
        group: "smtp",
      },
      {
        key: "smtp.host",
        value: (document.getElementById("smtp-host-input")?.value || "").trim(),
        valueType: "text",
        isSecret: false,
        group: "smtp",
      },
      {
        key: "smtp.port",
        value: (document.getElementById("smtp-port-input")?.value || "587").trim(),
        valueType: "number",
        isSecret: false,
        group: "smtp",
      },
      {
        key: "smtp.username",
        value: (document.getElementById("smtp-username-input")?.value || "").trim(),
        valueType: "text",
        isSecret: false,
        group: "smtp",
      },
      {
        key: "smtp.security",
        value: document.getElementById("smtp-security-select")?.value || "tls",
        valueType: "text",
        isSecret: false,
        group: "smtp",
      },
    ];

    await SaveAppSettings(items);

    extraSettingsState.smtpEnabled = document.getElementById("smtp-enabled-checkbox")?.checked || false;
    extraSettingsState.smtpHost = (document.getElementById("smtp-host-input")?.value || "").trim();
    extraSettingsState.smtpPort = (document.getElementById("smtp-port-input")?.value || "587").trim();
    extraSettingsState.smtpUsername = (document.getElementById("smtp-username-input")?.value || "").trim();
    extraSettingsState.smtpSecurity = document.getElementById("smtp-security-select")?.value || "tls";

    showToast("SMTP 설정이 저장되었습니다.", "success");
  } catch (error) {
    console.error(error);
    setInlineMessage(
      EXTRA_MESSAGE_ID,
      error?.message || "SMTP 설정 저장 중 오류가 발생했습니다.",
      "error"
    );
  }
}

async function handleEnterSmtpPasswordEditMode() {
  clearInlineMessage(EXTRA_MESSAGE_ID);

  await requirePinIfNeeded({
    reason: "edit_smtp_password",
    message: "SMTP 앱 비밀번호 수정을 위해 PIN을 입력해 주세요.",
    onSuccess: async () => {
      extraSettingsState.isEditingSmtpPassword = true;
      rerenderExtraTab();
    },
  });
}

async function handleSaveSmtpPassword() {
  clearInlineMessage(EXTRA_MESSAGE_ID);

  const input = document.getElementById("smtp-password-input");
  const password = input?.value || "";

  if (!password.trim()) {
    setInlineMessage(EXTRA_MESSAGE_ID, "SMTP 앱 비밀번호를 입력해 주세요.", "warning");
    return;
  }

  await requirePinIfNeeded({
    reason: "edit_smtp_password",
    message: "SMTP 앱 비밀번호 저장을 위해 PIN을 입력해 주세요.",
    onSuccess: async () => {
      try {
        const pinValue = window.__lastVerifiedPin || "";
        if (!pinValue) {
          setInlineMessage(EXTRA_MESSAGE_ID, "PIN 확인 정보가 없습니다. 다시 시도해 주세요.", "error");
          return;
        }

        await SaveSecretSettingWithPin("smtp.password", password, "password", "smtp", pinValue);

        extraSettingsState.hasSmtpPassword = true;
        extraSettingsState.isEditingSmtpPassword = false;

        showToast("SMTP 앱 비밀번호가 저장되었습니다.", "success");
        rerenderExtraTab();
      } catch (error) {
        console.error(error);
        setInlineMessage(
          EXTRA_MESSAGE_ID,
          error?.message || "SMTP 앱 비밀번호 저장 중 오류가 발생했습니다.",
          "error"
        );
      }
    },
  });
}

function handleCancelSmtpPasswordEdit() {
  clearInlineMessage(EXTRA_MESSAGE_ID);
  extraSettingsState.isEditingSmtpPassword = false;
  rerenderExtraTab();
}

export async function bindSettingsExtraTabEvents() {
  try {
    if (!extraSettingsState.loaded) {
      await loadExtraSettings();
      rerenderExtraTab();
      return;
    }
  } catch (error) {
    console.error(error);
    setInlineMessage(
      EXTRA_MESSAGE_ID,
      error?.message || "부가 기능 설정 불러오기 중 오류가 발생했습니다.",
      "error"
    );
    return;
  }

  const saveAiModeBtn = document.getElementById("save-ai-mode-btn");
  if (saveAiModeBtn) {
    saveAiModeBtn.addEventListener("click", async () => {
      await handleSaveAiMode();
    });
  }

  const editApiKeyBtn = document.getElementById("edit-remote-api-key-btn");
  if (editApiKeyBtn) {
    editApiKeyBtn.addEventListener("click", async () => {
      await handleEnterApiKeyEditMode();
    });
  }

  const cancelApiKeyEditBtn = document.getElementById("cancel-remote-api-key-edit-btn");
  if (cancelApiKeyEditBtn) {
    cancelApiKeyEditBtn.addEventListener("click", () => {
      handleCancelApiKeyEdit();
    });
  }

  const saveApiKeyBtn = document.getElementById("save-remote-api-key-btn");
  if (saveApiKeyBtn) {
    saveApiKeyBtn.addEventListener("click", async () => {
      await handleSaveApiKey();
    });
  }

  const saveSmtpSettingsBtn = document.getElementById("save-smtp-settings-btn");
  if (saveSmtpSettingsBtn) {
    saveSmtpSettingsBtn.addEventListener("click", async () => {
      await handleSaveSmtpSettings();
    });
  }

  const editSmtpPasswordBtn = document.getElementById("edit-smtp-password-btn");
  if (editSmtpPasswordBtn) {
    editSmtpPasswordBtn.addEventListener("click", async () => {
      await handleEnterSmtpPasswordEditMode();
    });
  }

  const cancelSmtpPasswordEditBtn = document.getElementById("cancel-smtp-password-edit-btn");
  if (cancelSmtpPasswordEditBtn) {
    cancelSmtpPasswordEditBtn.addEventListener("click", () => {
      handleCancelSmtpPasswordEdit();
    });
  }

  const saveSmtpPasswordBtn = document.getElementById("save-smtp-password-btn");
  if (saveSmtpPasswordBtn) {
    saveSmtpPasswordBtn.addEventListener("click", async () => {
      await handleSaveSmtpPassword();
    });
  }
}