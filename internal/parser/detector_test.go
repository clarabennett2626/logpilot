package parser

import (
	"testing"
)

// --- Real-world JSON log samples ---
var jsonSamples = []string{
	`{"timestamp":"2024-01-15T10:30:00Z","level":"info","message":"Server started","port":8080}`,
	`{"time":"2024-01-15T10:30:01Z","severity":"error","msg":"Connection refused","host":"db-01"}`,
	`{"ts":"2024-01-15T10:30:02.123Z","lvl":"debug","message":"Query executed","duration_ms":42}`,
	`{"@timestamp":"2024-01-15T10:30:03Z","level":"warn","message":"High memory usage","percent":92}`,
	`{"created_at":"2024-01-15 10:30:04","log_level":"info","text":"User logged in","user_id":"abc123"}`,
	`{"timestamp":1705312200,"level":"error","message":"Disk full","device":"/dev/sda1"}`,
	`{"time":"2024-01-15T10:30:06Z","level":"info","msg":"Request completed","method":"GET","path":"/api/v1/users","status":200}`,
	`{"timestamp":"2024-01-15T10:30:07Z","level":"debug","message":"Cache hit","key":"user:123","ttl":300}`,
	`{"ts":"2024-01-15T10:30:08.456Z","severity":"info","log":"Healthcheck passed","service":"api"}`,
	`{"time":"2024-01-15T10:30:09Z","level":"fatal","message":"Unable to bind port","port":443,"error":"permission denied"}`,
	`{"timestamp":"2024-01-15T10:30:10Z","level":"info","message":"Deployment started","version":"v2.3.1"}`,
	`{"@timestamp":"2024-01-15T10:30:11Z","level":"warn","msg":"Slow query","query":"SELECT *","duration":"3.2s"}`,
	`{"ts":1705312212000,"level":"error","message":"TLS handshake failed","remote":"10.0.0.5"}`,
	`{"time":"2024-01-15T10:30:13Z","level":"info","message":"Kafka consumer started","topic":"events","partition":0}`,
	`{"timestamp":"2024-01-15T10:30:14Z","level":"debug","message":"Middleware executed","middleware":"auth","latency_us":150}`,
	`{"time":"2024-01-15T10:30:15Z","severity":"critical","message":"Data corruption detected","table":"users"}`,
	`{"timestamp":"2024-01-15T10:30:16Z","level":"info","message":"Graceful shutdown initiated"}`,
	`{"ts":"2024-01-15T10:30:17Z","level":"warn","msg":"Rate limited","client_ip":"192.168.1.100","limit":"100/min"}`,
}

// --- Real-world logfmt samples ---
var logfmtSamples = []string{
	`ts=2024-01-15T10:30:00Z level=info msg="Server started" port=8080`,
	`time=2024-01-15T10:30:01Z level=error message="Connection refused" host=db-01`,
	`ts=2024-01-15T10:30:02Z lvl=debug msg="Query executed" duration_ms=42`,
	`timestamp=2024-01-15T10:30:03Z severity=warn msg="High memory" percent=92`,
	`ts=2024-01-15T10:30:04Z level=info msg="Request handled" method=GET path=/api/health status=200`,
	`time=2024-01-15T10:30:05Z level=error msg="Timeout" service=payment duration=30s`,
	`ts=2024-01-15T10:30:06Z level=info msg="User created" user_id=abc123 email=user@example.com`,
	`ts=2024-01-15T10:30:07Z level=debug msg="Cache miss" key=session:456`,
	`time=2024-01-15T10:30:08Z level=fatal msg="Out of memory" rss=4096mb`,
	`ts=2024-01-15T10:30:09Z level=warn msg="Deprecated endpoint" path=/v1/old caller=10.0.0.3`,
	`ts=2024-01-15T10:30:10Z level=info msg="Config reloaded" file=/etc/app.conf`,
	`time=2024-01-15T10:30:11Z level=error msg="DNS lookup failed" host=api.example.com`,
	`ts=2024-01-15T10:30:12Z level=info msg="Worker started" worker_id=7 queue=default`,
	`ts=2024-01-15T10:30:13Z level=debug msg="Span exported" trace_id=abc123 span_id=def456`,
	`time=2024-01-15T10:30:14Z level=warn msg="Certificate expiring" domain=example.com days=14`,
	`ts=2024-01-15T10:30:15Z level=info msg="Batch processed" count=500 duration=1.2s`,
	`ts=2024-01-15T10:30:16Z level=error msg="Write failed" table=events error="disk full"`,
	`time=2024-01-15T10:30:17Z level=info msg="Metrics flushed" metrics=42 endpoint=prometheus`,
}

