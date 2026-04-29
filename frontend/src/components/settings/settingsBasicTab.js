import {
  LoadAppSettingsByGroup,
  SaveAppSettings,
  IsPinEnabled,
  GetPinLength,
  SetupPin,
  ChangePin,
  SelectImageFile,
  LoadImageAsDataURI,
  PrepareSiteLogoFile,
  PrepareFooterBrandImage,
} from "../../../wailsjs/go/main/App";
import { showToast, setInlineMessage, clearInlineMessage } from "../../common/uiMessage";

const BASIC_MESSAGE_ID = "settings-basic-message";

const EMAIL_DOMAIN_OPTIONS = [
  { value: "gmail.com", label: "gmail.com" },
  { value: "naver.com", label: "naver.com" },
  { value: "daum.net", label: "daum.net" },
  { value: "hanmail.net", label: "hanmail.net" },
  { value: "kakao.com", label: "kakao.com" },
  { value: "outlook.com", label: "outlook.com" },
  { value: "__custom__", label: "직접 입력" },
];

let basicSettingsState = {
  loaded: false,

  // user
  userEmail: "",

  // pin
  pinEnabled: false,
  pinLength: 6,
  pinMode: "idle", // idle | setup | change
  pinDraft: {
    currentPin: "",
    newPin: "",
    confirmPin: "",
    step: "new", // setup: new|confirm / change: current|new|confirm
  },

  // church/brand
  churchName: "",
  logoPath: "",
  homepageUrl: "",
  footerText: "",
  brandImageIncluded: false,
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

function resetPinDraft(step = "new") {
  basicSettingsState.pinDraft = {
    currentPin: "",
    newPin: "",
    confirmPin: "",
    step,
  };
}

function parseEmailParts(email) {
  const text = safeValue(email).trim();
  if (!text || !text.includes("@")) {
    return {
      localPart: "",
      domainType: "gmail.com",
      customDomain: "",
    };
  }

  const [localPartRaw, domainRaw] = text.split("@");
  const localPart = safeValue(localPartRaw).trim();
  const domain = safeValue(domainRaw).trim().toLowerCase();

  const known = EMAIL_DOMAIN_OPTIONS.find(
    (option) => option.value !== "__custom__" && option.value === domain
  );

  if (known) {
    return {
      localPart,
      domainType: known.value,
      customDomain: "",
    };
  }

  return {
    localPart,
    domainType: "__custom__",
    customDomain: domain,
  };
}

function getEmailDomainOptionsHtml(selectedValue) {
  return EMAIL_DOMAIN_OPTIONS.map(
    (option) => `
      <option value="${option.value}" ${selectedValue === option.value ? "selected" : ""}>
        ${option.label}
      </option>
    `
  ).join("");
}

function buildEmailFromInputs() {
  const localPart = safeValue(document.getElementById("user-email-id-input")?.value).trim();
  const domainType = safeValue(document.getElementById("user-email-domain-select")?.value).trim();
  const customDomain = safeValue(document.getElementById("user-email-domain-input")?.value).trim();

  if (!localPart) return "";

  const domain = domainType === "__custom__" ? customDomain : domainType;
  if (!domain) return "";

  return `${localPart}@${domain}`;
}

function syncEmailDomainInputState() {
  const domainSelect = document.getElementById("user-email-domain-select");
  const customInput = document.getElementById("user-email-domain-input");
  if (!domainSelect || !customInput) return;

  const isCustom = domainSelect.value === "__custom__";
  customInput.disabled = !isCustom;

  if (!isCustom) {
    customInput.value = "";
  }
}

function isProbablyEmail(value) {
  if (!value.trim()) return false;
  return /^[^\s@]+@[^\s@]+\.[^\s@]+$/.test(value.trim());
}

function isProbablyUrl(value) {
  if (!value.trim()) return true;
  return /^https?:\/\/.+/i.test(value.trim());
}

async function loadBasicSettings() {
  const churchItems = ensureArray(await LoadAppSettingsByGroup("church").catch(() => []));
  const userItems = ensureArray(await LoadAppSettingsByGroup("user").catch(() => []));

  const churchMap = new Map(churchItems.map((item) => [item.key, item]));
  const userMap = new Map(userItems.map((item) => [item.key, item]));

  const brandImageIncludedValue =
  churchMap.get("church.brand_image_included")?.value ??
  churchMap.get("church.logo_with_name")?.value ??
  "";

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
  pinDraft: {
    currentPin: "",
    newPin: "",
    confirmPin: "",
    step: "new",
  },

  churchName: safeValue(churchMap.get("church.name")?.value || ""),
  logoPath: safeValue(churchMap.get("church.logo_path")?.value || ""),
  homepageUrl: safeValue(churchMap.get("church.homepage_url")?.value || ""),
  footerText: safeValue(churchMap.get("church.default_footer_text")?.value || ""),
  brandImageIncluded: parseBoolSetting(brandImageIncludedValue),
};
}

