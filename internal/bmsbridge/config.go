package bmsbridge

import (
	"fmt"
	"strings"

	"github.com/spf13/viper"
)

type Config struct {
	Enabled bool `mapstructure:"enabled"`

	MQTT struct {
		Broker   string `mapstructure:"broker"`
		User     string `mapstructure:"user"`
		Pass     string `mapstructure:"pass"`
		ClientID string `mapstructure:"client_id"`

		SubscribeTopic string `mapstructure:"subscribe_topic"`
		SubscribeQoS   byte   `mapstructure:"subscribe_qos"`

		TelemetryTopic        string `mapstructure:"telemetry_topic"`
		TelemetryQoS          byte   `mapstructure:"telemetry_qos"`
		AttributesTopicPrefix string `mapstructure:"attributes_topic_prefix"`
		AttributesQoS         byte   `mapstructure:"attributes_qos"`
		EventsTopicPrefix     string `mapstructure:"events_topic_prefix"`
		EventsQoS             byte   `mapstructure:"events_qos"`

		PublishTimeoutMs int `mapstructure:"publish_timeout_ms"`
	} `mapstructure:"mqtt"`

	Report struct {
		FunctionCode       byte   `mapstructure:"function_code"`
		StatusStartAddress uint16 `mapstructure:"status_start_address"`
	} `mapstructure:"report"`

	Rules struct {
		Path              string `mapstructure:"path"`
		ReloadIntervalSec int    `mapstructure:"reload_interval_sec"`
	} `mapstructure:"rules"`

	Workers struct {
		Concurrency int `mapstructure:"concurrency"`
		QueueSize   int `mapstructure:"queue_size"`
	} `mapstructure:"workers"`

	Events struct {
		StateTTLMinutes int `mapstructure:"state_ttl_minutes"`
	} `mapstructure:"events"`
}

func normalizeBrokerURL(broker string) string {
	broker = strings.TrimSpace(broker)
	if broker == "" {
		return ""
	}
	if strings.Contains(broker, "://") {
		return broker
	}
	return "tcp://" + broker
}

func LoadConfigFromViper(v *viper.Viper) (Config, error) {
	var cfg Config
	cfg.Enabled = true

	cfg.MQTT.SubscribeTopic = "device/socket/tx/+"
	cfg.MQTT.SubscribeQoS = 1
	cfg.MQTT.TelemetryTopic = "devices/telemetry"
	cfg.MQTT.TelemetryQoS = 0
	cfg.MQTT.AttributesTopicPrefix = "devices/attributes/"
	cfg.MQTT.AttributesQoS = 1
	cfg.MQTT.EventsTopicPrefix = "devices/event/"
	cfg.MQTT.EventsQoS = 1
	cfg.MQTT.PublishTimeoutMs = 3000

	cfg.Report.FunctionCode = 0xDD
	cfg.Report.StatusStartAddress = 0x100

	cfg.Rules.Path = "./configs/bms-bridge-rules.yml"
	cfg.Rules.ReloadIntervalSec = 5

	cfg.Workers.Concurrency = 8
	cfg.Workers.QueueSize = 4096

	cfg.Events.StateTTLMinutes = 60

	sub := v.Sub("bms_bridge")
	if sub != nil {
		if err := sub.Unmarshal(&cfg); err != nil {
			return Config{}, err
		}
	}

	if !cfg.Enabled {
		return cfg, nil
	}

	if cfg.MQTT.Broker == "" {
		// Fallback to backend mqtt config.
		cfg.MQTT.Broker = v.GetString("mqtt.broker")
		if cfg.MQTT.Broker == "" {
			cfg.MQTT.Broker = v.GetString("mqtt.access_address")
		}
	}
	cfg.MQTT.Broker = normalizeBrokerURL(cfg.MQTT.Broker)
	if cfg.MQTT.Broker == "" {
		return Config{}, fmt.Errorf("bms_bridge.mqtt.broker is required")
	}

	if cfg.MQTT.User == "" {
		cfg.MQTT.User = v.GetString("mqtt.user")
	}
	if cfg.MQTT.Pass == "" {
		cfg.MQTT.Pass = v.GetString("mqtt.pass")
	}

	if cfg.MQTT.ClientID == "" {
		cfg.MQTT.ClientID = "fjbms-bms-bridge"
	}
	if cfg.MQTT.SubscribeTopic == "" {
		return Config{}, fmt.Errorf("bms_bridge.mqtt.subscribe_topic is required")
	}
	if cfg.MQTT.PublishTimeoutMs <= 0 {
		cfg.MQTT.PublishTimeoutMs = 3000
	}
	if cfg.Workers.Concurrency < 1 {
		cfg.Workers.Concurrency = 1
	}
	if cfg.Workers.QueueSize < 1 {
		cfg.Workers.QueueSize = 1
	}
	if cfg.Rules.ReloadIntervalSec < 1 {
		cfg.Rules.ReloadIntervalSec = 1
	}
	if cfg.Events.StateTTLMinutes < 1 {
		cfg.Events.StateTTLMinutes = 1
	}

	return cfg, nil
}
