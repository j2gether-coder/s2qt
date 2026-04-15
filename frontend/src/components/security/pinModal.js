import {
  GetPinLength,
  VerifyPin,
  GetPinLockoutStatus,
  ResetPinLockout,
} from "../../../wailsjs/go/main/App";
import { setInlineMessage, clearInlineMessage } from "../../common/uiMessage";

const modalState = {
  visible: false,
  reason: "",
  message: "",
  input: "",
  maxLength: 6,
  digits: [],
  onSuccess: null,
  onCancel: null,
  lockoutRemaining: 0,
  lockoutTimerId: null,
  permanentLock: false,
};

function getModalRoot() {
  return document.getElementById("pin-modal-root");
}

function ensureModalRoot() {
  let root = getModalRoot();
  if (root) return root;

  root = document.createElement("div");
  root.id = "pin-modal-root";
  document.body.appendChild(root);
  return root;
}

function shuffleArray(arr) {
  const copy = [...arr];
  for (let i = copy.length - 1; i > 0; i -= 1) {
    const j = Math.floor(Math.random() * (i + 1));
    [copy[i], copy[j]] = [copy[j], copy[i]];
  }
  return copy;
}

function buildRandomDigits() {
  return shuffleArray(["1", "2", "3", "4", "5", "6", "7", "8", "9", "0"]);
}

