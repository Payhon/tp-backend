package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"math/rand/v2"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"
	"time"

	"project/internal/adapter/mqttadapter"
	"project/internal/app"
	"project/internal/bms/protocol"

	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/sirupsen/logrus"
)

type socketPayload struct {
	Hex string `json:"hex"`
}

type regStore struct {
	mu   sync.RWMutex
	regs map[uint16]uint16
}

func newRegStore() *regStore {
	return &regStore{regs: make(map[uint16]uint16, 4096)}
}

func (s *regStore) get(addr uint16) uint16 {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if v, ok := s.regs[addr]; ok {
		return v
	}
	return 0xFFFF
}

func (s *regStore) set(addr uint16, v uint16) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.regs[addr] = v
}

func decodeSocketHex(payload []byte) (string, error) {
	var body socketPayload
	if err := json.Unmarshal(payload, &body); err == nil && body.Hex != "" {
		return body.Hex, nil
	}
	s := strings.TrimSpace(string(payload))
	if s == "" {
		return "", fmt.Errorf("empty payload")
	}
	return s, nil
}

func regsToBEBytes(regs []uint16) []byte {
	out := make([]byte, len(regs)*2)
	for i := 0; i < len(regs); i++ {
		v := regs[i]
		out[i*2] = byte((v >> 8) & 0xFF)
		out[i*2+1] = byte(v & 0xFF)
	}
	return out
}

func encodeAsciiFixed(str string, byteLength int) []byte {
	out := make([]byte, byteLength)
	for i := 0; i < len(out); i++ {
		out[i] = 0x00
	}
	s := str
	if s == "" {
		return out
	}
	n := len(s)
	if n > byteLength-1 {
		n = byteLength - 1
	}
	copy(out[:n], []byte(s)[:n])
	return out
}

func writeAsciiToRegs(store *regStore, startAddress uint16, byteLength int, str string) {
	bytes := encodeAsciiFixed(str, byteLength)
	regsNeeded := (byteLength + 1) / 2
	for i := 0; i < regsNeeded; i++ {
		hi := bytes[i*2]
		lo := byte(0x00)
		if i*2+1 < len(bytes) {
			lo = bytes[i*2+1]
		}
		store.set(startAddress+uint16(i), uint16(hi)<<8|uint16(lo))
	}
}

