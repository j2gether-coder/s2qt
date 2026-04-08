import {
  ClassicEditor,
  Essentials,
  Paragraph,
  Bold,
  Italic,
  Heading,
  List
} from 'ckeditor5';

import { appState, updateEditor, updateOutput, updateStatus } from '../state/appState';
import { refreshShellStatus } from '../components/appShell';

let summaryEditorInstance = null;
let messageEditorInstance = null;
let reflectionEditorInstance = null;
let prayerEditorInstance = null;

function escapeHtml(value) {
  return String(value ?? '')
    .replaceAll('&', '&amp;')
    .replaceAll('<', '&lt;')
    .replaceAll('>', '&gt;')
    .replaceAll('"', '&quot;')
    .replaceAll("'", '&#39;');
}

function stripFooterHtml(html) {
  if (!html) return '';

  return String(html)
    .replace(/<div[^>]*class="[^"]*qt-footer[^"]*"[^>]*>[\s\S]*?<\/div>\s*$/i, '')
    .replace(/<div[^>]*class="[^"]*qt-footer-line[^"]*"[^>]*>[\s\S]*?<\/div>[\s\S]*$/i, '')
    .replace(/<div[^>]*class="[^"]*qt-footer-text[^"]*"[^>]*>[\s\S]*$/i, '')
    .trim();
}

function stripPrayerWrapper(html) {
  if (!html) return '';

  let cleaned = String(html).trim();
  cleaned = cleaned.replace(/^<div[^>]*class="[^"]*qt-box[^"]*qt-prayer[^"]*"[^>]*>\s*/i, '');
  cleaned = cleaned.replace(/\s*<\/div>\s*$/i, '');

  return cleaned.trim();
}

function findSectionStart(source, titleText) {
  if (!source) return -1;

  const escaped = titleText
    .replace(/[.*+?^${}()|[\]\\]/g, '\\$&')
    .replace(/\s+/g, '\\s*');

  const re = new RegExp(
    `<h2[^>]*class="[^"]*qt-section-title[^"]*"[^>]*>\\s*${escaped}\\s*<\\/h2>`,
    'i'
  );

  const match = re.exec(source);
  return match ? match.index : -1;
}

function findPrayerStart(source) {
  if (!source) return -1;

  const re = /<div[^>]*class="[^"]*qt-box[^"]*qt-prayer[^"]*"[^>]*>/i;
  const match = re.exec(source);
  return match ? match.index : -1;
}

function findFooterStart(source) {
  if (!source) return -1;

  const re = /<div[^>]*class="[^"]*qt-footer[^"]*"[^>]*>/i;
  const match = re.exec(source);
  return match ? match.index : -1;
}

function extractSectionByMarkers(source, startIndex, endIndex) {
  if (!source || startIndex < 0) return '';

  const end = endIndex >= 0 ? endIndex : source.length;
  return source.substring(startIndex, end).trim();
}

function extractPrayerInner(source) {
  if (!source) return '';

  const prayerStart = findPrayerStart(source);
  if (prayerStart < 0) return '';

  const footerStart = findFooterStart(source);
  const prayerBlock = extractSectionByMarkers(source, prayerStart, footerStart);

  return stripPrayerWrapper(stripFooterHtml(prayerBlock));
}

function extractStyleBlock(source) {
  if (!source) return '';

  const match = String(source).match(/<style[\s\S]*?<\/style>/i);
  return match ? match[0].trim() : '';
}

function resolveHymnText() {
  const metaHymn = appState.meta?.hymn?.trim();
  if (metaHymn) {
    return metaHymn;
  }

  const draftHtml = appState.draft?.draftHtml || '';

  const patterns = [
    /찬송\s*:\s*([^<\n]+)/i,
    /찬송가\s*:\s*([^<\n]+)/i,
    /찬양\s*:\s*([^<\n]+)/i
  ];

  for (const pattern of patterns) {
    const match = draftHtml.match(pattern);
    if (match?.[1]?.trim()) {
      return match[1].trim();
    }
  }

  return '';
}

