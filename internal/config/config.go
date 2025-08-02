package config

import (
	"os"

	"gopkg.in/yaml.v2"
)

type Config struct {
	Server struct {
		Port int `yaml:"port"`
	} `yaml:"server"`
	Files struct {
		AllowedExtensions   []string `yaml:"allowed_extensions"`
		AllowedContentTypes []string `yaml:"allowed_content_types"`
	} `yaml:"files"`

	ArchiveDir string `yaml:"archive_dir"`
}

func LoadConfig(path string) (*Config, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	var cfg Config
	dec := yaml.NewDecoder(f)
	if err := dec.Decode(&cfg); err != nil {
		return nil, err
	}

	if cfg.ArchiveDir == "" {
		cfg.ArchiveDir = "archives"
	}

	return &cfg, nil
}