func seedDefaultStatus(store *regStore, seriesCount int, tempCount int, deviceID string) {
	if seriesCount < 1 {
		seriesCount = 16
	}
	if tempCount < 1 {
		tempCount = 4
	}
	store.set(0x100, uint16((seriesCount&0xFF)<<8|(tempCount&0xFF)))
	store.set(0x101, uint16((10<<8)|10)) // 1.0 / 1.0
	store.set(0x102, uint16((1<<8)|1))   // specialId/protocolVersion
	// SOC/SOH at 0x10D: H=soc*2, L=soh
	store.set(0x10D, uint16((80<<8)|95)) // soc=40%, soh=95%

	// Voltages
	store.set(0x115, uint16(520)) // 52.0V
	store.set(0x116, uint16(520))
	store.set(0x117, uint16(520))
	store.set(0x118, uint16(520))

	// Current i32 at 0x119: 0.1mA/bit, set to 12.3A -> 123000 bits
	rawCurrent := uint32(123000)
	store.set(0x119, uint16((rawCurrent>>16)&0xFFFF))
	store.set(0x11A, uint16(rawCurrent&0xFFFF))

	// Temperatures bytes with -40 offset stored as u8
	store.set(0x11D, uint16((40<<8)|40)) // 0°C for both MOS
	store.set(0x11E, uint16((40<<8)|40)) // 0°C precharge/ambient
	store.set(0x11F, uint16((40<<8)|40)) // 0°C heating/pole

	// Highest/lowest temp index/value
	store.set(0x126, uint16((1<<8)|40))
	store.set(0x127, uint16((2<<8)|40))
	store.set(0x128, uint16((1<<8)|2))

	// Protection/indicator/alarm bitfields
	store.set(0x12D, 0x0000)
	store.set(0x12E, 0x0000)
	store.set(0x132, 0x0000)
	store.set(0x133, 0x0000)
	store.set(0x134, 0x0000)
	store.set(0x135, 0x0000)

	// Production date raw: 2025-01-12 encoded (year=25, month=1, day=12)
	year := uint16(25)
	month := uint16(1)
	day := uint16(12)
	prod := (year << 9) | (month << 5) | day
	store.set(0x138, prod)

	// Custom params 0x139~0x140 (8 regs) fill zeros
	for i := 0; i < 8; i++ {
		store.set(0x139+uint16(i), 0)
	}

	// Cell voltages 0x141.. (mV)
	for i := 0; i < seriesCount; i++ {
		store.set(0x141+uint16(i), uint16(3250+rand.IntN(30)))
	}

	// Cell temps follow: Kelvin*10 with offset: 25C => 2981
	cellTempsStart := uint16(0x141 + seriesCount)
	for i := 0; i < tempCount; i++ {
		store.set(cellTempsStart+uint16(i), 2981)
	}

	hwModelStart := cellTempsStart + uint16(tempCount)
	batteryGroupStart := hwModelStart + 16
	boardCodeStart := batteryGroupStart + 16
	macStart := boardCodeStart + 16

	writeAsciiToRegs(store, hwModelStart, 32, "FJBMS-HW-MODEL")
	writeAsciiToRegs(store, batteryGroupStart, 32, "GROUP-"+deviceID[:min(8, len(deviceID))])
	writeAsciiToRegs(store, boardCodeStart, 32, "BOARD-"+deviceID[:min(8, len(deviceID))])

	// Bluetooth MAC bytes: 10 bytes, first 6 used.
	macBytes := []byte{0xAA, 0xBB, 0xCC, 0xDD, 0xEE, 0xFF, 0x00, 0x00, 0x00, 0x00}
	for i := 0; i < 5; i++ {
		hi := macBytes[i*2]
		lo := macBytes[i*2+1]
		store.set(macStart+uint16(i), uint16(hi)<<8|uint16(lo))
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func statusTotalRegs(seriesCount, tempCount int) int {
	cellVoltagesStart := 0x141
	macStart := cellVoltagesStart + seriesCount + tempCount + 16 + 16 + 16
	macRegs := 5
	lastAddr := macStart + macRegs - 1
	return lastAddr - 0x100 + 1
}

func main() {
	var (
		configPath = flag.String("config", "", "后端配置文件路径（可选，用于复用 mqtt.*）")

		broker   = flag.String("broker", "", "MQTT broker (tcp://host:port 或 host:port)")
		user     = flag.String("user", "", "MQTT username")
		pass     = flag.String("pass", "", "MQTT password")
		clientID = flag.String("client-id", "fjbms-bms-sim", "MQTT client id")

		deviceID = flag.String("device-id", "", "设备ID(UUID)，用于 topic device/socket/*/{device_id}")
		bmsAddr  = flag.Uint("bms-addr", 0x01, "设备地址(1-254)，用于协议帧 target/source address")

		subscribeRx = flag.Bool("subscribe-rx", true, "订阅 device/socket/rx/{device_id} 并自动响应读写请求")

		reportEnable   = flag.Bool("report", true, "是否主动上报")
		reportInterval = flag.Duration("report-interval", 5*time.Second, "上报间隔")
		reportFunc     = flag.Uint("report-func", 0xDD, "上报功能码（默认 0xDD）")
		reportStart    = flag.Uint("report-start", 0x100, "上报起始寄存器地址（用于 write 格式上报）")
		reportQty      = flag.Uint("report-quantity", 0, "上报寄存器数量（0 表示自动推导 status 总长度）")
		reportFormat   = flag.String("report-format", "write", "上报帧格式：write|read（write=携带起始地址，read=仅数据）")

		seriesCount = flag.Int("status-series", 16, "status 上报：电池串数 S（仅当 report-start=0x100 且 report-quantity=0 时生效）")
		tempCount   = flag.Int("status-temps", 4, "status 上报：电芯温度数量 N（仅当 report-start=0x100 且 report-quantity=0 时生效）")

		verbose = flag.Bool("v", false, "打印更多调试日志")
	)
	flag.Parse()

	logrus.SetFormatter(&logrus.TextFormatter{FullTimestamp: true})
	log := logrus.StandardLogger()
	if *verbose {
		log.SetLevel(logrus.DebugLevel)
	} else {
		log.SetLevel(logrus.InfoLevel)
	}

	if *deviceID == "" {
		fmt.Fprintln(os.Stderr, "missing -device-id")
		os.Exit(2)
	}

	if *configPath != "" {
		v, err := app.LoadConfigFile(*configPath)
		if err != nil {
			log.WithError(err).Fatal("load config failed")
		}
		if *broker == "" {
			*broker = v.GetString("mqtt.broker")
			if *broker == "" {
				*broker = v.GetString("mqtt.access_address")
			}
		}
		if *user == "" {
			*user = v.GetString("mqtt.user")
		}
		if *pass == "" {
			*pass = v.GetString("mqtt.pass")
		}
	}
	if *broker == "" {
		fmt.Fprintln(os.Stderr, "missing -broker (or provide -config with mqtt.broker)")
		os.Exit(2)
	}
	if !strings.Contains(*broker, "://") {
		*broker = "tcp://" + *broker
	}

	store := newRegStore()
	seedDefaultStatus(store, *seriesCount, *tempCount, *deviceID)

	mqttCfg := mqttadapter.MQTTConfig{
		Broker:   *broker,
		Username: *user,
		Password: *pass,
		ClientID: *clientID,
	}
	mc, err := mqttadapter.CreateMQTTClient(mqttCfg, log)
	if err != nil {
		log.WithError(err).Fatal("mqtt connect failed")
	}
	defer mqttadapter.DisconnectMQTTClient(mc, log)

	txTopic := fmt.Sprintf("device/socket/tx/%s", *deviceID)
	rxTopic := fmt.Sprintf("device/socket/rx/%s", *deviceID)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if *subscribeRx {
		token := mc.Subscribe(rxTopic, 1, func(_ mqtt.Client, m mqtt.Message) {
			hexStr, err := decodeSocketHex(m.Payload())
			if err != nil {
				log.WithError(err).Warn("decode payload failed")
				return
			}
			frameBytes, err := protocol.DecodeHexString(hexStr)
			if err != nil {
				log.WithError(err).Warn("decode hex failed")
				return
			}
			parsed, err := protocol.ParseFrame(frameBytes)
			if err != nil {
				log.WithError(err).Warn("parse frame failed")
				return
			}

			switch f := parsed.(type) {
			case protocol.ReadRequestFrame:
				if *verbose {
					log.WithFields(logrus.Fields{
						"func":         fmt.Sprintf("0x%02X", f.FunctionCode),
						"startAddress": fmt.Sprintf("0x%04X", f.StartAddress),
						"quantity":     f.Quantity,
					}).Debug("rx read request")
				}

				regs := make([]uint16, int(f.Quantity))
				for i := 0; i < int(f.Quantity); i++ {
					regs[i] = store.get(f.StartAddress + uint16(i))
				}
				data := regsToBEBytes(regs)
				if f.FunctionCode == 0x0F {
					prefix := []byte{byte(f.StartAddress >> 8), byte(f.StartAddress & 0xFF), byte(f.Quantity >> 8), byte(f.Quantity & 0xFF)}
					data = append(prefix, data...)
				}

				resp := protocol.BuildReadResponseFrame(byte(*bmsAddr), f.SourceAddress, f.FunctionCode, data)
				out := socketPayload{Hex: protocol.BytesToHexUpper(resp)}
				bs, _ := json.Marshal(out)
				mc.Publish(txTopic, 1, false, bs)

			case protocol.WriteRequestFrame:
				if *verbose {
					log.WithFields(logrus.Fields{
						"func":         fmt.Sprintf("0x%02X", f.FunctionCode),
						"startAddress": fmt.Sprintf("0x%04X", f.StartAddress),
						"quantity":     f.Quantity,
						"byteCount":    f.ByteCount,
					}).Debug("rx write request")
				}
				regs, err := protocol.SplitIntoRegistersBE(f.Data)
				if err != nil {
					log.WithError(err).Warn("split registers failed")
					return
				}
				for i := 0; i < len(regs); i++ {
					store.set(f.StartAddress+uint16(i), regs[i])
				}
				resp := protocol.BuildWriteResponseFrame(byte(*bmsAddr), f.SourceAddress, f.FunctionCode, f.StartAddress, f.Quantity)
				out := socketPayload{Hex: protocol.BytesToHexUpper(resp)}
				bs, _ := json.Marshal(out)
				mc.Publish(txTopic, 1, false, bs)

			default:
				if *verbose {
					log.WithField("type", fmt.Sprintf("%T", parsed)).Debug("rx frame ignored")
				}
			}
		})
		if token.Wait() && token.Error() != nil {
			log.WithError(token.Error()).Fatal("subscribe rx failed")
		}
		log.WithField("topic", rxTopic).Info("subscribed rx")
	}

	if *reportEnable {
		go func() {
			t := time.NewTicker(*reportInterval)
			defer t.Stop()
			for {
				select {
				case <-ctx.Done():
					return
				case <-t.C:
					start := uint16(*reportStart)
					qty := int(*reportQty)
					if qty == 0 && start == 0x100 {
						qty = statusTotalRegs(*seriesCount, *tempCount)
					}
					if qty <= 0 {
						continue
					}

					regs := make([]uint16, qty)
					for i := 0; i < qty; i++ {
						regs[i] = store.get(start + uint16(i))
					}

					funcCode := byte(*reportFunc & 0xFF)
					var frame []byte
					switch strings.ToLower(strings.TrimSpace(*reportFormat)) {
					case "read":
						data := regsToBEBytes(regs)
						if funcCode == 0x0F {
							prefix := []byte{byte(start >> 8), byte(start & 0xFF), byte(qty >> 8), byte(qty & 0xFF)}
							data = append(prefix, data...)
						}
						// read response style does NOT carry start address; bms_bridge will use its configured status_start_address.
						frame = protocol.BuildReadResponseFrame(byte(*bmsAddr), 0xFE, funcCode, data)
					default:
						// write request style carries start address + data, so bms_bridge can parse arbitrary ranges.
						frame = protocol.BuildWriteRequestFrame(byte(*bmsAddr), 0xFE, funcCode, start, regs)
					}

					out := socketPayload{Hex: protocol.BytesToHexUpper(frame)}
					bs, _ := json.Marshal(out)
					mc.Publish(txTopic, 1, false, bs)
					if *verbose {
						log.WithFields(logrus.Fields{
							"topic":        txTopic,
							"func":         fmt.Sprintf("0x%02X", funcCode),
							"startAddress": fmt.Sprintf("0x%04X", start),
							"quantity":     qty,
							"format":       *reportFormat,
						}).Debug("sent report")
					}
				}
			}
		}()
		log.WithFields(logrus.Fields{
			"topic":           txTopic,
			"report_interval": (*reportInterval).String(),
		}).Info("report loop started")
	}

	// wait for signal
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	<-sigCh
	cancel()
}
