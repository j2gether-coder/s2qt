package service

import "testing"

func TestNormalizeVideoURL(t *testing.T) {
	cases := []struct {
		name string
		in   string
		want string
	}{
		{
			name: "재생목록 파라미터 제거",
			in:   "https://www.youtube.com/watch?v=6Q2Xs8KcnoM&list=PLn3gC0zxOsmzIvC9JKPGiZKfy_V9j4X_a",
			want: "https://www.youtube.com/watch?v=6Q2Xs8KcnoM",
		},
		{
			name: "재생목록 + 인덱스 제거",
			in:   "https://www.youtube.com/watch?v=6Q2Xs8KcnoM&list=PL123&index=4&t=30s",
			want: "https://www.youtube.com/watch?v=6Q2Xs8KcnoM",
		},
		{
			name: "일반 watch URL 유지",
			in:   "https://www.youtube.com/watch?v=6Q2Xs8KcnoM",
			want: "https://www.youtube.com/watch?v=6Q2Xs8KcnoM",
		},
		{
			name: "앞뒤 공백 제거",
			in:   "  https://www.youtube.com/watch?v=6Q2Xs8KcnoM&list=PL123  ",
			want: "https://www.youtube.com/watch?v=6Q2Xs8KcnoM",
		},
		{
			name: "youtu.be 단축 URL",
			in:   "https://youtu.be/6Q2Xs8KcnoM?list=PL123",
			want: "https://www.youtube.com/watch?v=6Q2Xs8KcnoM",
		},
		{
			name: "모바일 URL",
			in:   "https://m.youtube.com/watch?v=6Q2Xs8KcnoM&list=PL123",
			want: "https://www.youtube.com/watch?v=6Q2Xs8KcnoM",
		},
		{
			name: "shorts는 원본 유지",
			in:   "https://www.youtube.com/shorts/6Q2Xs8KcnoM",
			want: "https://www.youtube.com/shorts/6Q2Xs8KcnoM",
		},
		{
			name: "재생목록 전용 URL은 원본 유지",
			in:   "https://www.youtube.com/playlist?list=PL123",
			want: "https://www.youtube.com/playlist?list=PL123",
		},
		{
			name: "유튜브가 아닌 URL은 원본 유지",
			in:   "https://vimeo.com/123456?list=PL123",
			want: "https://vimeo.com/123456?list=PL123",
		},
		{
			name: "빈 문자열",
			in:   "   ",
			want: "",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := NormalizeVideoURL(tc.in); got != tc.want {
				t.Errorf("NormalizeVideoURL(%q) = %q, want %q", tc.in, got, tc.want)
			}
		})
	}
}
