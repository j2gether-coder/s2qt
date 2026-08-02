package service

import (
	"encoding/json"
	"testing"
)

func TestRepairJSONCommasFixesMissingCommas(t *testing.T) {
	// message 블록 사이 콤마 누락(사용자 사례 재현)
	broken := `{
  "metadata": { "title": "t", "support_scriptures_full": [
    { "reference": "a", "text": "aa" }
    { "reference": "b", "text": "bb" }
  ] },
  "sections": [
    { "type": "message", "blocks": [
      { "type": "message_title", "text": "one" }
      { "type": "paragraph", "text": "p1" }
      { "type": "message_title", "text": "two" }
    ] }
  ]
}`

	if json.Valid([]byte(broken)) {
		t.Fatal("test fixture should be invalid JSON")
	}

	out := CleanLLMJSONOutput(broken)
	if !json.Valid([]byte(out)) {
		t.Fatalf("repaired output still invalid: %s", out)
	}
}

func TestCleanLLMJSONOutputLeavesValidJSONStructurallyIntact(t *testing.T) {
	valid := `{"a":{"x":"1"},"b":["p","q"],"c":"end"}`

	out := CleanLLMJSONOutput(valid)
	if !json.Valid([]byte(out)) {
		t.Fatalf("valid json became invalid: %s", out)
	}

	// 문자열 내부의 }{ 같은 문자는 건드리면 안 된다.
	tricky := `{"text":"a}{b","k":"v"}`
	if got := CleanLLMJSONOutput(tricky); got != tricky {
		t.Fatalf("string content mutated: got %s", got)
	}
}

func TestRepairDoesNotInsertBetweenKeyAndValue(t *testing.T) {
	// 키 뒤 값이 오브젝트/배열/문자열이어도 콤마를 넣으면 안 된다.
	cases := []string{
		`{"k":{"n":"1"}}`,
		`{"k":["1","2"]}`,
		`{"k":"v"}`,
	}
	for _, c := range cases {
		if got := repairJSONCommas(c); got != c {
			t.Fatalf("mutated valid json: in=%s out=%s", c, got)
		}
	}
}
