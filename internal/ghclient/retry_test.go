package ghclient

import (
	"context"
	"errors"
	"net/http"
	"regexp"
	"testing"
)

func Test_checkRetryNoRatelimit(t *testing.T) {
	t.Parallel()

	for _, tt := range []struct {
		name    string
		ctx     context.Context
		resp    *http.Response
		err     error
		want    bool
		wantErr *string
	}{
		{
			name: "does_not_retry_ok",
			ctx:  t.Context(),
			resp: &http.Response{StatusCode: http.StatusOK},
			err:  nil,
			want: false,
		},
		{
			name: "does_not_retry_4xx",
			ctx:  t.Context(),
			resp: &http.Response{StatusCode: http.StatusBadRequest},
			err:  nil,
			want: false,
		},
		{
			name: "does_not_retry_ratelimit",
			ctx:  t.Context(),
			resp: &http.Response{StatusCode: http.StatusTooManyRequests},
			err:  nil,
			want: false,
		},
		{
			name: "retry_5xx",
			ctx:  t.Context(),
			resp: &http.Response{StatusCode: http.StatusInternalServerError},
			err:  nil,
			want: true,
		},
		{
			name: "retry_transport_error",
			ctx:  t.Context(),
			resp: nil,
			err:  errors.New("transport error"),
			want: true,
		},
		{
			name: "handles_error",
			ctx: func() context.Context {
				ctx, cancel := context.WithCancel(t.Context())
				cancel()
				return ctx
			}(),
			resp:    &http.Response{StatusCode: http.StatusOK},
			err:     nil,
			want:    false,
			wantErr: new("context canceled"),
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := checkRetryNoRatelimit(tt.ctx, tt.resp, tt.err)
			if err != nil {
				if tt.wantErr == nil || !regexp.MustCompile(regexp.QuoteMeta(*tt.wantErr)).MatchString(err.Error()) {
					t.Fatalf("want error %s, got %v", *tt.wantErr, err)
				}
				return
			}

			if tt.wantErr != nil {
				t.Fatalf("want error %s, got %v", *tt.wantErr, err)
			}

			if got != tt.want {
				t.Fatalf("want %v, got %v", tt.want, got)
			}
		})
	}
}
