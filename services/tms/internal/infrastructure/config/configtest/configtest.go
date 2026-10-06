package configtest

func ValidYAML() string {
	return `
app:
  name: trenova
  version: "1.0.0"
server:
  host: "0.0.0.0"
  port: 8080
database:
  host: localhost
  port: 5432
  name: testdb
  user: postgres
  password: testpass
  sslMode: require
security:
  session:
    secret: "a-very-long-secret-that-is-at-least-32-characters-long"
    name: "__Host-trenova_session"
    maxAge: "24h"
    httpOnly: true
    secure: true
    sameSite: "strict"
    path: "/"
    domain: ""
  csrf:
    tokenName: "csrf_token"
    headerName: "X-CSRF-Token"
  rateLimit:
    requestsPerMinute: 60
    burstSize: 10
  encryption:
    mode: envelope
    keyManager: gcp-autokey
    key: "a-very-long-encryption-key-that-is-at-least-32-chars"
    gcpKms:
      cryptoKey: "projects/test/locations/us/keyRings/autokey/cryptoKeys/trenova"
logging:
  level: info
  format: json
  output: stdout
monitoring:
  metrics:
    enabled: false
    port: 9090
    path: "/metrics"
  tracing:
    enabled: false
    provider: "otlp"
    endpoint: "localhost:4317"
    serviceName: "trenova-tms"
cache:
  host: localhost
  port: 6379
temporal:
  hostPort: "localhost:7233"
  security:
    enableEncryption: false
    encryptionKeyID: "test-key-id"
audit:
  batchSize: 500
  maxEntriesPerFlush: 5000
  dlqMaxRetries: 5
storage:
  endpoint: "http://localhost:9000"
  accessKey: "test"
  secretKey: "test"
  bucket: "test"
system:
  systemUserPassword: "test-system-password"
`
}
