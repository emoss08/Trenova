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
  mfa:
    recoveryCodePepper: "9f2c4e7a1b3d5f60718293a4b5c6d7e8f9a0b1c2"
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
imaging:
  endpoint: "dns:///imaging:50051"
  serverName: "imaging"
  tls:
    enabled: true
    caFile: "/certs/imaging/ca.pem"
    certFile: "/certs/imaging/client.pem"
    keyFile: "/certs/imaging/client-key.pem"
`
}
