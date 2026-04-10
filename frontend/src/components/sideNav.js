import { appState, setSelectedMenu } from '../state/appState';

const MENUS = [
  { id: 'qt_prepare', label: 'QT 준비' },
  { id: 'adult', label: '장년 QT' },
  { id: 'young_adult', label: '청년 QT' },
  { id: 'teen', label: '중고등부 QT' },
  { id: 'child', label: '어린이 QT' },
];

export function renderSideNav() {
  return `
    <aside class="side-nav">
      <nav class="side-nav-menu">
        ${MENUS.map(
          (menu) => `
            <button
              class="side-nav-item ${appState.selectedMenu === menu.id ? 'active' : ''}"
              type="button"
              data-menu-id="${menu.id}"
            >
              ${menu.label}
            </button>
          `
        ).join('')}
      </nav>
    </aside>
  `;
}

export function bindSideNavEvents(onMenuChange) {
  const buttons = document.querySelectorAll('[data-menu-id]');
  buttons.forEach((button) => {
    button.addEventListener('click', () => {
      const menuId = button.dataset.menuId;
      setSelectedMenu(menuId);
      if (typeof onMenuChange === 'function') {
        onMenuChange(menuId);
      }
    });
  });
}