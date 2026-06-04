import {
  appState,
  setSelectedMenu,
  getMenuLabel,
  isMenuVisible,
  resetWorkspaceSource,
  resetAudienceProgress,
} from "../state/appState";
import { GetSideNavQRDataURI } from "../../wailsjs/go/main/App";

const MENU_ORDER = [
  { id: "qt_prepare" },
  { id: "adult" },
  { id: "young_adult" },
  { id: "teen" },
  { id: "child" },
  { id: "history" },
  { id: "settings" },
];

function getVisibleMenus() {
  return MENU_ORDER.filter((menu) => isMenuVisible(menu.id)).map((menu) => ({
    id: menu.id,
    label: getMenuLabel(menu.id),
  }));
}

export function renderSideNav() {
  const visibleMenus = getVisibleMenus();

  return `
    <aside class="side-nav">
      <nav class="side-nav-menu">
        ${visibleMenus
          .map(
            (menu) => `
              <button
                class="side-nav-item ${appState.selectedMenu === menu.id ? "active" : ""}"
                type="button"
                data-menu-id="${menu.id}"
              >
                ${menu.label}
              </button>
            `
          )
          .join("")}
      </nav>

      <div class="side-nav-bottom" id="sideNavQrWrap" hidden>
        <img
          class="side-nav-qr"
          id="sideNavQrImg"
          alt="S2QT 안내 QR"
        />
      </div>
    </aside>
  `;
}

async function loadSideNavQR() {
  const wrap = document.getElementById("sideNavQrWrap");
  const img = document.getElementById("sideNavQrImg");

  if (!wrap || !img) return;

  try {
    if (appState.sideNavQRDataURI) {
      img.src = appState.sideNavQRDataURI;
      wrap.hidden = false;
      return;
    }

    const dataURI = await GetSideNavQRDataURI();
    if (!dataURI || !String(dataURI).trim()) {
      wrap.hidden = true;
      return;
    }

    appState.sideNavQRDataURI = dataURI;
    img.src = dataURI;
    wrap.hidden = false;
  } catch (error) {
    console.error(error);
    wrap.hidden = true;
  }
}

function handleSideNavClick(menuId, onMenuChange) {
  if (menuId !== appState.selectedMenu) {
    if (menuId === "qt_prepare") {
      resetWorkspaceSource();
    } else {
      resetAudienceProgress();
    }
  }

  setSelectedMenu(menuId);

  if (typeof onMenuChange === "function") {
    onMenuChange(menuId);
  }
}

export function bindSideNavEvents(onMenuChange) {
  const buttons = document.querySelectorAll("[data-menu-id]");

  buttons.forEach((button) => {
    button.addEventListener("click", () => {
      handleSideNavClick(button.dataset.menuId, onMenuChange);
    });
  });

  void loadSideNavQR();
}

function renderSideNavMenuItems() {
  const visibleMenus = getVisibleMenus();
  return visibleMenus
    .map(
      (menu) => `
        <button
          class="side-nav-item ${appState.selectedMenu === menu.id ? "active" : ""}"
          type="button"
          data-menu-id="${menu.id}"
        >
          ${menu.label}
        </button>
      `
    )
    .join("");
}

export function rerenderSideNavMenu(onMenuChange) {
  const menuRoot = document.querySelector(".side-nav-menu");
  if (!menuRoot) return;

  menuRoot.innerHTML = renderSideNavMenuItems();

  const buttons = menuRoot.querySelectorAll("[data-menu-id]");
  buttons.forEach((button) => {
    button.addEventListener("click", () => {
      handleSideNavClick(button.dataset.menuId, onMenuChange);
    });
  });
}