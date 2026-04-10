package service

import (
	"archive/zip"
	"fmt"
	"os"
	"strings"
)

type PPTService struct {
	office *OfficeService
}

type PPTSlide struct {
	SlideType string
	Title     string
	SubTitle  string
	BodyLines []string
}

func NewPPTService() (*PPTService, error) {
	office, err := NewOfficeService()
	if err != nil {
		return nil, err
	}

	return &PPTService{
		office: office,
	}, nil
}

func (s *PPTService) MakePPTXFromJSON() (*OfficeResult, error) {
	if s.office == nil {
		return nil, fmt.Errorf("office service가 초기화되지 않았습니다")
	}

	doc, err := s.office.LoadQTDocumentFromJSON()
	if err != nil {
		return nil, err
	}

	outPath := s.office.GetTempPptxPath()

	if err := os.Remove(outPath); err != nil && !os.IsNotExist(err) {
		return nil, fmt.Errorf("기존 pptx 파일 삭제 실패: %w", err)
	}

	// 1. office_export.exe 우선
	if err := s.office.ExportPPTXWithExe(doc, outPath); err == nil {
		return &OfficeResult{
			Success:  true,
			Message:  "temp.pptx 생성이 완료되었습니다.",
			DocxFile: "",
			PptxFile: outPath,
		}, nil
	}

	// 2. 실패 시 Go fallback
	slides := s.BuildSlides(doc)
	if err := s.BuildPPTX(slides, outPath); err != nil {
		return nil, err
	}

	return &OfficeResult{
		Success:  true,
		Message:  "temp.pptx 생성이 완료되었습니다.",
		DocxFile: "",
		PptxFile: outPath,
	}, nil
}

func (s *PPTService) BuildSlides(doc *QTDocument) []PPTSlide {
	if doc == nil {
		return nil
	}

	var slides []PPTSlide

	subTitleLines := []string{}
	if doc.BibleText != "" {
		subTitleLines = append(subTitleLines, "본문: "+doc.BibleText)
	}
	if doc.Hymn != "" {
		subTitleLines = append(subTitleLines, "찬송: "+doc.Hymn)
	}
	if doc.Preacher != "" {
		subTitleLines = append(subTitleLines, "설교자: "+doc.Preacher)
	}
	if doc.ChurchName != "" {
		subTitleLines = append(subTitleLines, "교회명: "+doc.ChurchName)
	}
	if doc.SermonDate != "" {
		subTitleLines = append(subTitleLines, "설교일: "+doc.SermonDate)
	}

	slides = append(slides, PPTSlide{
		SlideType: "title",
		Title:     doc.Title,
		SubTitle:  joinLines(subTitleLines),
		BodyLines: nil,
	})

	if len(doc.Summary) > 0 {
		slides = append(slides, PPTSlide{
			SlideType: "content",
			Title:     "말씀의 창: 본문 요약",
			SubTitle:  "",
			BodyLines: doc.Summary,
		})
	}

	if len(doc.Messages) > 0 {
		var messageLines []string
		for _, msg := range doc.Messages {
			line := stringsTrimSpace(msg.Title)
			if len(msg.Paragraphs) > 0 {
				if line != "" {
					line += " - "
				}
				line += msg.Paragraphs[0]
			}
			if stringsTrimSpace(line) != "" {
				messageLines = append(messageLines, line)
			}
		}

		slides = append(slides, PPTSlide{
			SlideType: "content",
			Title:     "오늘의 메시지",
			SubTitle:  "",
			BodyLines: messageLines,
		})
	}

	if len(doc.ReflectionItems) > 0 {
		slides = append(slides, PPTSlide{
			SlideType: "bullet",
			Title:     "깊은 묵상과 적용",
			SubTitle:  "",
			BodyLines: doc.ReflectionItems,
		})
	}

	if len(doc.PrayerParagraphs) > 0 {
		title := doc.PrayerTitle
		if stringsTrimSpace(title) == "" {
			title = "오늘의 기도"
		}

		slides = append(slides, PPTSlide{
			SlideType: "content",
			Title:     title,
			SubTitle:  "",
			BodyLines: doc.PrayerParagraphs,
		})
	}

	return slides
}

