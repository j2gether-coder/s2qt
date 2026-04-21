const DEFAULT_MENU_CONFIG = {
  qt_prepare: { visible: true, label: "QT 준비" },
  adult: { visible: true, label: "장년 QT" },
  young_adult: { visible: true, label: "청년 QT" },
  teen: { visible: true, label: "중고등부 QT" },
  child: { visible: true, label: "어린이 QT" },
  history: { visible: true, label: "작업 내역" },
  settings: { visible: true, label: "환경 설정" },
};

export const appState = {
  selectedMenu: "qt_prepare", // qt_prepare | adult | young_adult | teen | child | history | settings

  menuConfig: {
    ...DEFAULT_MENU_CONFIG,
  },

  source: {
    sourceType: "video", // video | audio | text
    basicInfo: {
      title: "",
      bibleText: "",
      hymn: "",
      preacher: "",
      churchName: "",
      sermonDate: "",
    },
    transcript: {
      rawText: "",
      cleanedText: "",
    },
    sourceRef: {
      url: "",
      filePath: "",
    },
    sourceStatus: "NOT_READY",
    sourceId: "",
    lastSavedAt: "",
    basicInfoSavedAt: "",
  },

  audienceSteps: {
    adult: "step1",
    young_adult: "step1",
    teen: "step1",
    child: "step1",
  },

  audienceStepStatus: {
    adult: { step1: "idle", step2: "idle", step3: "idle" },
    young_adult: { step1: "idle", step2: "idle", step3: "idle" },
    teen: { step1: "idle", step2: "idle", step3: "idle" },
    child: { step1: "idle", step2: "idle", step3: "idle" },
  },

  historySelected: {
    historyId: null,
    audienceId: "",
    step1ResultJson: "",
  },

  output: {
    htmlFile: "",
    pdfFile: "",
    docxFile: "",
    pptxFile: "",
    pngFile: "",
  },
};

export function setSelectedMenu(menu) {
  appState.selectedMenu = menu;
}

export function setSourceType(type) {
  appState.source.sourceType = type;
}

export function setSourceUrl(url) {
  appState.source.sourceRef.url = url;
}

export function setSourceFilePath(filePath) {
  appState.source.sourceRef.filePath = filePath;
}

export function setRawText(text) {
  appState.source.transcript.rawText = text;
}

export function setCleanedText(text) {
  appState.source.transcript.cleanedText = text;
}

export function setSourceStatus(status) {
  appState.source.sourceStatus = status;
}

export function setBasicInfoField(field, value) {
  if (Object.prototype.hasOwnProperty.call(appState.source.basicInfo, field)) {
    appState.source.basicInfo[field] = value;
  }
}

export function setAudienceStep(audienceId, stepId) {
  if (appState.audienceSteps && Object.prototype.hasOwnProperty.call(appState.audienceSteps, audienceId)) {
    appState.audienceSteps[audienceId] = stepId;
  }
}

export function ensureAudienceStepStatus(audienceId) {
  if (!appState.audienceStepStatus) {
    appState.audienceStepStatus = {};
  }

  if (!appState.audienceStepStatus[audienceId]) {
    appState.audienceStepStatus[audienceId] = {
      step1: "idle",
      step2: "idle",
      step3: "idle",
    };
  }

  return appState.audienceStepStatus[audienceId];
}

export function getAudienceStepStatus(audienceId) {
  return ensureAudienceStepStatus(audienceId);
}

export function setAudienceStepStatus(audienceId, stepId, status) {
  const target = ensureAudienceStepStatus(audienceId);
  if (Object.prototype.hasOwnProperty.call(target, stepId)) {
    target[stepId] = status;
  }
}

export function getDefaultMenuConfig() {
  return {
    ...DEFAULT_MENU_CONFIG,
  };
}

export function resetMenuConfig() {
  appState.menuConfig = {
    ...DEFAULT_MENU_CONFIG,
  };
}

export function setMenuConfig(config = {}) {
  const nextConfig = {
    ...DEFAULT_MENU_CONFIG,
  };

  Object.keys(DEFAULT_MENU_CONFIG).forEach((menuId) => {
    const incoming = config?.[menuId] || {};
    const defaultItem = DEFAULT_MENU_CONFIG[menuId];

    nextConfig[menuId] = {
      visible:
        typeof incoming.visible === "boolean"
          ? incoming.visible
          : defaultItem.visible,
      label:
        String(incoming.label ?? "").trim() || defaultItem.label,
    };
  });

  appState.menuConfig = nextConfig;
}

export function updateMenuConfigItem(menuId, patch = {}) {
  if (!Object.prototype.hasOwnProperty.call(DEFAULT_MENU_CONFIG, menuId)) {
    return;
  }

  const current = appState.menuConfig?.[menuId] || DEFAULT_MENU_CONFIG[menuId];

  appState.menuConfig[menuId] = {
    visible:
      typeof patch.visible === "boolean"
        ? patch.visible
        : current.visible,
    label:
      String(patch.label ?? "").trim() || current.label,
  };
}

export function getMenuConfigItem(menuId) {
  return appState.menuConfig?.[menuId] || DEFAULT_MENU_CONFIG[menuId] || {
    visible: true,
    label: "",
  };
}

export function isMenuVisible(menuId) {
  return !!getMenuConfigItem(menuId).visible;
}

export function getMenuLabel(menuId) {
  return getMenuConfigItem(menuId).label || "";
}

export function getVisibleMenuIds() {
  return Object.keys(DEFAULT_MENU_CONFIG).filter((menuId) => isMenuVisible(menuId));
}

export function getSourceStatusLabel(status) {
  switch (status) {
    case "NOT_READY":
      return "준비 전";
    case "READY":
      return "준비 완료";
    case "RUNNING":
      return "실행 중";
    case "COMPLETED":
      return "완료";
    default:
      return status || "";
  }
}

export function getStepStatusLabel(status) {
  switch (status) {
    case "running":
      return "진행중";
    case "done":
      return "완료";
    case "error":
      return "오류";
    default:
      return "대기";
  }
}