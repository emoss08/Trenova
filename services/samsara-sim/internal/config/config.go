package config

import (
	"fmt"
	"os"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

type ServerConfig struct {
	Host string `yaml:"host"`
	Port int    `yaml:"port"`
}

type AuthConfig struct {
	Tokens []string `yaml:"tokens"`
}

type SeedConfig struct {
	FixturePath       string `yaml:"fixturePath"`
	DeterministicSeed string `yaml:"deterministicSeed"`
	RouteDatasetPath  string `yaml:"routeDatasetPath"`
}

type ScenarioConfig struct {
	DefaultProfile string `yaml:"defaultProfile"`
}

type SimulationConfig struct {
	FleetSize      int     `yaml:"fleetSize"`
	TripHoursMin   int     `yaml:"tripHoursMin"`
	TripHoursMax   int     `yaml:"tripHoursMax"`
	EventIntensity string  `yaml:"eventIntensity"`
	ViolationRate  float64 `yaml:"violationRate"`
	SpeedingRate   float64 `yaml:"speedingRate"`
	ScriptPath     string  `yaml:"scriptPath"`
	ScriptMode     string  `yaml:"scriptMode"`
	ScriptTimezone string  `yaml:"scriptTimezone"`
}

type WebhooksConfig struct {
	Enabled        bool          `yaml:"enabled"`
	SigningSecret  string        `yaml:"signingSecret"`
	MaxAttempts    int           `yaml:"maxAttempts"`
	InitialBackoff time.Duration `yaml:"initialBackoff"`
}

type Config struct {
	Server     ServerConfig     `yaml:"server"`
	Auth       AuthConfig       `yaml:"auth"`
	Seed       SeedConfig       `yaml:"seed"`
	Scenario   ScenarioConfig   `yaml:"scenario"`
	Simulation SimulationConfig `yaml:"simulation"`
	Webhooks   WebhooksConfig   `yaml:"webhooks"`
}

func Default() Config {
	return Config{
		Server: ServerConfig{
			Host: "0.0.0.0",
			Port: 8091,
		},
		Auth: AuthConfig{
			Tokens: []string{"dev-samsara-token", "dev-samsara-token-readonly"},
		},
		Seed: SeedConfig{
			FixturePath:       "./config/fixtures/default.json",
			DeterministicSeed: "samsara-sim-v1",
			RouteDatasetPath:  "./config/datasets/texas_osm_routes.geojson",
		},
		Scenario: ScenarioConfig{
			DefaultProfile: "default",
		},
		Simulation: SimulationConfig{
			FleetSize:      12,
			TripHoursMin:   8,
			TripHoursMax:   12,
			EventIntensity: "balanced",
			ViolationRate:  0.08,
			SpeedingRate:   0.14,
			ScriptPath:     "./config/scenarios/default.yaml",
			ScriptMode:     "merge",
			ScriptTimezone: "UTC",
		},
		Webhooks: WebhooksConfig{
			Enabled:        true,
			SigningSecret:  "sim-signing-secret",
			MaxAttempts:    3,
			InitialBackoff: 200 * time.Millisecond,
		},
	}
}

func Load(path string) (Config, error) {
	cfg := Default()
	if strings.TrimSpace(path) == "" {
		return cfg, nil
	}

	content, err := os.ReadFile(path)
	if err != nil {
		return Config{}, fmt.Errorf("read config file: %w", err)
	}
	if err = yaml.Unmarshal(content, &cfg); err != nil {
		return Config{}, fmt.Errorf("parse config file: %w", err)
	}
	normalizeConfig(&cfg)

	return cfg, nil
}

func normalizeConfig(cfg *Config) {
	if cfg == nil {
		return
	}

	normalizeServerConfig(&cfg.Server)
	normalizeAuthConfig(&cfg.Auth)
	normalizeSeedConfig(&cfg.Seed)
	normalizeScenarioConfig(&cfg.Scenario)
	normalizeSimulationConfig(&cfg.Simulation)
	normalizeWebhooksConfig(&cfg.Webhooks)
}

func normalizeServerConfig(server *ServerConfig) {
	if server == nil {
		return
	}
	if server.Host == "" {
		server.Host = "0.0.0.0"
	}
	if server.Port <= 0 {
		server.Port = 8091
	}
}

func normalizeAuthConfig(auth *AuthConfig) {
	if auth == nil {
		return
	}
	if len(auth.Tokens) == 0 {
		auth.Tokens = []string{"dev-samsara-token", "dev-samsara-token-readonly"}
	}
}

func normalizeSeedConfig(seed *SeedConfig) {
	if seed == nil {
		return
	}
	if seed.FixturePath == "" {
		seed.FixturePath = "./config/fixtures/default.json"
	}
	if seed.DeterministicSeed == "" {
		seed.DeterministicSeed = "samsara-sim-v1"
	}
	if seed.RouteDatasetPath == "" {
		seed.RouteDatasetPath = "./config/datasets/texas_osm_routes.geojson"
	}
}

func normalizeScenarioConfig(scenario *ScenarioConfig) {
	if scenario == nil {
		return
	}
	if scenario.DefaultProfile == "" {
		scenario.DefaultProfile = "default"
	}
}

func normalizeSimulationConfig(simulation *SimulationConfig) {
	if simulation == nil {
		return
	}

	if simulation.FleetSize <= 0 {
		simulation.FleetSize = 12
	}
	if simulation.TripHoursMin <= 0 {
		simulation.TripHoursMin = 8
	}
	if simulation.TripHoursMax <= 0 {
		simulation.TripHoursMax = 12
	}
	if simulation.TripHoursMax < simulation.TripHoursMin {
		simulation.TripHoursMax = simulation.TripHoursMin
	}
	if strings.TrimSpace(simulation.EventIntensity) == "" {
		simulation.EventIntensity = "balanced"
	}
	simulation.ViolationRate = clampFraction(simulation.ViolationRate, 0.08)
	simulation.SpeedingRate = clampFraction(simulation.SpeedingRate, 0.14)
	if strings.TrimSpace(simulation.ScriptPath) == "" {
		simulation.ScriptPath = "./config/scenarios/default.yaml"
	}
	scriptMode := strings.ToLower(strings.TrimSpace(simulation.ScriptMode))
	switch scriptMode {
	case "merge", "override":
		simulation.ScriptMode = scriptMode
	default:
		simulation.ScriptMode = "merge"
	}
	if strings.TrimSpace(simulation.ScriptTimezone) == "" {
		simulation.ScriptTimezone = "UTC"
	}
}

func normalizeWebhooksConfig(webhooks *WebhooksConfig) {
	if webhooks == nil {
		return
	}
	if webhooks.MaxAttempts <= 0 {
		webhooks.MaxAttempts = 3
	}
	if webhooks.InitialBackoff <= 0 {
		webhooks.InitialBackoff = 200 * time.Millisecond
	}
}

func clampFraction(value, fallback float64) float64 {
	if value < 0 || value > 1 {
		return fallback
	}
	return value
}
