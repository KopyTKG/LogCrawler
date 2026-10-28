// Package redact replaces what identifies people and systems in a log with stable placeholders:
// the same input always maps to the same placeholder inside one Redactor, so a redacted log still
// shows which call, user or issue is which.
//
// It is a second line of defence after the mobile app's own masking, and it errs on the side of
// removing too much.
package redact

import (
	"fmt"
	"regexp"
	"strings"
)

var (
	bearerRe = regexp.MustCompile(`(?i)\bBearer\s+[A-Za-z0-9._~+/=-]+`)
	jwtRe    = regexp.MustCompile(`\beyJ[A-Za-z0-9_-]+\.[A-Za-z0-9_-]+\.[A-Za-z0-9_-]*`)
	emailRe  = regexp.MustCompile(`[A-Za-z0-9._%+'-]+@[A-Za-z0-9-]+(?:\.[A-Za-z0-9-]+)+`)
	uuidRe   = regexp.MustCompile(`\b[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}\b`)
	urlRe    = regexp.MustCompile(`(https?://)([A-Za-z0-9.-]+)(:\d+)?([^\s"'<>]*)`)
	queryRe  = regexp.MustCompile(`([?&][A-Za-z0-9_.-]+=)([^&\s"'#]*)`)
	issueRe  = regexp.MustCompile(`\b[A-Z]{2,8}-U?\d{4,}\b`)
	bigNumRe = regexp.MustCompile(`\b\d{5,}\b`)
	// JSON string values that carry personal or free text: any key ending in name, email, login or
	// phone, plus the free-text keys below. The same pattern with escaped quotes catches JSON that
	// was logged inside a string. Names of error classes (AxiosError) are kept.
	jsonTextRe = regexp.MustCompile(
		`("(?:\w*(?:[Nn]ame|[Ee]mail|[Ll]ogin|[Pp]hone)|title|content|description|reporter|assignee|author|comment|text|subject|value|label)"\s*:\s*)"((?:[^"\\]|\\.)*)"`)
	jsonTextEscRe = regexp.MustCompile(
		`(\\"(?:\w*(?:[Nn]ame|[Ee]mail|[Ll]ogin|[Pp]hone)|title|content|description|reporter|assignee|author|comment|text|subject|value|label)\\"\s*:\s*)\\"((?:[^"\\]|\\[^"])*?)\\"`)
	keepKeyRe    = regexp.MustCompile(`^"(?:xsrf\w*|typeName|bookmarkName|transitionName)"`)
	errorClassRe = regexp.MustCompile(`^[A-Za-z]*Error$`)
	startedRe    = regexp.MustCompile(`(--- log started \S+ — ).*?( \d[\d.]* )\(([^)]*)\) ---`)
	timeLineRe   = regexp.MustCompile(`^\d{2}:\d{2}:\d{2}\.\d{3} `)
)

// Redactor holds the placeholder mappings. Use one per output so placeholders stay consistent.
type Redactor struct {
	maps map[string]map[string]string
}

func New() *Redactor { return &Redactor{maps: map[string]map[string]string{}} }

func (r *Redactor) name(kind, in string, format func(n int) string) string {
	m := r.maps[kind]
	if m == nil {
		m = map[string]string{}
		r.maps[kind] = m
	}
	if out, ok := m[in]; ok {
		return out
	}
	out := format(len(m) + 1)
	m[in] = out
	return out
}