func (s *PPTService) BuildPPTX(slides []PPTSlide, outPath string) error {
	if outPath == "" {
		return fmt.Errorf("출력 경로가 비어 있습니다")
	}
	if len(slides) == 0 {
		return fmt.Errorf("생성할 슬라이드가 없습니다")
	}

	files := map[string]string{
		`[Content_Types].xml`: buildPPTContentTypes(len(slides)),
		`_rels/.rels`: `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships">
  <Relationship Id="rId1" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/officeDocument" Target="ppt/presentation.xml"/>
  <Relationship Id="rId2" Type="http://schemas.openxmlformats.org/package/2006/relationships/metadata/core-properties" Target="docProps/core.xml"/>
  <Relationship Id="rId3" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/extended-properties" Target="docProps/app.xml"/>
</Relationships>`,
		`docProps/core.xml`:                 buildCoreXML("S2QT PPTX"),
		`docProps/app.xml`:                  buildPPTAppXML(len(slides)),
		`ppt/presentation.xml`:              buildPresentationXML(len(slides)),
		`ppt/_rels/presentation.xml.rels`:   buildPresentationRelsXML(len(slides)),
		`ppt/slideMasters/slideMaster1.xml`: buildSlideMasterXML(),
		`ppt/slideMasters/_rels/slideMaster1.xml.rels`: `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships">
  <Relationship Id="rId1" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/slideLayout" Target="../slideLayouts/slideLayout1.xml"/>
  <Relationship Id="rId2" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/theme" Target="../theme/theme1.xml"/>
</Relationships>`,
		`ppt/slideLayouts/slideLayout1.xml`: buildSlideLayoutXML(),
		`ppt/slideLayouts/_rels/slideLayout1.xml.rels`: `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships">
  <Relationship Id="rId1" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/slideMaster" Target="../slideMasters/slideMaster1.xml"/>
</Relationships>`,
		`ppt/theme/theme1.xml`: buildThemeXML(),
		`ppt/presProps.xml`: `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<p:presentationPr xmlns:p="http://schemas.openxmlformats.org/presentationml/2006/main"/>`,
		`ppt/viewProps.xml`: `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<p:viewPr xmlns:p="http://schemas.openxmlformats.org/presentationml/2006/main"/>`,
		`ppt/tableStyles.xml`: `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<a:tblStyleLst xmlns:a="http://schemas.openxmlformats.org/drawingml/2006/main" def=""/>`,
	}

	for i, slide := range slides {
		n := i + 1
		files[fmt.Sprintf("ppt/slides/slide%d.xml", n)] = buildSlideXML(slide)
		files[fmt.Sprintf("ppt/slides/_rels/slide%d.xml.rels", n)] = `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships">
  <Relationship Id="rId1" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/slideLayout" Target="../slideLayouts/slideLayout1.xml"/>
</Relationships>`
	}

	return writePPTZipPackage(outPath, files)
}

func buildSlideXML(slide PPTSlide) string {
	title := stringsTrimSpace(slide.Title)
	if title == "" {
		title = "QT"
	}

	subTitle := stringsTrimSpace(slide.SubTitle)
	bodyLines := compactTextLines(slide.BodyLines)
	bodyXML := buildSlideBodyXML(slide, bodyLines)

	return `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<p:sld xmlns:a="http://schemas.openxmlformats.org/drawingml/2006/main"
 xmlns:r="http://schemas.openxmlformats.org/officeDocument/2006/relationships"
 xmlns:p="http://schemas.openxmlformats.org/presentationml/2006/main">
  <p:cSld>
    <p:spTree>
      <p:nvGrpSpPr>
        <p:cNvPr id="1" name=""/>
        <p:cNvGrpSpPr/>
        <p:nvPr/>
      </p:nvGrpSpPr>
      <p:grpSpPr>
        <a:xfrm>
          <a:off x="0" y="0"/>
          <a:ext cx="0" cy="0"/>
          <a:chOff x="0" y="0"/>
          <a:chExt cx="0" cy="0"/>
        </a:xfrm>
      </p:grpSpPr>

      <p:sp>
        <p:nvSpPr>
          <p:cNvPr id="2" name="Title 1"/>
          <p:cNvSpPr/>
          <p:nvPr/>
        </p:nvSpPr>
        <p:spPr>
          <a:xfrm>
            <a:off x="457200" y="228600"/>
            <a:ext cx="8229600" cy="914400"/>
          </a:xfrm>
        </p:spPr>
        <p:txBody>
          <a:bodyPr/>
          <a:lstStyle/>
          <a:p>
            <a:r>
              <a:rPr lang="ko-KR" sz="2800" b="1"/>
              <a:t>` + xmlEscape(title) + `</a:t>
            </a:r>
          </a:p>
        </p:txBody>
      </p:sp>

      ` + buildSubTitleShapeXML(subTitle) + `

      <p:sp>
        <p:nvSpPr>
          <p:cNvPr id="4" name="Content Placeholder 2"/>
          <p:cNvSpPr/>
          <p:nvPr/>
        </p:nvSpPr>
        <p:spPr>
          <a:xfrm>
            <a:off x="457200" y="1800000"/>
            <a:ext cx="8229600" cy="3880000"/>
          </a:xfrm>
        </p:spPr>
        <p:txBody>
          <a:bodyPr wrap="square"/>
          <a:lstStyle/>
          ` + bodyXML + `
        </p:txBody>
      </p:sp>
    </p:spTree>
  </p:cSld>
  <p:clrMapOvr>
    <a:masterClrMapping/>
  </p:clrMapOvr>
</p:sld>`
}