// --- Real-world plain text samples ---
var plainSamples = []string{
	`2024-01-15T10:30:00Z INFO  Server started on port 8080`,
	`2024-01-15 10:30:01 ERROR Connection refused to database`,
	`Jan 15 10:30:02 myhost sshd[1234]: Accepted publickey for user from 10.0.0.1`,
	`[15/Jan/2024:10:30:03 +0000] "GET /index.html HTTP/1.1" 200 1234`,
	`2024/01/15 10:30:04 [error] 5678#0: *9 open() "/usr/share/nginx/html/missing" failed`,
	`Jan  5 10:30:05 server kernel: [12345.678] Out of memory: Kill process 999`,
	`2024-01-15T10:30:06.789Z DEBUG Executing query against primary replica`,
	`2024-01-15 10:30:07 WARN  Disk usage above 90% on /dev/sda1`,
	`FATAL: role "postgres" does not exist`,
	`panic: runtime error: index out of range [5] with length 3`,
	`2024-01-15 10:30:10 INFO  Starting graceful shutdown`,
	`Jan 15 10:30:11 lb haproxy[5678]: 10.0.0.1:443 [15/Jan/2024:10:30:11.000] frontend~ backend/server1 0/0/0/1/1 200 500`,
	`2024-01-15T10:30:12Z WARNING Memory allocation failed, retrying`,
	`[2024-01-15 10:30:13] [CRITICAL] Database replication lag exceeded 60s`,
	`2024/01/15 10:30:14 http: TLS handshake error from 10.0.0.5:54321`,
	`2024-01-15 10:30:15.000 TRACE Entering function ProcessBatch`,
	`--- FAIL: TestUserCreate (0.01s)`,
	`E0115 10:30:17.000000   12345 server.go:123] Unable to attach to pod`,
}

func TestDetectFormat(t *testing.T) {
	tests := []struct {
		name   string
		lines  []string
		expect Format
	}{
		{"json lines", jsonSamples, FormatJSON},
		{"logfmt lines", logfmtSamples, FormatLogfmt},
		{"plain lines", plainSamples, FormatPlain},
		{"empty", nil, FormatUnknown},
		{"single json", jsonSamples[:1], FormatJSON},
		{"single logfmt", logfmtSamples[:1], FormatLogfmt},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := DetectFormat(tt.lines)
			if got != tt.expect {
				t.Errorf("DetectFormat() = %v, want %v", got, tt.expect)
			}
		})
	}
}

func TestDetectLineCounts(t *testing.T) {
	// Verify each sample is detected as its expected format
	for i, line := range jsonSamples {
		if f := detectLine(line); f != FormatJSON {
			t.Errorf("jsonSamples[%d] detected as %v, want JSON: %s", i, f, line)
		}
	}
	for i, line := range logfmtSamples {
		if f := detectLine(line); f != FormatLogfmt {
			t.Errorf("logfmtSamples[%d] detected as %v, want Logfmt: %s", i, f, line)
		}
	}
	for i, line := range plainSamples {
		if f := detectLine(line); f != FormatPlain {
			t.Errorf("plainSamples[%d] detected as %v, want Plain: %s", i, f, line)
		}
	}
}

