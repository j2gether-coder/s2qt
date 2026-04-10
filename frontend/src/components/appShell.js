import { appState } from '../state/appState';
import { renderSideNav, bindSideNavEvents } from './sideNav';
import { renderMainWorkspace, bindMainWorkspaceEvents } from './mainWorkspace';

function renderHeader() {
  return `
    <header class="app-header">
      <div class="app-header-left">
        <h1 class="app-title">QT 문서 만들기</h1>
        <div class="app-subtitle">
          ${
            appState.source.basicInfo.title
              ? `현재 제목: ${appState.source.basicInfo.title}`
              : '말씀을 묵상으로, 묵상을 삶으로'
          }
        </div>
      </div>
    </header>
  `;
}

export function renderAppShell() {
  return `
    <div class="app-shell">
      ${renderHeader()}
      <div class="app-body">
        ${renderSideNav()}
        ${renderMainWorkspace()}
      </div>
    </div>
  `;
}

export function mountAppShell(rootId = 'app') {
  const root = document.getElementById(rootId);
  if (!root) return;

  root.innerHTML = renderAppShell();

  window.scrollTo({ top: 0, left: 0, behavior: 'auto' });

  const appBody = document.querySelector('.app-body');
  if (appBody) {
    appBody.scrollTop = 0;
  }

  const mainWorkspace = document.querySelector('.main-workspace');
  if (mainWorkspace) {
    mainWorkspace.scrollTop = 0;
  }

  bindSideNavEvents(() => {
    mountAppShell(rootId);
  });

  bindMainWorkspaceEvents();
}