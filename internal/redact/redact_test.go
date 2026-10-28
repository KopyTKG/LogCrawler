package redact

import (
	"strings"
	"testing"
)

func TestText(t *testing.T) {
	cases := []struct{ name, in, want string }{
		{"bearer", `Authorization: Bearer abc.DEF-123_x`, `Authorization: Bearer [redacted]`},
		{"email stable", `a@corp.test b@corp.test A@corp.test`, `user1@example.test user2@example.test user1@example.test`},
		{"uuid", `"requestId":"f8303386-8295-4a71-b02d-1bf51fbca1dd"`, `"requestId":"aaaaaaaa-aaaa-aaaa-aaaa-a00000000001"`},
		{"host and query", `GET https://api.corp.test/sd/v1/x?token=SECRET&page=2`, `GET https://host1.example.test/sd/v1/x?token=x&page=x`},
		{"issue number", `number":"SSL-2601779"`, `number":"ISS-1001"`},
		{"big id", `parameters for 263067:`, `parameters for 10001:`},
		{"text fields", `{"title":"Broken printer","name":"Jane Doe"}`, `{"title":"redacted","name":"redacted"}`},
		{"error name kept", `{"name":"AxiosError","message":"x"}`, `{"name":"AxiosError","message":"x"}`},
		{"timing untouched", `19:17:09.647 LOG [http] #7 ✕ 500 GET /x 8526ms`, `19:17:09.647 LOG [http] #7 ✕ 500 GET /x 8526ms`},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := New().Text(c.in); got != c.want {
				t.Errorf("got  %q\nwant %q", got, c.want)
			}
		})
	}
}

func TestFileHeaderAndStarted(t *testing.T) {
	in := "Some Customer Connect\nVariant: customerprod\nBuild: 34\nAPI: https://api.customer.test/sd\n\n" +
		"=== current session ===\n" +
		"10:00:00.000 LOG   --- log started 2026-01-01T10:00:00.000Z — Some Customer Connect 2.6.2 (customerprod) ---\n"
	out := New().File(in)
	for _, leak := range []string{"Customer", "customer"} {
		if strings.Contains(out, leak) {
			t.Errorf("%q leaked:\n%s", leak, out)
		}
	}
	if !strings.Contains(out, "Variant: exampletest") || !strings.Contains(out, "Build: 34") {
		t.Errorf("header wrong:\n%s", out)
	}
}

func TestFileRemovesVariantAndBrandEverywhere(t *testing.T) {
	in := "Acme Connect\nVariant: acmeprod\n\n=== current session ===\n" +
		`10:00:00.000 LOG   config: {"env":"acmeprod","code":"ACME.HSEQ","store":"/app/acme-connect/id1"}` + "\n"
	out := New().File(in)
	if strings.Contains(strings.ToLower(out), "acme") {
		t.Errorf("leaked:\n%s", out)
	}
}

func TestFreeTextKeys(t *testing.T) {
	cases := []struct{ in, want string }{
		{`{"reporterName":"Jane Doe","value":"Jane Doe"}`, `{"reporterName":"redacted","value":"redacted"}`},
		{`{"xsrfCookieName":"XSRF-TOKEN"}`, `{"xsrfCookieName":"XSRF-TOKEN"}`},
		{`"{\"reporter\":{\"name\":\"Jane Doe\",\"id\":1}}"`, `"{\"reporter\":{\"name\":\"redacted\",\"id\":1}}"`},
	}
	for _, c := range cases {
		if got := New().Text(c.in); got != c.want {
			t.Errorf("got  %s\nwant %s", got, c.want)
		}
	}
}