func buildSubTitleShapeXML(subTitle string) string {
	if subTitle == "" {
		return ""
	}

	return `<p:sp>
        <p:nvSpPr>
          <p:cNvPr id="3" name="SubTitle 1"/>
          <p:cNvSpPr/>
          <p:nvPr/>
        </p:nvSpPr>
        <p:spPr>
          <a:xfrm>
            <a:off x="457200" y="1180000"/>
            <a:ext cx="8229600" cy="457200"/>
          </a:xfrm>
        </p:spPr>
        <p:txBody>
          <a:bodyPr/>
          <a:lstStyle/>
          <a:p>
            <a:r>
              <a:rPr lang="ko-KR" sz="1800"/>
              <a:t>` + xmlEscape(subTitle) + `</a:t>
            </a:r>
          </a:p>
        </p:txBody>
      </p:sp>`
}

func buildSlideBodyXML(slide PPTSlide, bodyLines []string) string {
	if len(bodyLines) == 0 {
		return `<a:p><a:r><a:rPr lang="ko-KR" sz="1800"/><a:t>내용 없음</a:t></a:r></a:p>`
	}

	var b strings.Builder

	for i, line := range bodyLines {
		line = stringsTrimSpace(line)
		if line == "" {
			continue
		}

		switch slide.SlideType {
		case "bullet":
			b.WriteString(`<a:p><a:pPr lvl="0" marL="0" indent="0"><a:buChar char="•"/></a:pPr><a:r><a:rPr lang="ko-KR" sz="1800"/><a:t>`)
			b.WriteString(xmlEscape(line))
			b.WriteString(`</a:t></a:r></a:p>`)
		case "title":
			if i == 0 {
				b.WriteString(`<a:p><a:r><a:rPr lang="ko-KR" sz="1800"/><a:t>`)
				b.WriteString(xmlEscape(line))
				b.WriteString(`</a:t></a:r></a:p>`)
			}
		default:
			b.WriteString(`<a:p><a:r><a:rPr lang="ko-KR" sz="1800"/><a:t>`)
			b.WriteString(xmlEscape(line))
			b.WriteString(`</a:t></a:r></a:p>`)
		}
	}

	if b.Len() == 0 {
		return `<a:p><a:r><a:rPr lang="ko-KR" sz="1800"/><a:t>내용 없음</a:t></a:r></a:p>`
	}

	return b.String()
}

