package config

import (
	"os"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Addr string `yaml:"addr"`
	Strategy string `yaml:"strategy"`
	Backends []string `yaml:"backends"`
	Health HealthConfig `yaml:"health"`
}

type HealthConfig struct {
	IntervalSecs int64 `yaml:"interval_secs"`
	TimeoutSecs int64 `yaml:"timeout_secs"`
	FallThreshold int64 `yaml:"fall_threshold"`
	RiseThreshold int64 `yaml:"rise_threshold"`
}

func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var config Config
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, err
	}
	return &config, nil
}
