export const appState = {
  selectedMenu: 'qt_prepare', // qt_prepare | adult | young_adult | teen | child
  source: {
    sourceType: 'video', // video | audio | text
    basicInfo: {
      title: '',
      bibleText: '',
      hymn: '',
      preacher: '',
      churchName: '',
      sermonDate: '',
    },
    transcript: {
      rawText: '',
      cleanedText: '',
    },
    sourceRef: {
      url: '',
      filePath: '',
    },
    sourceStatus: 'NOT_READY',
    sourceId: '',
    lastSavedAt: '',
  },

  audienceSteps: {
    adult: 'step1',
    young_adult: 'step1',
    teen: 'step1',
    child: 'step1',
  },

  audienceStepStatus: {
    adult: { step1: 'idle', step2: 'idle', step3: 'idle' },
    young_adult: { step1: 'idle', step2: 'idle', step3: 'idle' },
    teen: { step1: 'idle', step2: 'idle', step3: 'idle' },
    child: { step1: 'idle', step2: 'idle', step3: 'idle' },
  },

  output: {
    htmlFile: '',
    pdfFile: '',
    docxFile: '',
    pptxFile: '',
    pngFile: '',
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
      step1: 'idle',
      step2: 'idle',
      step3: 'idle',
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

export function getMenuLabel(menu) {
  switch (menu) {
    case 'qt_prepare':
      return 'QT 준비';
    case 'adult':
      return '장년 QT';
    case 'young_adult':
      return '청년 QT';
    case 'teen':
      return '중고등부 QT';
    case 'child':
      return '어린이 QT';
    default:
      return '';
  }
}

export function getSourceStatusLabel(status) {
  switch (status) {
    case 'NOT_READY':
      return '준비 전';
    case 'READY':
      return '준비 완료';
    case 'RUNNING':
      return '실행 중';
    case 'COMPLETED':
      return '완료';
    default:
      return status || '';
  }
}

export function getStepStatusLabel(status) {
  switch (status) {
    case 'running':
      return '진행중';
    case 'done':
      return '완료';
    case 'error':
      return '오류';
    default:
      return '대기';
  }
}
