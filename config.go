package main

import (
	"encoding/json"
	"os"
	"time"

	urapi "github.com/gbl08ma/uptimerobot-api"
)

// Config holds the application-wide settings
type Config struct {
	MonitorComponentMap       map[int][]int
	MonitorMetricMap          map[int][]int
	MonitorComponentStatusMap map[urapi.MonitorStatus]int
	CheckInterval             time.Duration
	CachetAPIkey              string
	CachetEndpoint            string
	UptimeRobotAPIkey         string
	BindAddress               string
}

// NewConfig creates a Config with default settings
func NewConfig() *Config {
	return &Config{
		MonitorComponentMap: make(map[int][]int),
		MonitorMetricMap:    make(map[int][]int),
		MonitorComponentStatusMap: map[urapi.MonitorStatus]int{
			urapi.MonitorStatusDown: 4,
			urapi.MonitorStatusUp:   1,
		},
		CachetAPIkey:      os.Getenv("UPCACHET_CACHET_APIKEY"),
		CachetEndpoint:    os.Getenv("UPCACHET_CACHET_ENDPOINT"),
		UptimeRobotAPIkey: os.Getenv("UPCACHET_UPTIMEROBOT_APIKEY"),
	}
}

// Load loads a JSON-encoded Config from the specified filename
func (c *Config) Load(filename string) error {
	file, _ := os.Open(filename)
	decoder := json.NewDecoder(file)
	return decoder.Decode(c)
}

// Save saves a JSON-encoded Config to the specified filename
func (c *Config) Save(filename string) error {
	file, _ := os.OpenFile(filename, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0664)
	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "    ")
	return encoder.Encode(c)
}
