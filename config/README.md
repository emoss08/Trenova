# `/config` Directory Documentation

## Overview

The `config` directory contains all configuration files for different environments and configuration examples. This directory manages application settings, credentials, and environment-specific configurations.

## Directory Structure

```markdown
/config/
├── development/              # Development environment configs
│   ├── .env.development
│   └── config.development.yaml
├── staging/                 # Staging environment configs
│   ├── .env.staging
│   └── config.staging.yaml
├── production/             # Production environment configs
│   ├── .env.production
│   └── config.production.yaml
├── .env.example           # Example .env file with documentation
└── config.example.yaml    # Example YAML with documentation
```

## Configuration Types

### Environment Variables (.env)

```bash
# .env.example
# Server Configuration
SERVER_PORT=8080
SERVER_HOST=0.0.0.0
SERVER_TIMEOUT=30s

# Database Configuration
DB_HOST=localhost
DB_PORT=5432
DB_NAME=trenova
DB_USER=postgres
DB_PASSWORD=your_password
DB_SSL_MODE=disable

# Redis Configuration
REDIS_HOST=localhost
REDIS_PORT=6379
REDIS_PASSWORD=

# JWT Configuration
JWT_SECRET=your_jwt_secret
JWT_EXPIRY=24h

# External Services
MAPS_API_KEY=your_google_maps_key
WEATHER_API_KEY=your_weather_api_key
```

### YAML Configuration

```yaml
# config.example.yaml
server:
  port: 8080
  host: "0.0.0.0"
  timeouts:
    read: 30s
    write: 30s
    idle: 60s
  cors:
    allowed_origins:
      - "https://app.trenova.io"
    allowed_methods:
      - "GET"
      - "POST"
      - "PUT"
      - "DELETE"

database:
  max_open_conns: 25
  max_idle_conns: 25
  conn_max_lifetime: 5m

monitoring:
  tracing:
    enabled: true
    sampling_rate: 0.1
  metrics:
    enabled: true
    path: "/metrics"

features:
  route_optimization:
    enabled: true
    batch_size: 100
    worker_count: 5
```

## Best Practices

1. **Security**
   - Never commit sensitive values
   - Use environment variables for secrets
   - Encrypt sensitive configs in production
   - Rotate secrets regularly

2. **Documentation**
   - Document all configuration options
   - Include validation rules
   - Provide example values
   - Explain impacts of changes

3. **Structure**
   - Group related settings
   - Use consistent naming
   - Keep hierarchy logical
   - Limit nesting depth

4. **Validation**
   - Validate all configurations
   - Provide clear error messages
   - Set sensible defaults
   - Check required values

## What Does NOT Belong Here

1. **Application Code**
   - No logic implementation
   - No code execution
   - No dynamic configuration

2. **Sensitive Data**
   - No real passwords
   - No API keys
   - No certificates
   - No private keys

3. **Generated Files**
   - No build artifacts
   - No temporary files
   - No logs
   - No cache files

## Example Implementation

### Configuration Loading

```go
// internal/pkg/config/config.go
type Config struct {
    Server   ServerConfig   `yaml:"server"`
    Database DatabaseConfig `yaml:"database"`
    Redis    RedisConfig    `yaml:"redis"`
    Features FeaturesConfig `yaml:"features"`
}

type ServerConfig struct {
    Port     int           `yaml:"port" validate:"required,min=1,max=65535"`
    Host     string        `yaml:"host" validate:"required"`
    Timeouts TimeoutConfig `yaml:"timeouts"`
}

func Load() (*Config, error) {
    // Determine environment
    env := os.Getenv("APP_ENV")
    if env == "" {
        env = "development"
    }

    // Load .env file
    if err := godotenv.Load(fmt.Sprintf("config/%s/.env.%s", env, env)); err != nil {
        return nil, fmt.Errorf("error loading .env file: %w", err)
    }

    // Load YAML config
    configPath := fmt.Sprintf("config/%s/config.%s.yaml", env, env)
    configFile, err := os.ReadFile(configPath)
    if err != nil {
        return nil, fmt.Errorf("error reading config file: %w", err)
    }

    var config Config
    if err := yaml.Unmarshal(configFile, &config); err != nil {
        return nil, fmt.Errorf("error parsing config file: %w", err)
    }

    // Validate configuration
    if err := validateConfig(&config); err != nil {
        return nil, fmt.Errorf("invalid configuration: %w", err)
    }

    return &config, nil
}
```

### Configuration Validation

```go
// internal/pkg/config/validator.go
func validateConfig(config *Config) error {
    validate := validator.New()
    
    // Register custom validators
    validate.RegisterValidation("timestring", validateTimeString)
    
    if err := validate.Struct(config); err != nil {
        return processValidationErrors(err)
    }
    
    return nil
}

func processValidationErrors(err error) error {
    var validationErrors validator.ValidationErrors
    if errors.As(err, &validationErrors) {
        // Create user-friendly error messages
        messages := make([]string, 0, len(validationErrors))
        for _, err := range validationErrors {
            messages = append(messages, formatValidationError(err))
        }
        return fmt.Errorf("configuration validation failed: %s", strings.Join(messages, "; "))
    }
    return err
}
```

## Environment-Specific Guidelines

1. **Development**
   - Use local services
   - Enable debug logging
   - Use shorter timeouts
   - Enable development features

2. **Staging**
   - Mirror production config
   - Use separate services
   - Enable additional logging
   - Test production features

3. **Production**
   - Use secure values
   - Optimize performance
   - Enable monitoring
   - Disable debug features

## Version Control Guidelines

1. **Gitignore Rules**

```gitignore
# Ignore all config files
/config/**/*.env
/config/**/*.yaml

# Except example files
!/config/**/*.example*
```

2. **Secrets Management**

- Use secret management service in production
- Use vault for sensitive values
- Implement secret rotation
- Log access to secrets