function initEditorFromDraftIfEmpty() {
  const { draftHtml } = appState.draft;
  const { editor } = appState;

  const hasEditorContent =
    editor.summaryHtml?.trim() ||
    editor.messageHtml?.trim() ||
    editor.reflectionHtml?.trim() ||
    editor.prayerHtml?.trim();

  if (hasEditorContent || !draftHtml?.trim()) {
    return;
  }

  const summaryStart = findSectionStart(draftHtml, '🌿 말씀의 창: 본문 요약');
  const messageStart = findSectionStart(draftHtml, '✨ 오늘의 메시지');
  const reflectionStart = findSectionStart(draftHtml, '🔍 깊은 묵상과 적용');
  const prayerStart = findPrayerStart(draftHtml);
  const footerStart = findFooterStart(draftHtml);

  const summaryHtml = extractSectionByMarkers(draftHtml, summaryStart, messageStart);
  const messageHtml = extractSectionByMarkers(draftHtml, messageStart, reflectionStart);
  const reflectionHtml = extractSectionByMarkers(draftHtml, reflectionStart, prayerStart);
  const prayerHtml = extractPrayerInner(draftHtml);

  updateEditor({
    summaryHtml: stripFooterHtml(summaryHtml),
    messageHtml: stripFooterHtml(messageHtml),
    reflectionHtml: stripFooterHtml(reflectionHtml),
    prayerHtml: stripPrayerWrapper(stripFooterHtml(prayerHtml))
  });

  console.log('draftHtml exists:', !!draftHtml, draftHtml?.length || 0);
  console.log('summary start:', summaryStart);
  console.log('message start:', messageStart);
  console.log('reflection start:', reflectionStart);
  console.log('prayer start:', prayerStart);
  console.log('footer start:', footerStart);
}

function buildEditedHtml() {
  const { meta, editor } = appState;

  const hymnText = resolveHymnText();
  const draftStyle = extractStyleBlock(appState.draft?.draftHtml || '');

  const summaryHtml =
    stripFooterHtml(editor.summaryHtml?.trim() || '') || '<div class="qt-body"><p></p></div>';

  const messageHtml = stripFooterHtml(editor.messageHtml?.trim() || '');
  const reflectionHtml = stripFooterHtml(editor.reflectionHtml?.trim() || '');
  const prayerHtml = stripPrayerWrapper(stripFooterHtml(editor.prayerHtml?.trim() || ''));

  return `
${draftStyle}
<div class="qt-wrap">
  <div class="qt-main">
    <h1 class="qt-title">[QT] ${escapeHtml(meta.title || '')}</h1>

    <div class="qt-subbox">
      본문: ${escapeHtml(meta.bibleText || '')}<br>
      찬송: ${escapeHtml(hymnText || '')}
    </div>

    ${summaryHtml}

    ${messageHtml}

    ${reflectionHtml}

    <div class="qt-box qt-prayer">
      ${prayerHtml}
    </div>
  </div>

  <div class="qt-footer">
    <div class="qt-footer-line"></div>
    <div class="qt-footer-text">말씀을 묵상으로, 묵상을 삶으로</div>
  </div>
</div>
  `.trim();
}

function syncEditReady() {
  const ready = !!(
    appState.editor.summaryHtml?.trim() &&
    appState.editor.messageHtml?.trim() &&
    appState.editor.reflectionHtml?.trim() &&
    appState.editor.prayerHtml?.trim()
  );

  updateStatus({ editReady: ready });
  refreshShellStatus();
}

function syncPreview() {
  const finalHtml = buildEditedHtml();

  updateEditor({ editedHtml: finalHtml });
  updateOutput({ finalHtml });
  syncEditReady();
}

function getEditorConfig() {
  return {
    licenseKey: 'GPL',
    plugins: [Essentials, Paragraph, Bold, Italic, Heading, List],
    toolbar: {
      items: [
        'heading',
        '|',
        'bold',
        'italic',
        '|',
        'bulletedList',
        'numberedList',
        '|',
        'undo',
        'redo'
      ],
      shouldNotGroupWhenFull: false
    },
    placeholder: '내용을 자연스럽게 다듬어 주세요.'
  };
}

async function createOneEditor(elementId, initialData, onChange) {
  const el = document.getElementById(elementId);
  if (!el) return null;

  const editor = await ClassicEditor.create(el, getEditorConfig());
  editor.setData(initialData || '');

  editor.model.document.on('change:data', () => {
    onChange(editor.getData());
  });

  return editor;
}

function updateSummaryHtml(html) {
  updateEditor({ summaryHtml: stripFooterHtml(html) });
  syncPreview();
}

function updateMessageHtml(html) {
  updateEditor({ messageHtml: stripFooterHtml(html) });
  syncPreview();
}

function updateReflectionHtml(html) {
  updateEditor({ reflectionHtml: stripFooterHtml(html) });
  syncPreview();
}

function updatePrayerHtml(html) {
  updateEditor({ prayerHtml: stripPrayerWrapper(stripFooterHtml(html)) });
  syncPreview();
}