function parseBoolSetting(value) {
  const v = safeValue(value).trim().toLowerCase();
  return v === "1" || v === "true" || v === "y" || v === "yes" || v === "on";
}

async function rerenderBasicTab() {
  const { rerenderCurrentSettingsPanel } = await import("./appSettings");
  rerenderCurrentSettingsPanel();
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
  const emailParts = parseEmailParts(basicSettingsState.userEmail);

  return `
    <section class="card">
      <h3 class="mini-title">기본 이메일</h3>
      <p class="body-note topgap-sm">보안 기준 이메일을 설정합니다.</p>

      <div class="form-field topgap-sm">
        <label class="form-label">기본 이메일</label>

        <div class="email-inline-row">
          <input
            type="text"
            id="user-email-id-input"
            class="input"
            value="${escapeHtml(emailParts.localPart)}"
            placeholder="아이디"
          />

          <div class="email-at-mark">@</div>

          <select
            id="user-email-domain-select"
            class="input"
          >
            ${getEmailDomainOptionsHtml(emailParts.domainType)}
          </select>

          <input
            type="text"
            id="user-email-domain-input"
            class="input"
            value="${escapeHtml(emailParts.customDomain)}"
            placeholder="직접 입력"
            ${emailParts.domainType === "__custom__" ? "" : "disabled"}
          />
        </div>

        <div class="field-help-text">
          주소는 선택하거나 직접 입력할 수 있습니다.
        </div>
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

function renderPinSlots(value, length) {
  const filled = safeValue(value);
  return Array.from({ length }, (_, index) => {
    const isFilled = index < filled.length;
    return `
      <span class="${isFilled ? "pin-slot-filled" : "pin-slot-empty"}">
        ${isFilled ? "●" : "○"}
      </span>
    `;
  }).join("");
}

function getPinGuideText() {
  if (basicSettingsState.pinMode === "setup") {
    if (basicSettingsState.pinDraft.step === "new") {
      return `새 PIN ${basicSettingsState.pinLength}자리를 입력해 주세요.`;
    }
    return "PIN을 한 번 더 입력해 주세요.";
  }

  if (basicSettingsState.pinMode === "change") {
    if (basicSettingsState.pinDraft.step === "current") {
      return "현재 PIN을 입력해 주세요.";
    }
    if (basicSettingsState.pinDraft.step === "new") {
      return `새 PIN ${basicSettingsState.pinLength}자리를 입력해 주세요.`;
    }
    return "새 PIN을 한 번 더 입력해 주세요.";
  }

  return "";
}

function getCurrentPinValue() {
  const draft = basicSettingsState.pinDraft;

  if (basicSettingsState.pinMode === "setup") {
    return draft.step === "new" ? draft.newPin : draft.confirmPin;
  }

  if (basicSettingsState.pinMode === "change") {
    if (draft.step === "current") return draft.currentPin;
    if (draft.step === "new") return draft.newPin;
    return draft.confirmPin;
  }

  return "";
}

function renderPinKeypad() {
  const keys = ["1", "2", "3", "4", "5", "6", "7", "8", "9"];

  return `
    <div class="pin-keypad-grid topgap-sm">
      ${keys
        .map(
          (key) => `
            <button
              type="button"
              class="pin-key"
              data-basic-pin-key="${key}"
            >
              ${key}
            </button>
          `
        )
        .join("")}
      <button type="button" class="pin-key" data-basic-pin-clear="true">지움</button>
      <button type="button" class="pin-key" data-basic-pin-key="0">0</button>
      <button type="button" class="pin-key" data-basic-pin-backspace="true">←</button>
    </div>
  `;
}

function renderPinActionArea() {
  if (basicSettingsState.pinMode === "idle") {
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

  return `
    <div class="topgap-sm">
      <div class="hint">${getPinGuideText()}</div>

      <div class="pin-display topgap-sm">
        ${renderPinSlots(getCurrentPinValue(), basicSettingsState.pinLength)}
      </div>

      ${renderPinKeypad()}

      <div class="half-action-row topgap-sm">
        <button type="button" class="button-ghost" id="cancel-pin-edit-btn">취소</button>
        <button type="button" class="button" id="save-pin-progress-btn">
          ${basicSettingsState.pinMode === "setup" ? "PIN 등록" : "PIN 변경"}
        </button>
      </div>
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

      <ul class="settings-guide-list topgap-xs">
        <li>교회명은 "교단명, 교회명" 형식으로 입력하는 것을 권장합니다.</li>
        <li>로고는 밝은 배경에서도 글자와 심볼이 잘 보이는 이미지를 권장합니다.</li>
        <li>흰색 또는 매우 연한 색의 글자가 포함된 로고는 문서 하단에서 잘 보이지 않을 수 있습니다.</li>
        <li>로고에 교회명 또는 브랜드명이 없으면, 교회명을 함께 표시하는 방식을 사용할 수 있습니다.</li>
      </ul>

      <div class="form-grid two-column-grid topgap-sm">
        <div class="form-field">
          <label class="form-label">교단명, 교회명 또는 브랜드명</label>
          <input
            type="text"
            id="church-name-input"
            class="input"
            value="${escapeHtml(basicSettingsState.churchName)}"
            placeholder="예: 대한OOO, OO교회"
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
        <div class="label-inline-row">
          <label class="form-label">로고 파일</label>

          <label class="checkbox-label checkbox-label-compact">
            <input
              type="checkbox"
              id="church-brand-image-included-check"
              ${basicSettingsState.brandImageIncluded ? "checked" : ""}
            />
            <span>로고 이미지에 교회명 또는 브랜드명이 함께 있으면 체크하세요.</span>
          </label>
        </div>

        <div class="half-action-row">
          <button type="button" class="button-ghost" id="select-logo-btn">파일 탐색기</button>
          <button type="button" class="button-ghost" id="preview-logo-btn">로고 미리 보기</button>
        </div>

        <div
          id="church-logo-path-text"
          class="field-help-text topgap-sm"
        >
          ${
            basicSettingsState.logoPath
              ? `선택된 파일: ${escapeHtml(basicSettingsState.logoPath)}`
              : "선택된 파일이 없습니다."
          }
        </div>
      </div>

      <div class="form-field topgap-sm">
        <label class="form-label">기본 하단 문구</label>
        <textarea
          id="church-footer-text-input"
          class="textarea-2rows"
          rows="2"
          placeholder="문서 하단 공통 문구를 입력해 주세요."
        >${escapeHtml(basicSettingsState.footerText)}</textarea>
        <div class="field-help-text">
          권장: 1~2줄, 60자 내외 (예: 말씀을 묵상으로, 묵상을 삶으로)
        </div>
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

  const email = buildEmailFromInputs();

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
        value: email,
        valueType: "text",
        isSecret: false,
        group: "user",
      },
    ]);

    basicSettingsState.userEmail = email;
    showToast("기본 이메일이 저장되었습니다.", "success");
    rerenderBasicTab();
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
  resetPinDraft("new");
  rerenderBasicTab();
}

