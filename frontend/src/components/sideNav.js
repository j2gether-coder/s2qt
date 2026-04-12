import {
  appState,
  setSelectedMenu,
  getMenuLabel,
  isMenuVisible,
} from "../state/appState";

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
    </aside>
  `;
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
}