function destroyEditorInstance(editor) {
  if (!editor) return Promise.resolve();
  return editor.destroy().catch(() => {});
}

async function destroyAllEditors() {
  await destroyEditorInstance(summaryEditorInstance);
  await destroyEditorInstance(messageEditorInstance);
  await destroyEditorInstance(reflectionEditorInstance);
  await destroyEditorInstance(prayerEditorInstance);

  summaryEditorInstance = null;
  messageEditorInstance = null;
  reflectionEditorInstance = null;
  prayerEditorInstance = null;
}

function handleOpenPreview() {
  const previewHtml = appState.output.finalHtml || appState.editor.editedHtml || buildEditedHtml();
  const previewWindow = window.open('', '_blank', 'width=900,height=900');

  if (!previewWindow) {
    alert('미리보기 창을 열 수 없습니다.');
    return;
  }

  previewWindow.document.open();
  previewWindow.document.write(`
    <!doctype html>
    <html lang="ko">
      <head>
        <meta charset="utf-8" />
        <title>QT 미리보기</title>
        <style>
          body {
            margin: 0;
            padding: 24px;
            font-family: Arial, sans-serif;
            background: #f6f8fb;
            color: #1f2937;
          }
          .preview-wrap {
            max-width: 860px;
            margin: 0 auto;
            background: #ffffff;
            border-radius: 14px;
            padding: 24px;
            box-shadow: 0 6px 18px rgba(0,0,0,0.08);
          }
        </style>
      </head>
      <body>
        <div class="preview-wrap">
          ${previewHtml}
        </div>
      </body>
    </html>
  `);
  previewWindow.document.close();
}

export function renderEditorStep() {
  return `
    <section class="card editor-guide-card">
      <div class="editor-step-badge">Step 3. 편집</div>
      <h2 class="editor-step-title">전사 초안을 읽기 좋게 다듬어 주세요</h2>
      <p class="editor-step-desc">
        Step2에서 생성된 초안이 각 섹션에 자동 배치됩니다. 툴바는 간단히 유지하고, 본문 흐름과 문장 표현을 중심으로 수정해 주세요.
      </p>
    </section>

    <section class="card editor-card">
      <div class="editor-card-head">
        <h3 class="mini-title">말씀의 창: 본문 요약</h3>
        <div class="editor-card-help">핵심 요약만 간결하게 정리</div>
      </div>
      <div id="summaryEditor" class="ck-editor-host"></div>
    </section>

    <section class="card editor-card">
      <div class="editor-card-head">
        <h3 class="mini-title">오늘의 메시지</h3>
        <div class="editor-card-help">가장 전달하고 싶은 메시지 중심</div>
      </div>
      <div id="messageEditor" class="ck-editor-host ck-editor-host-lg"></div>
    </section>

    <section class="card editor-card">
      <div class="editor-card-head">
        <h3 class="mini-title">깊은 묵상과 적용</h3>
        <div class="editor-card-help">적용 포인트가 보이도록 정리</div>
      </div>
      <div id="reflectionEditor" class="ck-editor-host"></div>
    </section>

    <section class="card editor-card">
      <div class="editor-card-head">
        <h3 class="mini-title">오늘의 기도</h3>
        <div class="editor-card-help">자연스럽고 담백한 기도문으로 마무리</div>
      </div>
      <div id="prayerEditor" class="ck-editor-host"></div>

      <div class="row topgap-sm editor-action-row">
        <button id="openPreviewBtn" class="button button-ghost" type="button">편집결과 미리보기</button>
      </div>
    </section>
  `;
}

export async function bindEditorStepEvents() {
  initEditorFromDraftIfEmpty();
  syncPreview();

  await destroyAllEditors();

  summaryEditorInstance = await createOneEditor(
    'summaryEditor',
    appState.editor.summaryHtml || '',
    updateSummaryHtml
  );

  messageEditorInstance = await createOneEditor(
    'messageEditor',
    appState.editor.messageHtml || '',
    updateMessageHtml
  );

  reflectionEditorInstance = await createOneEditor(
    'reflectionEditor',
    appState.editor.reflectionHtml || '',
    updateReflectionHtml
  );

  prayerEditorInstance = await createOneEditor(
    'prayerEditor',
    appState.editor.prayerHtml || '',
    updatePrayerHtml
  );

  const openPreviewBtn = document.getElementById('openPreviewBtn');
  if (openPreviewBtn) {
    openPreviewBtn.addEventListener('click', handleOpenPreview);
  }
}