function handleOpenPinChange() {
  clearInlineMessage(BASIC_MESSAGE_ID);
  basicSettingsState.pinMode = "change";
  resetPinDraft("current");
  rerenderBasicTab();
}

function handleCancelPinEdit() {
  clearInlineMessage(BASIC_MESSAGE_ID);
  basicSettingsState.pinMode = "idle";
  resetPinDraft("new");
  rerenderBasicTab();
}

function applyPinDigit(digit) {
  const draft = basicSettingsState.pinDraft;
  const maxLen = basicSettingsState.pinLength;

  function append(targetKey) {
    if ((draft[targetKey] || "").length >= maxLen) return;
    draft[targetKey] = `${draft[targetKey] || ""}${digit}`;
  }

  if (basicSettingsState.pinMode === "setup") {
    append(draft.step === "new" ? "newPin" : "confirmPin");
    return;
  }

  if (basicSettingsState.pinMode === "change") {
    if (draft.step === "current") append("currentPin");
    else if (draft.step === "new") append("newPin");
    else append("confirmPin");
  }
}

function backspacePinDigit() {
  const draft = basicSettingsState.pinDraft;

  function backspace(targetKey) {
    draft[targetKey] = safeValue(draft[targetKey]).slice(0, -1);
  }

  if (basicSettingsState.pinMode === "setup") {
    backspace(draft.step === "new" ? "newPin" : "confirmPin");
    return;
  }

  if (basicSettingsState.pinMode === "change") {
    if (draft.step === "current") backspace("currentPin");
    else if (draft.step === "new") backspace("newPin");
    else backspace("confirmPin");
  }
}

