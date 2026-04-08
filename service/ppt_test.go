package service

import (
	"os"
	"testing"
)

func TestMakePPTXFromJSON(t *testing.T) {
	if _, err := os.Stat("../var/temp/temp.json"); err != nil {
		t.Fatalf("temp.json 확인 실패: %v", err)
	}

	svc, err := NewPPTService()
	if err != nil {
		t.Fatalf("NewPPTService 실패: %v", err)
	}

	result, err := svc.MakePPTXFromJSON()
	if err != nil {
		t.Fatalf("MakePPTXFromJSON 실패: %v", err)
	}

	if result == nil || result.PptxFile == "" {
		t.Fatalf("PPTX 결과 경로가 비어 있습니다")
	}

	if _, err := os.Stat(result.PptxFile); err != nil {
		t.Fatalf("생성된 PPTX 파일 확인 실패: %v", err)
	}

	t.Logf("PPTX 생성 성공: %s", result.PptxFile)
}