func writePPTZipPackage(outPath string, files map[string]string) error {
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

func buildPPTContentTypes(slideCount int) string {
	var b strings.Builder
	b.WriteString(`<?xml version="1.0" encoding="UTF-8" standalone="yes"?>`)
	b.WriteString(`<Types xmlns="http://schemas.openxmlformats.org/package/2006/content-types">`)
	b.WriteString(`<Default Extension="rels" ContentType="application/vnd.openxmlformats-package.relationships+xml"/>`)
	b.WriteString(`<Default Extension="xml" ContentType="application/xml"/>`)
	b.WriteString(`<Override PartName="/ppt/presentation.xml" ContentType="application/vnd.openxmlformats-officedocument.presentationml.presentation.main+xml"/>`)
	b.WriteString(`<Override PartName="/ppt/slideMasters/slideMaster1.xml" ContentType="application/vnd.openxmlformats-officedocument.presentationml.slideMaster+xml"/>`)
	b.WriteString(`<Override PartName="/ppt/slideLayouts/slideLayout1.xml" ContentType="application/vnd.openxmlformats-officedocument.presentationml.slideLayout+xml"/>`)
	b.WriteString(`<Override PartName="/ppt/theme/theme1.xml" ContentType="application/vnd.openxmlformats-officedocument.theme+xml"/>`)
	b.WriteString(`<Override PartName="/docProps/core.xml" ContentType="application/vnd.openxmlformats-package.core-properties+xml"/>`)
	b.WriteString(`<Override PartName="/docProps/app.xml" ContentType="application/vnd.openxmlformats-officedocument.extended-properties+xml"/>`)
	for i := 1; i <= slideCount; i++ {
		b.WriteString(fmt.Sprintf(`<Override PartName="/ppt/slides/slide%d.xml" ContentType="application/vnd.openxmlformats-officedocument.presentationml.slide+xml"/>`, i))
	}
	b.WriteString(`</Types>`)
	return b.String()
}

func buildPPTAppXML(slideCount int) string {
	return `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<Properties xmlns="http://schemas.openxmlformats.org/officeDocument/2006/extended-properties"
 xmlns:vt="http://schemas.openxmlformats.org/officeDocument/2006/docPropsVTypes">
  <Application>S2QT</Application>
  <Slides>` + fmt.Sprintf("%d", slideCount) + `</Slides>
  <Notes>0</Notes>
  <HiddenSlides>0</HiddenSlides>
  <MMClips>0</MMClips>
  <PresentationFormat>On-screen Show (4:3)</PresentationFormat>
</Properties>`
}

func buildPresentationXML(slideCount int) string {
	var b strings.Builder
	b.WriteString(`<?xml version="1.0" encoding="UTF-8" standalone="yes"?>`)
	b.WriteString(`<p:presentation xmlns:a="http://schemas.openxmlformats.org/drawingml/2006/main" xmlns:r="http://schemas.openxmlformats.org/officeDocument/2006/relationships" xmlns:p="http://schemas.openxmlformats.org/presentationml/2006/main">`)
	b.WriteString(`<p:sldMasterIdLst><p:sldMasterId id="2147483648" r:id="rId1"/></p:sldMasterIdLst>`)
	b.WriteString(`<p:sldIdLst>`)
	for i := 1; i <= slideCount; i++ {
		b.WriteString(fmt.Sprintf(`<p:sldId id="%d" r:id="rId%d"/>`, 255+i, i+1))
	}
	b.WriteString(`</p:sldIdLst>`)
	b.WriteString(`<p:sldSz cx="9144000" cy="6858000"/>`)
	b.WriteString(`<p:notesSz cx="6858000" cy="9144000"/>`)
	b.WriteString(`</p:presentation>`)
	return b.String()
}

func buildPresentationRelsXML(slideCount int) string {
	var b strings.Builder
	b.WriteString(`<?xml version="1.0" encoding="UTF-8" standalone="yes"?>`)
	b.WriteString(`<Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships">`)
	b.WriteString(`<Relationship Id="rId1" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/slideMaster" Target="slideMasters/slideMaster1.xml"/>`)
	for i := 1; i <= slideCount; i++ {
		b.WriteString(fmt.Sprintf(`<Relationship Id="rId%d" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/slide" Target="slides/slide%d.xml"/>`, i+1, i))
	}
	b.WriteString(`</Relationships>`)
	return b.String()
}

func buildSlideMasterXML() string {
	return `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<p:sldMaster xmlns:a="http://schemas.openxmlformats.org/drawingml/2006/main"
 xmlns:r="http://schemas.openxmlformats.org/officeDocument/2006/relationships"
 xmlns:p="http://schemas.openxmlformats.org/presentationml/2006/main">
  <p:cSld name="Office Master">
    <p:bg>
      <p:bgRef idx="1001">
        <a:schemeClr val="bg1"/>
      </p:bgRef>
    </p:bg>
    <p:spTree>
      <p:nvGrpSpPr>
        <p:cNvPr id="1" name=""/>
        <p:cNvGrpSpPr/>
        <p:nvPr/>
      </p:nvGrpSpPr>
      <p:grpSpPr>
        <a:xfrm>
          <a:off x="0" y="0"/>
          <a:ext cx="0" cy="0"/>
          <a:chOff x="0" y="0"/>
          <a:chExt cx="0" cy="0"/>
        </a:xfrm>
      </p:grpSpPr>
    </p:spTree>
  </p:cSld>
  <p:clrMap accent1="accent1" accent2="accent2" accent3="accent3" accent4="accent4" accent5="accent5" accent6="accent6"
    bg1="lt1" bg2="lt2" folHlink="folHlink" hlink="hlink" tx1="dk1" tx2="dk2"/>
  <p:sldLayoutIdLst>
    <p:sldLayoutId id="1" r:id="rId1"/>
  </p:sldLayoutIdLst>
</p:sldMaster>`
}

func buildSlideLayoutXML() string {
	return `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<p:sldLayout xmlns:a="http://schemas.openxmlformats.org/drawingml/2006/main"
 xmlns:r="http://schemas.openxmlformats.org/officeDocument/2006/relationships"
 xmlns:p="http://schemas.openxmlformats.org/presentationml/2006/main" type="titleAndText" preserve="1">
  <p:cSld name="Title and Text">
    <p:spTree>
      <p:nvGrpSpPr>
        <p:cNvPr id="1" name=""/>
        <p:cNvGrpSpPr/>
        <p:nvPr/>
      </p:nvGrpSpPr>
      <p:grpSpPr>
        <a:xfrm>
          <a:off x="0" y="0"/>
          <a:ext cx="0" cy="0"/>
          <a:chOff x="0" y="0"/>
          <a:chExt cx="0" cy="0"/>
        </a:xfrm>
      </p:grpSpPr>
    </p:spTree>
  </p:cSld>
  <p:clrMapOvr>
    <a:masterClrMapping/>
  </p:clrMapOvr>
</p:sldLayout>`
}

func buildThemeXML() string {
	return `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<a:theme xmlns:a="http://schemas.openxmlformats.org/drawingml/2006/main" name="Office Theme">
  <a:themeElements>
    <a:clrScheme name="Office">
      <a:dk1><a:sysClr val="windowText" lastClr="000000"/></a:dk1>
      <a:lt1><a:sysClr val="window" lastClr="FFFFFF"/></a:lt1>
      <a:dk2><a:srgbClr val="1F2937"/></a:dk2>
      <a:lt2><a:srgbClr val="F3F4F6"/></a:lt2>
      <a:accent1><a:srgbClr val="1F8F55"/></a:accent1>
      <a:accent2><a:srgbClr val="1D4ED8"/></a:accent2>
      <a:accent3><a:srgbClr val="F4C542"/></a:accent3>
      <a:accent4><a:srgbClr val="B39DDB"/></a:accent4>
      <a:accent5><a:srgbClr val="4B5563"/></a:accent5>
      <a:accent6><a:srgbClr val="D1D5DB"/></a:accent6>
      <a:hlink><a:srgbClr val="0563C1"/></a:hlink>
      <a:folHlink><a:srgbClr val="954F72"/></a:folHlink>
    </a:clrScheme>
    <a:fontScheme name="Office">
      <a:majorFont>
        <a:latin typeface="Arial"/>
      </a:majorFont>
      <a:minorFont>
        <a:latin typeface="Arial"/>
      </a:minorFont>
    </a:fontScheme>
    <a:fmtScheme name="Office">
      <a:fillStyleLst>
        <a:solidFill><a:schemeClr val="phClr"/></a:solidFill>
      </a:fillStyleLst>
      <a:lnStyleLst>
        <a:ln w="9525" cap="flat" cmpd="sng" algn="ctr"><a:solidFill><a:schemeClr val="phClr"/></a:solidFill></a:ln>
      </a:lnStyleLst>
      <a:effectStyleLst>
        <a:effectStyle><a:effectLst/></a:effectStyle>
      </a:effectStyleLst>
      <a:bgFillStyleLst>
        <a:solidFill><a:schemeClr val="phClr"/></a:solidFill>
      </a:bgFillStyleLst>
    </a:fmtScheme>
  </a:themeElements>
  <a:objectDefaults/>
  <a:extraClrSchemeLst/>
</a:theme>`
}

func joinLines(lines []string) string {
	return strings.Join(compactTextLines(lines), "\n")
}

func compactTextLines(lines []string) []string {
	var out []string
	for _, line := range lines {
		line = stringsTrimSpace(line)
		if line == "" {
			continue
		}
		out = append(out, line)
	}
	return out
}

func stringsTrimSpace(s string) string {
	return strings.TrimSpace(s)
}
