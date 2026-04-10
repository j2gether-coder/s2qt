// frontend/src/common/uiMessage.js

let toastContainer = null;

function ensureToastContainer() {
  if (toastContainer) return toastContainer;

  toastContainer = document.getElementById("ui-toast-container");
  if (toastContainer) return toastContainer;

  toastContainer = document.createElement("div");
  toastContainer.id = "ui-toast-container";
  toastContainer.className = "ui-toast-container";
  document.body.appendChild(toastContainer);
  return toastContainer;
}

export function showToast(message, type = "info", duration = 2200) {
  if (!message) return;

  const container = ensureToastContainer();

  const toast = document.createElement("div");
  toast.className = `ui-toast ${type}`;
  toast.textContent = message;

  container.appendChild(toast);

  requestAnimationFrame(() => {
    toast.classList.add("show");
  });

  window.setTimeout(() => {
    toast.classList.remove("show");
    toast.classList.add("hide");

    window.setTimeout(() => {
      toast.remove();
    }, 200);
  }, duration);
}

export function setInlineMessage(targetId, message = "", type = "info") {
  const target = document.getElementById(targetId);
  if (!target) return;

  if (!message) {
    target.textContent = "";
    target.className = "ui-inline-message hidden";
    return;
  }

  target.textContent = message;
  target.className = `ui-inline-message ${type}`;
}

export function clearInlineMessage(targetId) {
  setInlineMessage(targetId, "", "info");
}