// Text redacts free text: a report, a pasted node, a single line.
func (r *Redactor) Text(s string) string {
	s = bearerRe.ReplaceAllString(s, "Bearer [redacted]")
	s = jwtRe.ReplaceAllString(s, "[redacted-jwt]")
	s = emailRe.ReplaceAllStringFunc(s, func(m string) string {
		return r.name("email", strings.ToLower(m), func(n int) string { return fmt.Sprintf("user%d@example.test", n) })
	})
	s = uuidRe.ReplaceAllStringFunc(s, func(m string) string {
		return r.name("uuid", strings.ToLower(m), func(n int) string { return fmt.Sprintf("aaaaaaaa-aaaa-aaaa-aaaa-a%011d", n) })
	})
	s = urlRe.ReplaceAllStringFunc(s, r.url)
	s = jsonTextRe.ReplaceAllStringFunc(s, func(m string) string {
		sub := jsonTextRe.FindStringSubmatch(m)
		if keepKeyRe.MatchString(sub[1]) || errorClassRe.MatchString(sub[2]) {
			return m
		}
		return sub[1] + `"redacted"`
	})
	s = jsonTextEscRe.ReplaceAllStringFunc(s, func(m string) string {
		sub := jsonTextEscRe.FindStringSubmatch(m)
		if errorClassRe.MatchString(sub[2]) {
			return m
		}
		return sub[1] + `\"redacted\"`
	})
	s = issueRe.ReplaceAllStringFunc(s, func(m string) string {
		return r.name("issue", m, func(n int) string { return fmt.Sprintf("ISS-%04d", 1000+n) })
	})
	s = bigNumRe.ReplaceAllStringFunc(s, func(m string) string {
		return r.name("num", m, func(n int) string { return fmt.Sprintf("%d", 10000+n) })
	})
	return s
}

func (r *Redactor) url(m string) string {
	p := urlRe.FindStringSubmatch(m)
	host := r.name("host", strings.ToLower(p[2]), func(n int) string { return fmt.Sprintf("host%d.example.test", n) })
	rest := queryRe.ReplaceAllString(p[4], "${1}x")
	return p[1] + host + p[3] + rest
}

// File redacts a whole log. Its header names the customer and the variant, which no pattern can
// find, so the header block (everything before the first marker or event) is rewritten by
// position: the title, and the variant in the first field. The variant also appears in the
// "log started" lines.
func (r *Redactor) File(content string) string {
	lines := strings.Split(content, "\n")
	names := customerNames(lines)
	inHeader := true
	field := 0
	for i, l := range lines {
		if strings.HasPrefix(l, "=== ") || timeLineRe.MatchString(l) {
			inHeader = false
		}
		if inHeader {
			lines[i] = r.headerLine(l, &field)
			continue
		}
		l = startedRe.ReplaceAllString(l, "${1}DRIVE! Example${2}(exampletest) ---")
		for _, n := range names {
			l = n.ReplaceAllString(l, "example")
		}
		lines[i] = r.Text(l)
	}
	return strings.Join(lines, "\n")
}

// customerNames finds what the variant is called (first header field) and the brand inside it
// ("grosvenorprod" → "grosvenor"), so they can be removed wherever they appear: config dumps,
// store links, category codes.
func customerNames(lines []string) []*regexp.Regexp {
	for _, l := range lines {
		if strings.HasPrefix(l, "=== ") || timeLineRe.MatchString(l) {
			break
		}
		if _, v, ok := strings.Cut(l, ": "); ok {
			v = strings.TrimSpace(v)
			names := []string{v}
			if b := strings.TrimRight(strings.TrimSuffix(strings.TrimSuffix(strings.TrimSuffix(strings.TrimSuffix(
				strings.ToLower(v), "prod"), "test"), "dev"), "uat"), "-_ "); len(b) >= 4 && b != strings.ToLower(v) {
				names = append(names, b)
			}
			var res []*regexp.Regexp
			for _, n := range names {
				if len(n) >= 4 {
					res = append(res, regexp.MustCompile(`(?i)`+regexp.QuoteMeta(n)))
				}
			}
			return res
		}
	}
	return nil
}

func (r *Redactor) headerLine(l string, field *int) string {
	if strings.TrimSpace(l) == "" {
		return l
	}
	key, val, ok := strings.Cut(l, ": ")
	if !ok {
		return "DRIVE! Example"
	}
	*field++
	if *field == 1 { // the variant
		val = "exampletest"
	}
	return key + ": " + r.Text(val)
}
