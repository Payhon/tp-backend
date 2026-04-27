package bmsbridge

import (
	"context"
	"strconv"
	"strings"

	"project/internal/bms/status"
)

type dynamicStatusMeta struct {
	SeriesCount   int
	CellTempCount int
}

func (m dynamicStatusMeta) valid() bool {
	return m.SeriesCount > 0 && m.CellTempCount >= 0
}

func (b *Bridge) rememberStatusMeta(deviceID string, st status.BmsStatus) {
	deviceID = strings.TrimSpace(deviceID)
	if deviceID == "" || st.Meta.SeriesCount <= 0 {
		return
	}
	b.statusMeta.Store(deviceID, dynamicStatusMeta{
		SeriesCount:   st.Meta.SeriesCount,
		CellTempCount: st.Meta.CellTempCount,
	})
}

func (b *Bridge) decodeDynamicCellReport(ctx context.Context, deviceID string, startAddress uint16, registers []uint16) map[string]any {
	if startAddress != 0x141 || len(registers) == 0 {
		return nil
	}
	meta, ok := b.loadStatusMeta(ctx, deviceID)
	if !ok {
		b.log.WithFields(map[string]any{
			"device_id":     deviceID,
			"start_address": "0x0141",
			"quantity":      len(registers),
		}).Debug("skip dynamic cell report: status metadata unavailable")
		return nil
	}
	if len(registers) < meta.SeriesCount+meta.CellTempCount {
		b.log.WithFields(map[string]any{
			"device_id":       deviceID,
			"quantity":        len(registers),
			"series_count":    meta.SeriesCount,
			"cell_temp_count": meta.CellTempCount,
		}).Debug("skip dynamic cell report: register count shorter than expected")
		return nil
	}

	voltages := make([]uint16, 0, meta.SeriesCount)
	var sum uint32
	var highest uint16
	var lowest uint16
	highestIdx := 0
	lowestIdx := 0
	for i := 0; i < meta.SeriesCount; i++ {
		v := registers[i]
		voltages = append(voltages, v)
		sum += uint32(v)
		if i == 0 || v > highest {
			highest = v
			highestIdx = i + 1
		}
		if i == 0 || v < lowest {
			lowest = v
			lowestIdx = i + 1
		}
	}

	cellTemps := make([]*float64, 0, meta.CellTempCount)
	for i := 0; i < meta.CellTempCount; i++ {
		cellTemps = append(cellTemps, decodeBmsCellTempCFromKelvin10(registers[meta.SeriesCount+i]))
	}

	avg := uint16(0)
	if meta.SeriesCount > 0 {
		avg = uint16(sum / uint32(meta.SeriesCount))
	}
	diff := uint16(0)
	if highest >= lowest {
		diff = highest - lowest
	}

	return map[string]any{
		"cell.voltagesMv":                     voltages,
		"temperature.cellTempsC":              cellTemps,
		"electrical.highestCellVoltageMv":     highest,
		"electrical.lowestCellVoltageMv":      lowest,
		"electrical.maxCellVoltageDiffMv":     diff,
		"electrical.cellVoltageIndex.highest": highestIdx,
		"electrical.cellVoltageIndex.lowest":  lowestIdx,
		"electrical.packCellSumVoltageV":      float64(sum) / 1000,
		"electrical.avgCellVoltageMv":         avg,
	}
}

func (b *Bridge) loadStatusMeta(ctx context.Context, deviceID string) (dynamicStatusMeta, bool) {
	deviceID = strings.TrimSpace(deviceID)
	if deviceID == "" {
		return dynamicStatusMeta{}, false
	}
	if cached, ok := b.statusMeta.Load(deviceID); ok {
		if meta, ok := cached.(dynamicStatusMeta); ok && meta.valid() {
			return meta, true
		}
	}
	meta, ok := b.loadStatusMetaFromCurrentData(ctx, deviceID)
	if ok {
		b.statusMeta.Store(deviceID, meta)
	}
	return meta, ok
}

func (b *Bridge) loadStatusMetaFromCurrentData(ctx context.Context, deviceID string) (dynamicStatusMeta, bool) {
	if b.db == nil {
		return dynamicStatusMeta{}, false
	}
	keys := []string{"seriesCount", "meta.seriesCount", "cellTempCount", "meta.cellTempCount"}
	meta := dynamicStatusMeta{}
	for _, table := range []string{"telemetry_current_datas", "attribute_datas"} {
		var rows []struct {
			Key     string   `gorm:"column:key"`
			NumberV *float64 `gorm:"column:number_v"`
			StringV *string  `gorm:"column:string_v"`
		}
		if err := b.db.WithContext(ctx).
			Table(table).
			Select("key, number_v, string_v").
			Where("device_id = ? AND key IN ?", deviceID, keys).
			Find(&rows).Error; err != nil {
			b.log.WithError(err).WithFields(map[string]any{
				"device_id": deviceID,
				"table":     table,
			}).Debug("load dynamic status metadata failed")
			continue
		}
		for _, row := range rows {
			n, ok := currentNumber(row.NumberV, row.StringV)
			if !ok {
				continue
			}
			switch row.Key {
			case "seriesCount", "meta.seriesCount":
				meta.SeriesCount = n
			case "cellTempCount", "meta.cellTempCount":
				meta.CellTempCount = n
			}
		}
		if meta.valid() {
			return meta, true
		}
	}
	return dynamicStatusMeta{}, false
}

func currentNumber(numberV *float64, stringV *string) (int, bool) {
	if numberV != nil {
		return int(*numberV), true
	}
	if stringV == nil {
		return 0, false
	}
	text := strings.TrimSpace(*stringV)
	if text == "" {
		return 0, false
	}
	if n, err := strconv.Atoi(text); err == nil {
		return n, true
	}
	f, err := strconv.ParseFloat(text, 64)
	if err != nil {
		return 0, false
	}
	return int(f), true
}

func decodeBmsCellTempCFromKelvin10(raw uint16) *float64 {
	if raw == 0xFFFF {
		return nil
	}
	v := (float64(raw) - 2731) / 10
	return &v
}