function clearPinDigits() {
  const draft = basicSettingsState.pinDraft;

  if (basicSettingsState.pinMode === "setup") {
    draft[draft.step === "new" ? "newPin" : "confirmPin"] = "";
    return;
  }

  if (basicSettingsState.pinMode === "change") {
    if (draft.step === "current") draft.currentPin = "";
    else if (draft.step === "new") draft.newPin = "";
    else draft.confirmPin = "";
  }
}

async function handleSavePinProgress() {
  clearInlineMessage(BASIC_MESSAGE_ID);

  const draft = basicSettingsState.pinDraft;

  if (basicSettingsState.pinMode === "setup") {
    const email = buildEmailFromInputs();

    if (!isProbablyEmail(email)) {
      setInlineMessage(
        BASIC_MESSAGE_ID,
        "PIN 등록 전에 기본 이메일을 먼저 올바르게 입력해 주세요.",
        "warning"
      );
      return;
    }

    if (draft.step === "new") {
      if (safeValue(draft.newPin).length !== basicSettingsState.pinLength) {
        setInlineMessage(
          BASIC_MESSAGE_ID,
          `PIN은 ${basicSettingsState.pinLength}자리로 입력해 주세요.`,
          "warning"
        );
        return;
      }

      draft.step = "confirm";
      rerenderBasicTab();
      return;
    }

    if (draft.step === "confirm") {
      if (draft.newPin !== draft.confirmPin) {
        setInlineMessage(BASIC_MESSAGE_ID, "PIN과 확인 PIN이 일치하지 않습니다.", "warning");
        draft.confirmPin = "";
        rerenderBasicTab();
        return;
      }

      try {
        await SaveAppSettings([
          {
            key: "user.email",
            value: email,
            valueType: "text",
            isSecret: false,
            group: "user",
          },
        ]);

        await SetupPin(draft.newPin);

        basicSettingsState.userEmail = email;
        basicSettingsState.pinEnabled = true;
        basicSettingsState.pinMode = "idle";
        resetPinDraft("new");

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

    return;
  }

  if (basicSettingsState.pinMode === "change") {
    if (draft.step === "current") {
      if (safeValue(draft.currentPin).length !== basicSettingsState.pinLength) {
        setInlineMessage(
          BASIC_MESSAGE_ID,
          `현재 PIN은 ${basicSettingsState.pinLength}자리로 입력해 주세요.`,
          "warning"
        );
        return;
      }

      draft.step = "new";
      rerenderBasicTab();
      return;
    }

    if (draft.step === "new") {
      if (safeValue(draft.newPin).length !== basicSettingsState.pinLength) {
        setInlineMessage(
          BASIC_MESSAGE_ID,
          `새 PIN은 ${basicSettingsState.pinLength}자리로 입력해 주세요.`,
          "warning"
        );
        return;
      }

      draft.step = "confirm";
      rerenderBasicTab();
      return;
    }

    if (draft.step === "confirm") {
      if (draft.newPin !== draft.confirmPin) {
        setInlineMessage(BASIC_MESSAGE_ID, "새 PIN과 확인 PIN이 일치하지 않습니다.", "warning");
        draft.confirmPin = "";
        rerenderBasicTab();
        return;
      }

      try {
        await ChangePin(draft.currentPin, draft.newPin);
        basicSettingsState.pinMode = "idle";
        basicSettingsState.pinEnabled = true;
        resetPinDraft("new");
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
  }
}

async function handleSelectLogo() {
  clearInlineMessage(BASIC_MESSAGE_ID);

  try {
    const selectedPath = await SelectImageFile();
    if (!selectedPath) {
      return;
    }

    const preparedPath = await PrepareSiteLogoFile(selectedPath);
    if (!preparedPath) {
      setInlineMessage(
        BASIC_MESSAGE_ID,
        "로고 파일을 앱 내부 경로로 복사하지 못했습니다.",
        "error"
      );
      return;
    }

    basicSettingsState.logoPath = preparedPath;

    const logoPathText = document.getElementById("church-logo-path-text");
    if (logoPathText) {
      logoPathText.textContent = `선택된 파일: ${preparedPath}`;
    }

    showToast("로고 파일이 준비되었습니다.", "success");
  } catch (error) {
    console.error(error);
    setInlineMessage(
      BASIC_MESSAGE_ID,
      error?.message || "파일 선택 중 오류가 발생했습니다.",
      "error"
    );
  }
}

async function handlePreviewLogo() {
  clearInlineMessage(BASIC_MESSAGE_ID);

  const logoPath = safeValue(basicSettingsState.logoPath).trim();
  if (!logoPath) {
    setInlineMessage(BASIC_MESSAGE_ID, "먼저 파일 탐색기에서 로고 파일을 선택해 주세요.", "warning");
    return;
  }

  try {
    const dataURI = await LoadImageAsDataURI(logoPath);
    openLogoPreviewModal(dataURI, logoPath);
  } catch (error) {
    console.error(error);
    setInlineMessage(
      BASIC_MESSAGE_ID,
      error?.message || "이미지 파일을 불러오지 못했습니다.",
      "error"
    );
  }
}

function openLogoPreviewModal(dataURI, captionPath) {
  closeLogoPreviewModal();

  const overlay = document.createElement("div");
  overlay.id = "logo-preview-overlay";
  overlay.className = "logo-preview-overlay";
  overlay.innerHTML = `
    <div class="logo-preview-panel" role="dialog" aria-label="로고 미리 보기">
      <div class="logo-preview-header">
        <div class="logo-preview-title">로고 미리 보기</div>
        <button type="button" class="button-ghost" id="logo-preview-close">닫기</button>
      </div>
      <div class="logo-preview-body">
        <img src="${escapeHtml(dataURI)}" alt="로고 미리 보기" />
      </div>
      <div class="field-help-text">흰색 배경에서도 글자와 심볼이 분명하게 보이는 로고를 권장합니다.</div>
      <div class="logo-preview-caption">${escapeHtml(captionPath)}</div>
    </div>
  `;

  overlay.addEventListener("click", (event) => {
    if (event.target === overlay) {
      closeLogoPreviewModal();
    }
  });

  document.body.appendChild(overlay);

  const closeBtn = document.getElementById("logo-preview-close");
  if (closeBtn) {
    closeBtn.addEventListener("click", closeLogoPreviewModal);
  }

  document.addEventListener("keydown", handleLogoPreviewKey);
}

function closeLogoPreviewModal() {
  const overlay = document.getElementById("logo-preview-overlay");
  if (overlay) {
    overlay.remove();
  }
  document.removeEventListener("keydown", handleLogoPreviewKey);
}

function handleLogoPreviewKey(event) {
  if (event.key === "Escape") {
    closeLogoPreviewModal();
  }
}

async function handleSaveChurchSettings() {
  clearInlineMessage(BASIC_MESSAGE_ID);

  const churchName = safeValue(
    document.getElementById("church-name-input")?.value
  ).trim();

  const homepageUrl = safeValue(
    document.getElementById("church-homepage-url-input")?.value
  ).trim();

  const footerText = safeValue(
    document.getElementById("church-footer-text-input")?.value
  );

  const logoPath = safeValue(basicSettingsState.logoPath).trim();

  const brandImageIncluded =
    !!document.getElementById("church-brand-image-included-check")?.checked;

  if (!isProbablyUrl(homepageUrl)) {
    setInlineMessage(
      BASIC_MESSAGE_ID,
      "홈페이지 URL은 http:// 또는 https:// 형식으로 입력해 주세요.",
      "warning"
    );
    return;
  }

  if (logoPath && !brandImageIncluded && !churchName) {
    setInlineMessage(
      BASIC_MESSAGE_ID,
      "로고 이미지에 교회명 또는 브랜드명이 포함되어 있지 않은 경우, 교회/브랜드명을 입력해야 합니다.",
      "warning"
    );
    return;
  }

  try {
    const items = [
      {
        key: "church.name",
        value: churchName,
        valueType: "text",
        isSecret: false,
        group: "church",
      },
      {
        key: "church.logo_path",
        value: logoPath,
        valueType: "text",
        isSecret: false,
        group: "church",
      },
      {
        key: "church.homepage_url",
        value: homepageUrl,
        valueType: "url",
        isSecret: false,
        group: "church",
      },
      {
        key: "church.default_footer_text",
        value: footerText,
        valueType: "multiline",
        isSecret: false,
        group: "church",
      },
      {
        key: "church.brand_image_included",
        value: brandImageIncluded ? "true" : "false",
        valueType: "boolean",
        isSecret: false,
        group: "church",
      },
    ];

    console.log("[settings-basic] save church settings", items);

    await SaveAppSettings(items);

    if (logoPath) {
      const brandResult = await PrepareFooterBrandImage();
      if (brandResult?.brandFile) {
        basicSettingsState.logoPath = brandResult.brandFile;

        const logoPathText = document.getElementById("church-logo-path-text");
        if (logoPathText) {
          logoPathText.textContent = `선택된 파일: ${brandResult.brandFile}`;
        }
      }
    }

    basicSettingsState.churchName = churchName;
    basicSettingsState.homepageUrl = homepageUrl;
    basicSettingsState.footerText = footerText;
    basicSettingsState.brandImageIncluded = brandImageIncluded;

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

  const emailDomainSelect = document.getElementById("user-email-domain-select");
  if (emailDomainSelect) {
    emailDomainSelect.addEventListener("change", () => {
      syncEmailDomainInputState();
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

  const savePinProgressBtn = document.getElementById("save-pin-progress-btn");
  if (savePinProgressBtn) {
    savePinProgressBtn.addEventListener("click", async () => {
      await handleSavePinProgress();
    });
  }

  const pinKeyButtons = document.querySelectorAll("[data-basic-pin-key]");
  pinKeyButtons.forEach((button) => {
    button.addEventListener("click", () => {
      applyPinDigit(button.dataset.basicPinKey || "");
      rerenderBasicTab();
    });
  });

  const pinBackspaceBtn = document.querySelector("[data-basic-pin-backspace]");
  if (pinBackspaceBtn) {
    pinBackspaceBtn.addEventListener("click", () => {
      backspacePinDigit();
      rerenderBasicTab();
    });
  }

  const pinClearBtn = document.querySelector("[data-basic-pin-clear]");
  if (pinClearBtn) {
    pinClearBtn.addEventListener("click", () => {
      clearPinDigits();
      rerenderBasicTab();
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
