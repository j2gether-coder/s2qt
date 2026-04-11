import { GetPinLength, VerifyPin } from "../../../wailsjs/go/main/App";
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

          <div class="pin-display" id="pin-display">
            ${getMaskedPinDisplay()}
          </div>

          <div id="pin-inline-message" class="ui-inline-message hidden"></div>

          <div class="pin-keypad-grid">
            ${renderDigitButtons()}
          </div>

          <div class="pin-keypad-actions">
            <button type="button" class="pin-key pin-action-key" id="pin-clear-one-btn">지우기</button>
            <button type="button" class="pin-key pin-action-key" id="pin-clear-all-btn">전체삭제</button>
          </div>
        </div>

        <div class="pin-modal-footer">
          <button type="button" class="secondary-button" id="pin-cancel-btn">취소</button>
          <button type="button" class="primary-button" id="pin-submit-btn">확인</button>
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

  loadPinLength()
    .then(() => {
      renderPinModal();
    })
    .catch((error) => {
      console.error(error);
      modalState.maxLength = 6;
      renderPinModal();
    });
}

export function closePinModal() {
  modalState.visible = false;
  modalState.reason = "";
  modalState.message = "";
  modalState.input = "";
  modalState.onSuccess = null;
  modalState.onCancel = null;
  modalState.digits = [];
  clearInlineMessage("pin-inline-message");
  renderPinModal();
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
      renderPinModal();
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
    setInlineMessage(
      "pin-inline-message",
      error?.message || "PIN 확인 중 오류가 발생했습니다.",
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
}