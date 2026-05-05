package bmsbridge

import (
	"os"
	"sync"
	"time"

	"gopkg.in/yaml.v3"
)

type EventRules struct {
	Enabled          bool     `yaml:"enabled"`
	EmitOnChange     bool     `yaml:"emit_on_change"`
	MethodPrefix     string   `yaml:"method_prefix"`
	TrackKeyPrefixes []string `yaml:"track_key_prefixes"`
}

type DBSyncRules struct {
	Enabled         bool              `yaml:"enabled"`
	DeviceBatteries map[string]string `yaml:"device_batteries"`
}

type RuleSet struct {
	Telemetry  map[string]string `yaml:"telemetry"`
	Attributes map[string]string `yaml:"attributes"`
	Events     EventRules        `yaml:"events"`
	DBSync     DBSyncRules       `yaml:"db_sync"`
}

type RulesFile struct {
	Version  int                `yaml:"version"`
	Default  RuleSet            `yaml:"default"`
	ByDevice map[string]RuleSet `yaml:"by_device"`
}

func (r RuleSet) withDefaults() RuleSet {
	out := r
	if out.Events.MethodPrefix == "" {
		out.Events.MethodPrefix = "bms."
	}
	if out.Events.TrackKeyPrefixes == nil {
		out.Events.TrackKeyPrefixes = []string{"status.alarmStatus.", "status.protectionStatus.", "status.failureStatus."}
	}
	return out
}

type FileRulesProvider struct {
	path           string
	reloadInterval time.Duration

	mu          sync.Mutex
	lastCheckAt time.Time
	lastMtime   time.Time
	loaded      RulesFile
	loadErr     error
}

func NewFileRulesProvider(path string, reloadIntervalSec int) *FileRulesProvider {
	if reloadIntervalSec < 1 {
		reloadIntervalSec = 1
	}
	return &FileRulesProvider{
		path:           path,
		reloadInterval: time.Duration(reloadIntervalSec) * time.Second,
	}
}

func (p *FileRulesProvider) loadLocked() {
	st, err := os.Stat(p.path)
	if err != nil {
		p.loadErr = err
		return
	}
	if !p.lastMtime.IsZero() && !st.ModTime().After(p.lastMtime) && p.loadErr == nil {
		return
	}

	raw, err := os.ReadFile(p.path)
	if err != nil {
		p.loadErr = err
		return
	}

	var rf RulesFile
	if err := yaml.Unmarshal(raw, &rf); err != nil {
		p.loadErr = err
		return
	}

	// Defaults
	rf.Default = rf.Default.withDefaults()
	for k, rs := range rf.ByDevice {
		rf.ByDevice[k] = rs.withDefaults()
	}

	p.loaded = rf
	p.lastMtime = st.ModTime()
	p.loadErr = nil
}

func (p *FileRulesProvider) Get(deviceID string) (RuleSet, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	now := time.Now()
	if p.lastCheckAt.IsZero() || now.Sub(p.lastCheckAt) >= p.reloadInterval {
		p.lastCheckAt = now
		p.loadLocked()
	}
	if p.loadErr != nil {
		return RuleSet{}, p.loadErr
	}
	if rs, ok := p.loaded.ByDevice[deviceID]; ok {
		return rs, nil
	}
	return p.loaded.Default, nil
}
