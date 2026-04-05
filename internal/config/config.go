package config

import (
	"fmt"
	"os"

	"github.com/BurntSushi/toml"
)

type Config struct {
	Server  ServerConfig             `toml:"server"`
	Auth    AuthConfig               `toml:"auth"`
	Station StationConfig            `toml:"station"`
	SDR     SDRConfig                `toml:"sdr"`
	FMRadio FMRadioConfig            `toml:"fm_radio"`
	Ham     HamRadioConfig           `toml:"ham_radio"`
	Airband AirbandConfig            `toml:"airband"`
	Pager   PagerConfig              `toml:"pager"`
	AIS     AISConfig                `toml:"ais"`
	ISS     ISSConfig                `toml:"iss"`
	GSM     GSMConfig                `toml:"gsm"`
	Docker  map[string]DockerService `toml:"docker_services"`
}

type ServerConfig struct {
	Listen  string `toml:"listen"`
	DataDir string `toml:"data_dir"`
}

type AuthConfig struct {
	AdminPasswordHash string `toml:"admin_password_hash"`
	SessionSecret     string `toml:"session_secret"`
}

type StationConfig struct {
	Name      string  `toml:"name"`
	Latitude  float64 `toml:"latitude"`
	Longitude float64 `toml:"longitude"`
	Altitude  float64 `toml:"altitude"`
}

type SDRConfig struct {
	DeviceIndex   int `toml:"device_index"`
	Gain          int `toml:"gain"`
	PPMCorrection int `toml:"ppm_correction"`
}

type Preset struct {
	Name string `toml:"name"`
	Freq string `toml:"freq"`
	Mode string `toml:"mode,omitempty"`
	Band string `toml:"band,omitempty"`
}

type FMRadioConfig struct {
	DefaultFreq  string   `toml:"default_freq"`
	SampleRate   int      `toml:"sample_rate"`
	AudioBitrate string   `toml:"audio_bitrate"`
	Presets      []Preset `toml:"presets"`
}

type HamRadioConfig struct {
	DefaultFreq string   `toml:"default_freq"`
	DefaultMode string   `toml:"default_mode"`
	Squelch     int      `toml:"squelch"`
	Presets     []Preset `toml:"presets"`
}

type AirbandChannel struct {
	Name       string  `toml:"name"`
	Freq       float64 `toml:"freq"`
	Modulation string  `toml:"modulation"`
}

type AirbandConfig struct {
	Gain      int              `toml:"gain"`
	Channels  []AirbandChannel `toml:"channels"`
}

type PagerConfig struct {
	DefaultFreq string   `toml:"default_freq"`
	Protocols   []string `toml:"protocols"`
}

type AISConfig struct {
	Gain       int `toml:"gain"`
	SampleRate int `toml:"sample_rate"`
}

type ISSConfig struct {
	RecordFreq        string `toml:"record_freq"`
	MaxRecordDuration int    `toml:"max_record_duration"`
	TLEUpdateInterval string `toml:"tle_update_interval"`
}

type GSMConfig struct {
	Gain             int           `toml:"gain"`
	ThresholdGSM900  float64       `toml:"threshold_gsm900"`
	ThresholdDCS1800 float64       `toml:"threshold_dcs1800"`
	Operators        []GSMOperator `toml:"operators"`
}

type GSMOperator struct {
	MCCMNC string `toml:"mcc_mnc"`
	Name   string `toml:"name"`
}

type DockerService struct {
	Name        string            `toml:"name"`
	Image       string            `toml:"image"`
	Dockerfile  string            `toml:"dockerfile,omitempty"`
	ComposeFile string            `toml:"compose_file,omitempty"`
	Port        int               `toml:"port"`
	Description string            `toml:"description"`
	Env         map[string]string `toml:"env,omitempty"`
}

func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading config: %w", err)
	}

	cfg := &Config{}
	if err := toml.Unmarshal(data, cfg); err != nil {
		return nil, fmt.Errorf("parsing config: %w", err)
	}

	cfg.applyDefaults()
	return cfg, nil
}

func (c *Config) applyDefaults() {
	if c.Server.Listen == "" {
		c.Server.Listen = ":8080"
	}
	if c.Server.DataDir == "" {
		c.Server.DataDir = "/var/lib/sdr"
	}
	if c.Auth.SessionSecret == "" {
		c.Auth.SessionSecret = "change-me-in-production"
	}
	if c.SDR.Gain == 0 {
		c.SDR.Gain = 40
	}
	if c.FMRadio.DefaultFreq == "" {
		c.FMRadio.DefaultFreq = "93.0M"
	}
	if c.FMRadio.SampleRate == 0 {
		c.FMRadio.SampleRate = 200000
	}
	if c.FMRadio.AudioBitrate == "" {
		c.FMRadio.AudioBitrate = "192k"
	}
	if c.Ham.DefaultFreq == "" {
		c.Ham.DefaultFreq = "145.500M"
	}
	if c.Ham.DefaultMode == "" {
		c.Ham.DefaultMode = "fm"
	}
	if c.Pager.DefaultFreq == "" {
		c.Pager.DefaultFreq = "153.350M"
	}
	if c.AIS.Gain == 0 {
		c.AIS.Gain = 40
	}
	if c.ISS.RecordFreq == "" {
		c.ISS.RecordFreq = "145.800M"
	}
	if c.ISS.MaxRecordDuration == 0 {
		c.ISS.MaxRecordDuration = 900
	}
	if c.ISS.TLEUpdateInterval == "" {
		c.ISS.TLEUpdateInterval = "6h"
	}
	if c.GSM.Gain == 0 {
		c.GSM.Gain = 49
	}
	if c.GSM.ThresholdGSM900 == 0 {
		c.GSM.ThresholdGSM900 = -10.0
	}
	if c.GSM.ThresholdDCS1800 == 0 {
		c.GSM.ThresholdDCS1800 = -5.0
	}
}
