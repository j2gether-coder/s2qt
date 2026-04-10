package service

import (
	"os"
	"testing"
)

func TestMakeDOCXFromJSON(t *testing.T) {
	if _, err := os.Stat("../var/temp/temp.json"); err != nil {
		t.Fatalf("temp.json 확인 실패: %v", err)
	}

	svc, err := NewWordService()
	if err != nil {
		t.Fatalf("NewWordService 실패: %v", err)
	}

	result, err := svc.MakeDOCXFromJSON()
	if err != nil {
		t.Fatalf("MakeDOCXFromJSON 실패: %v", err)
	}

	if result == nil || result.DocxFile == "" {
		t.Fatalf("DOCX 결과 경로가 비어 있습니다")
	}

	if _, err := os.Stat(result.DocxFile); err != nil {
		t.Fatalf("생성된 DOCX 파일 확인 실패: %v", err)
	}

	t.Logf("DOCX 생성 성공: %s", result.DocxFile)
}
