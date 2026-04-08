package service

import (
	"archive/zip"
	"bytes"
	"fmt"
	"os"
)

type WordService struct {
	office *OfficeService
}

func NewWordService() (*WordService, error) {
	office, err := NewOfficeService()
	if err != nil {
		return nil, err
	}

	return &WordService{
		office: office,
	}, nil
}

func (s *WordService) MakeDOCXFromJSON() (*OfficeResult, error) {
	if s.office == nil {
		return nil, fmt.Errorf("office service가 초기화되지 않았습니다")
	}

	doc, err := s.office.LoadQTDocumentFromJSON()
	if err != nil {
		return nil, err
	}

	outPath := s.office.GetTempDocxPath()

	if err := os.Remove(outPath); err != nil && !os.IsNotExist(err) {
		return nil, fmt.Errorf("기존 docx 파일 삭제 실패: %w", err)
	}

	// 1. office_export.exe 우선
	if err := s.office.ExportDOCXWithExe(doc, outPath); err == nil {
		return &OfficeResult{
			Success:  true,
			Message:  "temp.docx 생성이 완료되었습니다.",
			DocxFile: outPath,
			PptxFile: "",
		}, nil
	}

	// 2. 실패 시 Go fallback
	if err := s.BuildDOCX(doc, outPath); err != nil {
		return nil, err
	}

	return &OfficeResult{
		Success:  true,
		Message:  "temp.docx 생성이 완료되었습니다.",
		DocxFile: outPath,
		PptxFile: "",
	}, nil
}

func (s *WordService) BuildDOCX(doc *QTDocument, outPath string) error {
	if doc == nil {
		return fmt.Errorf("QTDocument가 nil 입니다")
	}
	if outPath == "" {
		return fmt.Errorf("출력 경로가 비어 있습니다")
	}

	documentXML := s.buildDocumentXML(doc)

	files := map[string]string{
		`[Content_Types].xml`: `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<Types xmlns="http://schemas.openxmlformats.org/package/2006/content-types">
  <Default Extension="rels" ContentType="application/vnd.openxmlformats-package.relationships+xml"/>
  <Default Extension="xml" ContentType="application/xml"/>
  <Override PartName="/word/document.xml" ContentType="application/vnd.openxmlformats-officedocument.wordprocessingml.document.main+xml"/>
  <Override PartName="/docProps/core.xml" ContentType="application/vnd.openxmlformats-package.core-properties+xml"/>
  <Override PartName="/docProps/app.xml" ContentType="application/vnd.openxmlformats-officedocument.extended-properties+xml"/>
</Types>`,
		`_rels/.rels`: `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships">
  <Relationship Id="rId1" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/officeDocument" Target="word/document.xml"/>
  <Relationship Id="rId2" Type="http://schemas.openxmlformats.org/package/2006/relationships/metadata/core-properties" Target="docProps/core.xml"/>
  <Relationship Id="rId3" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/extended-properties" Target="docProps/app.xml"/>
</Relationships>`,
		`docProps/core.xml`: buildCoreXML("S2QT DOCX"),
		`docProps/app.xml`: `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<Properties xmlns="http://schemas.openxmlformats.org/officeDocument/2006/extended-properties"
 xmlns:vt="http://schemas.openxmlformats.org/officeDocument/2006/docPropsVTypes">
  <Application>S2QT</Application>
</Properties>`,
		`word/document.xml`: documentXML,
	}

	return writeWordZipPackage(outPath, files)
}

