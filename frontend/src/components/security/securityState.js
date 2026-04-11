import { IsPinEnabled } from "../../../wailsjs/go/main/App";
import { openPinModal } from "./pinModal";

export const securityState = {
  unlocked: false,
  unlockedAt: "",
  unlockScope: "session",
};

export function isUnlocked() {
  return securityState.unlocked === true;
}

export function unlockSession() {
  securityState.unlocked = true;
  securityState.unlockedAt = new Date().toISOString();
}

export function lockSession() {
  securityState.unlocked = false;
  securityState.unlockedAt = "";
  delete window.__lastVerifiedPin;
}

export function getUnlockedAt() {
  return securityState.unlockedAt || "";
}

export async function isPinRequired() {
  try {
    const enabled = await IsPinEnabled();
    return !!enabled;
  } catch (error) {
    console.error(error);
    // 보수적으로 PIN 확인이 필요하다고 보지 않고 false 반환
    // 실제 운영에서 원하면 true로 바꿔도 됨
    return false;
  }
}

/**
 * PIN이 설정되어 있고 현재 세션이 잠금 상태이면 PIN 모달을 띄운다.
 * 성공 시 onSuccess를 실행한다.
 *
 * @param {Object} options
 * @param {string} options.reason
 * @param {string} options.message
 * @param {Function} options.onSuccess
 * @param {Function} [options.onCancel]
 */
export async function requirePinIfNeeded({
  reason = "",
  message = "PIN을 입력해 주세요.",
  onSuccess,
  onCancel,
} = {}) {
  if (typeof onSuccess !== "function") {
    throw new Error("onSuccess callback is required");
  }

  const pinEnabled = await isPinRequired();

  // PIN 미설정이면 바로 진행
  if (!pinEnabled) {
    await onSuccess();
    return;
  }

  // 이미 unlock 상태면 바로 진행
  if (isUnlocked()) {
    await onSuccess();
    return;
  }

  openPinModal({
    reason,
    message,
    onSuccess: async () => {
      unlockSession();
      await onSuccess();
    },
    onCancel: async () => {
      if (typeof onCancel === "function") {
        await onCancel();
      }
    },
  });
}