function escapeHtml(value = "") {
  return String(value)
    .replace(/&/g, "&amp;")
    .replace(/</g, "&lt;")
    .replace(/>/g, "&gt;")
    .replace(/"/g, "&quot;")
    .replace(/'/g, "&#039;");
}

function getMaskedPinDisplay() {
  const count = modalState.input.length;
  if (count <= 0) {
    return '<span class="pin-slot-empty">●</span>'.repeat(modalState.maxLength);
  }

  let html = "";
  for (let i = 0; i < modalState.maxLength; i += 1) {
    if (i < count) {
      html += '<span class="pin-slot-filled">●</span>';
    } else {
      html += '<span class="pin-slot-empty">●</span>';
    }
  }
  return html;
}

function renderDigitButtons() {
  return modalState.digits
    .map(
      (digit) => `
        <button
          type="button"
          class="pin-key pin-digit-key"
          data-pin-digit="${digit}"
        >
          ${digit}
        </button>
      `
    )
    .join("");
}

export function renderPinModal() {
  const root = ensureModalRoot();

  if (!modalState.visible) {
    root.innerHTML = "";
    return;
  }

  const locked = modalState.lockoutRemaining > 0 || modalState.permanentLock;
  const disabledAttr = locked ? "disabled" : "";

  let lockoutBanner = "";
  if (modalState.permanentLock) {
    lockoutBanner = `
      <div class="pin-lockout-banner pin-lockout-permanent">
        PIN이 영구 잠금되었습니다. 보안 설정을 초기화해야 합니다.<br>
        <small>초기화 시 저장된 SMTP / LLM 비밀값은 삭제되며 재입력이 필요합니다.</small>
      </div>
    `;
  } else if (modalState.lockoutRemaining > 0) {
    lockoutBanner = `
      <div class="pin-lockout-banner pin-lockout-temporary">
        PIN 잠금 중입니다. <strong id="pin-lockout-countdown">${modalState.lockoutRemaining}</strong>초 후 재시도 가능합니다.
      </div>
    `;
  }

  const footerButtons = modalState.permanentLock
    ? `
        <button type="button" class="secondary-button" id="pin-cancel-btn">닫기</button>
        <button type="button" class="primary-button" id="pin-reset-btn">보안 설정 초기화</button>
      `
    : `
        <button type="button" class="link-button" id="pin-forgot-btn">PIN을 잊으셨나요?</button>
        <button type="button" class="secondary-button" id="pin-cancel-btn">취소</button>
        <button type="button" class="primary-button" id="pin-submit-btn" ${disabledAttr}>확인</button>
      `;

  root.innerHTML = `
    <div class="pin-modal-overlay">
      <div class="pin-modal-panel" role="dialog" aria-modal="true" aria-labelledby="pin-modal-title">
        <div class="pin-modal-header">
          <h3 id="pin-modal-title" class="pin-modal-title">보안 확인</h3>
        </div>

        <div class="pin-modal-body">
          <div class="pin-modal-message">
            ${escapeHtml(modalState.message || "PIN을 입력해 주세요.")}
          </div>

          ${lockoutBanner}

          <div class="pin-display" id="pin-display">
            ${getMaskedPinDisplay()}
          </div>

          <div id="pin-inline-message" class="ui-inline-message hidden"></div>

          <div class="pin-keypad-grid">
            ${renderDigitButtons()}
          </div>

          <div class="pin-keypad-actions">
            <button type="button" class="pin-key pin-action-key" id="pin-clear-one-btn" ${disabledAttr}>지우기</button>
            <button type="button" class="pin-key pin-action-key" id="pin-clear-all-btn" ${disabledAttr}>전체삭제</button>
          </div>
        </div>

        <div class="pin-modal-footer">
          ${footerButtons}
        </div>
      </div>
    </div>
  `;

  bindPinModalEvents();
}

export function openPinModal({
  reason = "",
  message = "PIN을 입력해 주세요.",
  onSuccess,
  onCancel,
} = {}) {
  modalState.visible = true;
  modalState.reason = reason;
  modalState.message = message;
  modalState.input = "";
  modalState.onSuccess = onSuccess || null;
  modalState.onCancel = onCancel || null;
  modalState.digits = buildRandomDigits();
  modalState.lockoutRemaining = 0;
  modalState.permanentLock = false;
  stopLockoutTimer();

  Promise.all([loadPinLength(), refreshLockoutStatus()])
    .catch((error) => {
      console.error(error);
      modalState.maxLength = 6;
    })
    .finally(() => {
      renderPinModal();
      if (modalState.lockoutRemaining > 0) {
        startLockoutTimer();
      }
    });
}

export function closePinModal() {
  stopLockoutTimer();
  modalState.visible = false;
  modalState.reason = "";
  modalState.message = "";
  modalState.input = "";
  modalState.onSuccess = null;
  modalState.onCancel = null;
  modalState.digits = [];
  modalState.lockoutRemaining = 0;
  modalState.permanentLock = false;
  clearInlineMessage("pin-inline-message");
  renderPinModal();
}

async function refreshLockoutStatus() {
  try {
    const status = await GetPinLockoutStatus();
    if (!status) return;
    modalState.permanentLock = !!status.permanent;
    const remaining = Number(status.remainingSecs) || 0;
    modalState.lockoutRemaining = remaining > 0 ? remaining : 0;
  } catch (error) {
    console.error(error);
  }
}

function stopLockoutTimer() {
  if (modalState.lockoutTimerId) {
    clearInterval(modalState.lockoutTimerId);
    modalState.lockoutTimerId = null;
  }
}

function startLockoutTimer() {
  stopLockoutTimer();
  modalState.lockoutTimerId = setInterval(() => {
    if (modalState.lockoutRemaining > 0) {
      modalState.lockoutRemaining -= 1;
    }

    const countdownEl = document.getElementById("pin-lockout-countdown");
    if (countdownEl) {
      countdownEl.textContent = String(Math.max(modalState.lockoutRemaining, 0));
    }

    if (modalState.lockoutRemaining <= 0) {
      stopLockoutTimer();
      renderPinModal();
    }
  }, 1000);
}

async function loadPinLength() {
  try {
    const pinLength = await GetPinLength();
    modalState.maxLength = pinLength === 4 || pinLength === 6 ? pinLength : 6;
  } catch (error) {
    console.error(error);
    modalState.maxLength = 6;
  }
}

function updatePinDisplay() {
  const display = document.getElementById("pin-display");
  if (display) {
    display.innerHTML = getMaskedPinDisplay();
  }
}

export function appendPinDigit(digit) {
  if (!modalState.visible) return;
  if (modalState.permanentLock || modalState.lockoutRemaining > 0) return;
  if (modalState.input.length >= modalState.maxLength) return;

  modalState.input += String(digit);
  clearInlineMessage("pin-inline-message");
  updatePinDisplay();
}

export function removeLastPinDigit() {
  if (!modalState.visible) return;
  if (!modalState.input.length) return;

  modalState.input = modalState.input.slice(0, -1);
  clearInlineMessage("pin-inline-message");
  updatePinDisplay();
}

export function clearPinInput() {
  if (!modalState.visible) return;

  modalState.input = "";
  clearInlineMessage("pin-inline-message");
  updatePinDisplay();
}

export async function submitPin() {
  clearInlineMessage("pin-inline-message");

  if (modalState.permanentLock || modalState.lockoutRemaining > 0) {
    return;
  }

  if (modalState.input.length !== modalState.maxLength) {
    setInlineMessage(
      "pin-inline-message",
      `PIN ${modalState.maxLength}자리를 입력해 주세요.`,
      "warning"
    );
    return;
  }

  try {
    const ok = await VerifyPin(modalState.input);

    if (!ok) {
      setInlineMessage("pin-inline-message", "PIN이 올바르지 않습니다.", "error");
      clearPinInput();
      modalState.digits = buildRandomDigits();
      await refreshLockoutStatus();
      renderPinModal();
      if (modalState.lockoutRemaining > 0) {
        startLockoutTimer();
      }
      return;
    }

    // 마지막으로 검증된 PIN을 세션 메모리에 저장
    window.__lastVerifiedPin = modalState.input;

    const successHandler = modalState.onSuccess;
    closePinModal();

    if (typeof successHandler === "function") {
      await successHandler();
    }
  } catch (error) {
    console.error(error);
    const msg = error?.message || "PIN 확인 중 오류가 발생했습니다.";
    setInlineMessage("pin-inline-message", msg, "error");
    clearPinInput();
    modalState.digits = buildRandomDigits();
    await refreshLockoutStatus();
    renderPinModal();
    if (modalState.lockoutRemaining > 0) {
      startLockoutTimer();
    }
  }
}

function confirmSecurityReset() {
  const primary = window.confirm(
    [
      "[보안 설정 초기화]",
      "",
      "이 작업은 되돌릴 수 없습니다.",
      "",
      " - 저장된 SMTP 비밀번호, LLM API 키 등",
      "   암호화된 모든 비밀값이 삭제됩니다.",
      " - PIN 설정이 해제됩니다.",
      " - QT 히스토리와 일반 설정(교회 정보 등)은 유지됩니다.",
      "",
      "계속 진행하시겠습니까?",
    ].join("\n")
  );
  if (!primary) return false;

  const confirmText = window.prompt(
    '확인을 위해 "초기화" 를 정확히 입력해 주세요.'
  );
  return (confirmText || "").trim() === "초기화";
}

async function handleResetLockout() {
  if (!confirmSecurityReset()) return;

  try {
    await ResetPinLockout(true);
    window.alert(
      "보안 설정이 초기화되었습니다.\n설정 화면에서 PIN을 다시 등록한 뒤 SMTP / LLM 정보를 재입력해 주세요."
    );
    const cancelHandler = modalState.onCancel;
    closePinModal();
    if (typeof cancelHandler === "function") {
      await cancelHandler();
    }
  } catch (error) {
    console.error(error);
    setInlineMessage(
      "pin-inline-message",
      error?.message || "보안 설정 초기화 중 오류가 발생했습니다.",
      "error"
    );
  }
}

export function bindPinModalEvents() {
  const digitButtons = document.querySelectorAll("[data-pin-digit]");
  digitButtons.forEach((btn) => {
    btn.addEventListener("click", () => {
      appendPinDigit(btn.dataset.pinDigit || "");
    });
  });

  const clearOneBtn = document.getElementById("pin-clear-one-btn");
  if (clearOneBtn) {
    clearOneBtn.addEventListener("click", () => {
      removeLastPinDigit();
    });
  }

  const clearAllBtn = document.getElementById("pin-clear-all-btn");
  if (clearAllBtn) {
    clearAllBtn.addEventListener("click", () => {
      clearPinInput();
    });
  }

  const cancelBtn = document.getElementById("pin-cancel-btn");
  if (cancelBtn) {
    cancelBtn.addEventListener("click", async () => {
      const cancelHandler = modalState.onCancel;
      closePinModal();

      if (typeof cancelHandler === "function") {
        await cancelHandler();
      }
    });
  }

  const submitBtn = document.getElementById("pin-submit-btn");
  if (submitBtn) {
    submitBtn.addEventListener("click", async () => {
      await submitPin();
    });
  }

  const resetBtn = document.getElementById("pin-reset-btn");
  if (resetBtn) {
    resetBtn.addEventListener("click", async () => {
      await handleResetLockout();
    });
  }

  const forgotBtn = document.getElementById("pin-forgot-btn");
  if (forgotBtn) {
    forgotBtn.addEventListener("click", async () => {
      await handleResetLockout();
    });
  }
}