func (s *WordService) buildDocumentXML(doc *QTDocument) string {
	var b bytes.Buffer

	b.WriteString(`<?xml version="1.0" encoding="UTF-8" standalone="yes"?>`)
	b.WriteString(`<w:document xmlns:wpc="http://schemas.microsoft.com/office/word/2010/wordprocessingCanvas" `)
	b.WriteString(`xmlns:mc="http://schemas.openxmlformats.org/markup-compatibility/2006" `)
	b.WriteString(`xmlns:o="urn:schemas-microsoft-com:office:office" `)
	b.WriteString(`xmlns:r="http://schemas.openxmlformats.org/officeDocument/2006/relationships" `)
	b.WriteString(`xmlns:m="http://schemas.openxmlformats.org/officeDocument/2006/math" `)
	b.WriteString(`xmlns:v="urn:schemas-microsoft-com:vml" `)
	b.WriteString(`xmlns:wp14="http://schemas.microsoft.com/office/word/2010/wordprocessingDrawing" `)
	b.WriteString(`xmlns:wp="http://schemas.openxmlformats.org/drawingml/2006/wordprocessingDrawing" `)
	b.WriteString(`xmlns:w10="urn:schemas-microsoft-com:office:word" `)
	b.WriteString(`xmlns:w="http://schemas.openxmlformats.org/wordprocessingml/2006/main" `)
	b.WriteString(`xmlns:w14="http://schemas.microsoft.com/office/word/2010/wordml" `)
	b.WriteString(`xmlns:w15="http://schemas.microsoft.com/office/word/2012/wordml" `)
	b.WriteString(`xmlns:wpg="http://schemas.microsoft.com/office/word/2010/wordprocessingGroup" `)
	b.WriteString(`xmlns:wpi="http://schemas.microsoft.com/office/word/2010/wordprocessingInk" `)
	b.WriteString(`xmlns:wne="http://schemas.microsoft.com/office/word/2006/wordml" `)
	b.WriteString(`xmlns:wps="http://schemas.microsoft.com/office/word/2010/wordprocessingShape" `)
	b.WriteString(`mc:Ignorable="w14 w15 wp14">`)
	b.WriteString(`<w:body>`)

	if doc.Title != "" {
		b.WriteString(wordParagraphTitle(doc.Title))
		b.WriteString(wordParagraphBlank())
	}

	if doc.BibleText != "" {
		b.WriteString(wordParagraphInfo("본문: " + doc.BibleText))
	}
	if doc.Hymn != "" {
		b.WriteString(wordParagraphInfo("찬송: " + doc.Hymn))
	}
	if doc.Preacher != "" {
		b.WriteString(wordParagraphInfo("설교자: " + doc.Preacher))
	}
	if doc.ChurchName != "" {
		b.WriteString(wordParagraphInfo("교회명: " + doc.ChurchName))
	}
	if doc.SermonDate != "" {
		b.WriteString(wordParagraphInfo("설교일: " + doc.SermonDate))
	}
	if doc.BibleText != "" || doc.Hymn != "" || doc.Preacher != "" || doc.ChurchName != "" || doc.SermonDate != "" {
		b.WriteString(wordParagraphBlank())
	}

	if len(doc.Summary) > 0 {
		b.WriteString(wordParagraphSection("말씀의 창: 본문 요약"))
		for _, p := range doc.Summary {
			b.WriteString(wordParagraphBody(p))
		}
		b.WriteString(wordParagraphBlank())
	}

	if len(doc.Messages) > 0 {
		b.WriteString(wordParagraphSection("오늘의 메시지"))
		for _, msg := range doc.Messages {
			if msg.Title != "" {
				b.WriteString(wordParagraphSubTitle(msg.Title))
			}
			for _, p := range msg.Paragraphs {
				b.WriteString(wordParagraphBody(p))
			}
		}
		b.WriteString(wordParagraphBlank())
	}

	if len(doc.ReflectionItems) > 0 {
		b.WriteString(wordParagraphSection("깊은 묵상과 적용"))
		for _, item := range doc.ReflectionItems {
			b.WriteString(wordParagraphBullet(item))
		}
		b.WriteString(wordParagraphBlank())
	}

	if doc.PrayerTitle != "" || len(doc.PrayerParagraphs) > 0 {
		title := doc.PrayerTitle
		if title == "" {
			title = "오늘의 기도"
		}
		b.WriteString(wordParagraphSection(title))
		for _, p := range doc.PrayerParagraphs {
			b.WriteString(wordParagraphBody(p))
		}
		b.WriteString(wordParagraphBlank())
	}

	b.WriteString(`<w:sectPr>`)
	b.WriteString(`<w:pgSz w:w="11906" w:h="16838"/>`)
	b.WriteString(`<w:pgMar w:top="720" w:right="720" w:bottom="720" w:left="720" w:header="708" w:footer="708" w:gutter="0"/>`)
	b.WriteString(`</w:sectPr>`)

	b.WriteString(`</w:body></w:document>`)
	return b.String()
}