func TestJSONParser(t *testing.T) {
	p := &JSONParser{}

	entry := p.Parse(`{"timestamp":"2024-01-15T10:30:00Z","level":"info","message":"Server started","port":8080}`)
	if entry.Level != "INFO" {
		t.Errorf("Level = %q, want INFO", entry.Level)
	}
	if entry.Message != "Server started" {
		t.Errorf("Message = %q, want 'Server started'", entry.Message)
	}
	if entry.Timestamp.IsZero() {
		t.Error("Timestamp should not be zero")
	}
	if entry.Fields["port"] != "8080" {
		t.Errorf("Fields[port] = %q, want 8080", entry.Fields["port"])
	}

	// Unix timestamp
	entry2 := p.Parse(`{"timestamp":1705312200,"level":"error","message":"Disk full"}`)
	if entry2.Timestamp.IsZero() {
		t.Error("Unix timestamp should be parsed")
	}

	// Unix millis
	entry3 := p.Parse(`{"ts":1705312212000,"level":"warn","message":"test"}`)
	if entry3.Timestamp.IsZero() {
		t.Error("Unix millis timestamp should be parsed")
	}
}

func TestLogfmtParser(t *testing.T) {
	p := &LogfmtParser{}

	entry := p.Parse(`ts=2024-01-15T10:30:00Z level=info msg="Server started" port=8080`)
	if entry.Level != "INFO" {
		t.Errorf("Level = %q, want INFO", entry.Level)
	}
	if entry.Message != "Server started" {
		t.Errorf("Message = %q, want 'Server started'", entry.Message)
	}
	if entry.Timestamp.IsZero() {
		t.Error("Timestamp should not be zero")
	}
	if entry.Fields["port"] != "8080" {
		t.Errorf("Fields[port] = %q, want 8080", entry.Fields["port"])
	}
}

func TestPlainParser(t *testing.T) {
	p := &PlainParser{}

	entry := p.Parse(`2024-01-15T10:30:00Z INFO  Server started on port 8080`)
	if entry.Level != "INFO" {
		t.Errorf("Level = %q, want INFO", entry.Level)
	}
	if entry.Timestamp.IsZero() {
		t.Error("Timestamp should not be zero")
	}

	// Syslog format
	entry2 := p.Parse(`Jan 15 10:30:02 myhost sshd[1234]: Accepted publickey`)
	if entry2.Timestamp.IsZero() {
		t.Error("Syslog timestamp should be parsed")
	}

	// WARNING normalization
	entry3 := p.Parse(`2024-01-15T10:30:12Z WARNING Memory allocation failed`)
	if entry3.Level != "WARN" {
		t.Errorf("Level = %q, want WARN", entry3.Level)
	}
}

func TestAutoParser(t *testing.T) {
	ap := NewAutoParser()

	mixed := []string{
		`{"level":"info","message":"json line"}`,
		`ts=2024-01-15T10:30:00Z level=info msg="logfmt line"`,
		`2024-01-15 10:30:00 INFO plain text line`,
	}

	formats := []Format{FormatJSON, FormatLogfmt, FormatPlain}
	for i, line := range mixed {
		entry := ap.Parse(line)
		if entry.Format != formats[i] {
			t.Errorf("mixed[%d] format = %v, want %v", i, entry.Format, formats[i])
		}
	}
}

func TestFormatString(t *testing.T) {
	if FormatJSON.String() != "json" {
		t.Error("JSON string")
	}
	if FormatLogfmt.String() != "logfmt" {
		t.Error("Logfmt string")
	}
	if FormatPlain.String() != "plain" {
		t.Error("Plain string")
	}
	if FormatUnknown.String() != "unknown" {
		t.Error("Unknown string")
	}
}

// --- Benchmarks ---

func BenchmarkDetectFormat100JSON(b *testing.B) {
	lines := make([]string, 100)
	for i := range lines {
		lines[i] = jsonSamples[i%len(jsonSamples)]
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		DetectFormat(lines)
	}
}

func BenchmarkDetectFormat100Logfmt(b *testing.B) {
	lines := make([]string, 100)
	for i := range lines {
		lines[i] = logfmtSamples[i%len(logfmtSamples)]
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		DetectFormat(lines)
	}
}

func BenchmarkDetectFormat100Plain(b *testing.B) {
	lines := make([]string, 100)
	for i := range lines {
		lines[i] = plainSamples[i%len(plainSamples)]
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		DetectFormat(lines)
	}
}

func BenchmarkDetectFormat100Mixed(b *testing.B) {
	all := append(append(jsonSamples, logfmtSamples...), plainSamples...)
	lines := make([]string, 100)
	for i := range lines {
		lines[i] = all[i%len(all)]
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		DetectFormat(lines)
	}
}
