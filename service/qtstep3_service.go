package service

import (
	"fmt"
	"os"

	"s2qt/util"
)

type QTStep3Service struct {
	Paths *util.AppPaths
}

func NewQTStep3Service() (*QTStep3Service, error) {
	paths, err := util.GetAppPaths()
	if err != nil {
		return nil, err
	}
	return &QTStep3Service{Paths: paths}, nil
}

func (s *QTStep3Service) Run(req *QTStep3Request) (*QTStep3Result, error) {
	if req == nil {
		return nil, fmt.Errorf("step3 request가 비어 있습니다")
	}

	if _, err := os.Stat(s.Paths.TempHtml); err != nil {
		return nil, fmt.Errorf("temp.html이 없습니다. Step2 미리보기를 먼저 실행해 주세요")
	}

	if req.DPI <= 0 {
		req.DPI = 300
	}

	result := &QTStep3Result{}

	if req.MakeHTML {
		result.HTML = QTStep3FileResult{
			Success:  true,
			Status:   "완료",
			FilePath: s.Paths.TempHtml,
		}
	}

	// 핵심: footer 자산 준비는 Step3 전체에서 1회만 수행
	var footerCfg *QTFooterConfig
	var footerErr error

	if req.MakePDF || req.MakePNG {
		footerCfg, footerErr = s.buildFooterConfig()
		if footerErr != nil {
			return nil, footerErr
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
	db, err := OpenSQLite(s.Paths.DBFile)
	if err != nil {
		return nil, fmt.Errorf("step3 db open 실패: %w", err)
	}
	//defer db.Close()

	footerSvc, err := NewFooterService(db)
	if err != nil {
		return nil, fmt.Errorf("footer service 생성 실패: %w", err)
	}

	cfg, err := footerSvc.PrepareFooterConfigFromDB(QTFooterModeSubscriber)
	if err != nil {
		return nil, fmt.Errorf("footer config 준비 실패: %w", err)
	}

	return cfg, nil
}