func wordParagraphTitle(text string) string {
	return `<w:p>
  <w:pPr>
    <w:jc w:val="center"/>
    <w:spacing w:after="200"/>
  </w:pPr>
  <w:r>
    <w:rPr>
      <w:b/>
      <w:sz w:val="36"/>
    </w:rPr>
    <w:t xml:space="preserve">` + xmlEscape(text) + `</w:t>
  </w:r>
</w:p>`
}

func wordParagraphSection(text string) string {
	return `<w:p>
  <w:pPr>
    <w:spacing w:before="180" w:after="100"/>
  </w:pPr>
  <w:r>
    <w:rPr>
      <w:b/>
      <w:sz w:val="28"/>
    </w:rPr>
    <w:t xml:space="preserve">` + xmlEscape(text) + `</w:t>
  </w:r>
</w:p>`
}

func wordParagraphSubTitle(text string) string {
	return `<w:p>
  <w:pPr>
    <w:spacing w:before="120" w:after="60"/>
  </w:pPr>
  <w:r>
    <w:rPr>
      <w:b/>
      <w:sz w:val="24"/>
    </w:rPr>
    <w:t xml:space="preserve">` + xmlEscape(text) + `</w:t>
  </w:r>
</w:p>`
}

func wordParagraphBody(text string) string {
	return `<w:p>
  <w:pPr>
    <w:spacing w:after="100"/>
  </w:pPr>
  <w:r>
    <w:rPr>
      <w:sz w:val="22"/>
    </w:rPr>
    <w:t xml:space="preserve">` + xmlEscape(text) + `</w:t>
  </w:r>
</w:p>`
}

func wordParagraphInfo(text string) string {
	return `<w:p>
  <w:pPr>
    <w:spacing w:after="60"/>
  </w:pPr>
  <w:r>
    <w:rPr>
      <w:b/>
      <w:sz w:val="20"/>
    </w:rPr>
    <w:t xml:space="preserve">` + xmlEscape(text) + `</w:t>
  </w:r>
</w:p>`
}

func wordParagraphBullet(text string) string {
	return `<w:p>
  <w:pPr>
    <w:spacing w:after="60"/>
    <w:ind w:left="360" w:hanging="180"/>
  </w:pPr>
  <w:r>
    <w:rPr>
      <w:sz w:val="22"/>
    </w:rPr>
    <w:t xml:space="preserve">• ` + xmlEscape(text) + `</w:t>
  </w:r>
</w:p>`
}

func wordParagraphBlank() string {
	return `<w:p/>`
}

func writeWordZipPackage(outPath string, files map[string]string) error {
	f, err := os.Create(outPath)
	if err != nil {
		return fmt.Errorf("파일 생성 실패: %w", err)
	}
	defer f.Close()

	zw := zip.NewWriter(f)

	for name, content := range files {
		w, err := zw.Create(name)
		if err != nil {
			_ = zw.Close()
			return fmt.Errorf("zip 항목 생성 실패(%s): %w", name, err)
		}

		if _, err := w.Write([]byte(content)); err != nil {
			_ = zw.Close()
			return fmt.Errorf("zip 항목 쓰기 실패(%s): %w", name, err)
		}
	}

	if err := zw.Close(); err != nil {
		return fmt.Errorf("zip 종료 실패: %w", err)
	}

	return nil
}
