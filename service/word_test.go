package service

import (
	"os"
	"testing"
)

// TestMakeDOCXFromJSON은 실제 var/temp/의 temp.json을 읽고 같은 폴더에
// temp.docx와 temp_office_docx.json을 만든다.
//
// DOCX는 Step3에 아직 연결되어 있지 않아(qtstep3_service.go) 이 산출물은
// 앱 동작에 쓰이지 않는다. 그대로 두면 go test 한 번에 var/temp가 오염되고
// 사용자는 산출물에서 제외한 파일이 생긴 것으로 본다.
//
// 기능을 다시 연결할 때 S2QT_OFFICE_TEST=1 로 돌린다.
func TestMakeDOCXFromJSON(t *testing.T) {
	if os.Getenv("S2QT_OFFICE_TEST") == "" {
		t.Skip("S2QT_OFFICE_TEST=1 일 때만 실행합니다 (실제 var/temp에 파일을 만듭니다)")
	}

	if _, err := os.Stat("../var/temp/temp.json"); err != nil {
		t.Skipf("temp.json이 없어 건너뜁니다: %v", err)
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
