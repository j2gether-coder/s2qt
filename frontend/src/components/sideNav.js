import {
  appState,
  setSelectedMenu,
  getMenuLabel,
  isMenuVisible,
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

export function bindSideNavEvents(onMenuChange) {
  const buttons = document.querySelectorAll("[data-menu-id]");

  buttons.forEach((button) => {
    button.addEventListener("click", () => {
      const menuId = button.dataset.menuId;
      setSelectedMenu(menuId);

      if (typeof onMenuChange === "function") {
        onMenuChange(menuId);
      }
    });
  });

  void loadSideNavQR();
}