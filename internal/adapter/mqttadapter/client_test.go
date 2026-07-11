package mqttadapter

import (
	"testing"

	mqtt "github.com/eclipse/paho.mqtt.golang"
)

func TestApplyDeliveryOptionsKeepsLegacyDefaults(t *testing.T) {
	opts := mqtt.NewClientOptions()
	applyDeliveryOptions(opts, nil)

	if opts.CleanSession {
		t.Fatal("legacy default must keep persistent session")
	}
	if !opts.ResumeSubs {
		t.Fatal("legacy default must keep subscription resume enabled")
	}
	if opts.Order {
		t.Fatal("legacy default must keep concurrent callback dispatch")
	}
}

func TestApplyDeliveryOptionsSupportsRequestResponseBridge(t *testing.T) {
	opts := mqtt.NewClientOptions()
	applyDeliveryOptions(opts, &MQTTDeliveryOptions{
		CleanSession: true,
		ResumeSubs:   false,
		OrderMatters: true,
	})

	if !opts.CleanSession {
		t.Fatal("bridge must use a clean session")
	}
	if opts.ResumeSubs {
		t.Fatal("bridge must not resume offline subscriptions")
	}
	if !opts.Order {
		t.Fatal("bridge must preserve callback order")
	}
}
