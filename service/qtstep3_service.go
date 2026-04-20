package service

import (
	"database/sql"
	"fmt"
	"os"
	"strings"

	"s2qt/util"
)

type QTStep3Service struct {
	Paths *util.AppPaths
	DB    *sql.DB
}

// NewQTStep3Service is kept only for compatibility.
// 운영에서는 공유 DB를 주입하는 NewQTStep3ServiceWithDB를 사용합니다.
func NewQTStep3Service() (*QTStep3Service, error) {
	return nil, fmt.Errorf("shared db가 필요합니다. NewQTStep3ServiceWithDB를 사용해 주세요")
}

func NewQTStep3ServiceWithDB(db *sql.DB) (*QTStep3Service, error) {
	if db == nil {
		return nil, fmt.Errorf("db is nil")
	}

	paths, err := util.GetAppPaths()
	if err != nil {
		return nil, err
	}

	return &QTStep3Service{
		Paths: paths,
		DB:    db,
	}, nil
}

func (s *QTStep3Service) Run(req *QTStep3Request) (*QTStep3Result, error) {
	if req == nil {
		return nil, fmt.Errorf("step3 request가 비어 있습니다")
	}
	if s == nil {
		return nil, fmt.Errorf("step3 service가 nil 입니다")
	}
	if s.Paths == nil {
		return nil, fmt.Errorf("step3 paths가 nil 입니다")
	}

	if _, err := os.Stat(s.Paths.TempHtml); err != nil {
		return nil, fmt.Errorf("temp.html이 없습니다. Step2 미리보기를 먼저 실행해 주세요")
	}

	if req.DPI <= 0 {
		req.DPI = 300
	}

	result := &QTStep3Result{}

	var footerCfg *QTFooterConfig
	if req.MakePDF || req.MakePNG {
		var err error
		footerCfg, err = s.buildFooterConfig()
		if err != nil {
			return nil, err
		}
	}

	if req.MakeHTML {
		result.HTML = QTStep3FileResult{
			Success:  true,
			Status:   "완료",
			FilePath: s.Paths.TempHtml,
		}
	}

	if req.MakePDF {
		if err := s.makePDF(footerCfg); err != nil {
			result.PDF = QTStep3FileResult{
				Success: false,
				Status:  "실패",
				Error:   err.Error(),
			}
		} else {
			result.PDF = QTStep3FileResult{
				Success:  true,
				Status:   "완료",
				FilePath: s.Paths.TempPdf,
			}
		}
	}

	if req.MakePNG {
		if err := s.makePNG(req.DPI, footerCfg); err != nil {
			result.PNG = QTStep3FileResult{
				Success: false,
				Status:  "실패",
				Error:   err.Error(),
			}
		} else {
			result.PNG = QTStep3FileResult{
				Success:  true,
				Status:   "완료",
				FilePath: s.Paths.TempPng,
			}
		}
	}

	if req.MakePDF || req.MakePNG {
		s.applyTemplateAfterGenerate(result, req, footerCfg)
	}

	if req.MakeDOCX {
		result.DOCX = QTStep3FileResult{
			Success: false,
			Status:  "준비중",
			Error:   "DOCX 생성은 아직 연결되지 않았습니다.",
		}
	}

	if req.MakePPTX {
		result.PPTX = QTStep3FileResult{
			Success: false,
			Status:  "준비중",
			Error:   "PPTX 생성은 아직 연결되지 않았습니다.",
		}
	}

	return result, nil
}

func (s *QTStep3Service) makePDF(footerCfg *QTFooterConfig) error {
	pdfSvc, err := NewPDFService()
	if err != nil {
		return err
	}

	b, err := os.ReadFile(s.Paths.TempHtml)
	if err != nil {
		return fmt.Errorf("temp.html 읽기 실패: %w", err)
	}

	_, err = pdfSvc.SaveHtmlAndMakePDFWithFooter(string(b), footerCfg)
	return err
}

func (s *QTStep3Service) makePNG(dpi int, footerCfg *QTFooterConfig) error {
	pngSvc, err := NewPNGService()
	if err != nil {
		return err
	}

	_, err = pngSvc.GenerateFromTempHTMLWithFooter(dpi, footerCfg)
	return err
}

func (s *QTStep3Service) buildFooterConfig() (*QTFooterConfig, error) {
	if s.DB == nil {
		return nil, fmt.Errorf("step3 shared db is nil")
	}

	footerSvc, err := NewFooterService(s.DB)
	if err != nil {
		return nil, fmt.Errorf("footer service 생성 실패: %w", err)
	}

	cfg, err := footerSvc.PrepareFooterConfigFromDB(QTFooterModeSubscriber)
	if err != nil {
		return nil, fmt.Errorf("footer config 준비 실패: %w", err)
	}

	return cfg, nil
}

func (s *QTStep3Service) applyTemplateAfterGenerate(result *QTStep3Result, req *QTStep3Request, footerCfg *QTFooterConfig) {
	if result == nil || req == nil {
		return
	}

	tplSvc, err := NewTemplateServiceWithDB(s.DB)
	if err != nil {
		s.applyTemplateFatalError(result, req, err)
		return
	}

	tplRes, err := tplSvc.ApplySelectedTemplate(&TemplateApplyRequest{
		ApplyPDF:       req.MakePDF && result.PDF.Success,
		ApplyPNG:       req.MakePNG && result.PNG.Success,
		DPI:            req.DPI,
		FooterOverride: footerCfg,
	})

	if tplRes != nil {
		if strings.TrimSpace(tplRes.PDFError) != "" {
			result.PDF.Success = false
			result.PDF.Status = "실패"
			result.PDF.Error = tplRes.PDFError
		}
		if strings.TrimSpace(tplRes.PNGError) != "" {
			result.PNG.Success = false
			result.PNG.Status = "실패"
			result.PNG.Error = tplRes.PNGError
		}
	}

	if err != nil && tplRes == nil {
		s.applyTemplateFatalError(result, req, err)
	}
}

func (s *QTStep3Service) applyTemplateFatalError(result *QTStep3Result, req *QTStep3Request, err error) {
	if result == nil || req == nil || err == nil {
		return
	}

	msg := err.Error()
	if req.MakePDF && result.PDF.Success {
		result.PDF.Success = false
		result.PDF.Status = "실패"
		result.PDF.Error = msg
	}
	if req.MakePNG && result.PNG.Success {
		result.PNG.Success = false
		result.PNG.Status = "실패"
		result.PNG.Error = msg
	}
}
