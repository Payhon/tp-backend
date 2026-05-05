package bmsbridge

import (
	"fmt"
	"reflect"

	"project/internal/bms/status"
)

func FlattenStatus(s status.BmsStatus) map[string]any {
	out := make(map[string]any, 256)

	out["meta.seriesCount"] = s.Meta.SeriesCount
	out["meta.cellTempCount"] = s.Meta.CellTempCount
	out["meta.hardwareVersion"] = s.Meta.HardwareVersion
	out["meta.softwareVersion"] = s.Meta.SoftwareVersion
	out["meta.specialId"] = s.Meta.SpecialID
	out["meta.protocolVersion"] = s.Meta.ProtocolVersion
	out["meta.productionDate.raw"] = s.Meta.ProductionDate.Raw
	out["meta.productionDate.year"] = s.Meta.ProductionDate.Year
	out["meta.productionDate.month"] = s.Meta.ProductionDate.Month
	out["meta.productionDate.day"] = s.Meta.ProductionDate.Day

	out["energy.designCapacityMah"] = s.Energy.DesignCapacityMah
	out["energy.remainingCapacityMah"] = s.Energy.RemainingCapacityMah
	out["energy.fullCapacityMah"] = s.Energy.FullCapacityMah
	out["energy.fullWh"] = s.Energy.FullWh
	out["energy.remainingWh"] = s.Energy.RemainingWh
	out["energy.socPct"] = s.Energy.SocPct
	out["energy.sohPct"] = s.Energy.SohPct
	out["energy.cycleCount"] = s.Energy.CycleCount
	out["energy.totalChargeCapacityRaw"] = s.Energy.TotalChargeCapacityRaw
	out["energy.totalDischargeCapacityRaw"] = s.Energy.TotalDischargeCapacityRaw

	out["timing.maxChargeIntervalHours"] = s.Timing.MaxChargeIntervalHours
	out["timing.currentChargeIntervalHours"] = s.Timing.CurrentChargeIntervalHours
	out["timing.dischargeRemainingMin"] = s.Timing.DischargeRemainingMin
	out["timing.chargeRemainingMin"] = s.Timing.ChargeRemainingMin
	out["timing.chargeCount"] = s.Timing.ChargeCount
	out["timing.dischargeCount"] = s.Timing.DischargeCount
	out["timing.bmsTimestamp"] = s.Timing.BmsTimestamp
	out["timing.powerOnWorkHours"] = s.Timing.PowerOnWorkHours

	out["electrical.packCellSumVoltageV"] = s.Electrical.PackCellSumVoltageV
	out["electrical.vBatV"] = s.Electrical.VBatV
	out["electrical.vPackV"] = s.Electrical.VPackV
	out["electrical.vLoadV"] = s.Electrical.VLoadV
	out["electrical.currentA"] = s.Electrical.CurrentA
	out["electrical.highestCellVoltageMv"] = s.Electrical.HighestCellVoltageMv
	out["electrical.lowestCellVoltageMv"] = s.Electrical.LowestCellVoltageMv
	out["electrical.avgCellVoltageMv"] = s.Electrical.AvgCellVoltageMv
	out["electrical.maxCellVoltageDiffMv"] = s.Electrical.MaxCellVoltageDiffMv
	out["electrical.cellVoltageIndex.highest"] = s.Electrical.CellVoltageIndex.Highest
	out["electrical.cellVoltageIndex.lowest"] = s.Electrical.CellVoltageIndex.Lowest

	out["temperature.chargeMosC"] = s.Temperature.ChargeMosC
	out["temperature.dischargeMosC"] = s.Temperature.DischargeMosC
	out["temperature.prechargeMosC"] = s.Temperature.PrechargeMosC
	out["temperature.ambientC"] = s.Temperature.AmbientC
	out["temperature.heatingFilmC"] = s.Temperature.HeatingFilmC
	out["temperature.poleC"] = s.Temperature.PoleC
	out["temperature.highestTemp.index"] = s.Temperature.HighestTemp.Index
	out["temperature.highestTemp.valueC"] = s.Temperature.HighestTemp.ValueC
	out["temperature.lowestTemp.index"] = s.Temperature.LowestTemp.Index
	out["temperature.lowestTemp.valueC"] = s.Temperature.LowestTemp.ValueC
	out["temperature.cellTempsC"] = s.Temperature.CellTempsC

	out["cell.voltagesMv"] = s.Cell.VoltagesMv
	out["cell.balancing"] = s.Cell.Balancing

	for k, v := range s.Status.ProtectionStatus {
		out[fmt.Sprintf("status.protectionStatus.%s", k)] = v
	}
	for k, v := range s.Status.FailureStatus {
		out[fmt.Sprintf("status.failureStatus.%s", k)] = v
	}
	for k, v := range s.Status.IndicatorStatus {
		out[fmt.Sprintf("status.indicatorStatus.%s", k)] = v
	}
	for k, v := range s.Status.AlarmStatus {
		out[fmt.Sprintf("status.alarmStatus.%s", k)] = v
	}
	out["status.customStatus"] = s.Status.CustomStatus

	out["identity.hardwareModel"] = s.Identity.HardwareModel
	out["identity.batteryGroupId"] = s.Identity.BatteryGroupID
	out["identity.boardCode"] = s.Identity.BoardCode
	out["identity.bluetoothMac"] = s.Identity.BluetoothMac

	out["customParams"] = s.CustomParams

	return out
}

func selectValues(flat map[string]any, mapping map[string]string) map[string]any {
	if len(mapping) == 0 {
		return nil
	}
	out := make(map[string]any, len(mapping))
	for outKey, flatKey := range mapping {
		v, ok := flat[flatKey]
		if !ok || isNilLike(v) {
			continue
		}
		out[outKey] = v
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

func isNilLike(v any) bool {
	if v == nil {
		return true
	}
	rv := reflect.ValueOf(v)
	switch rv.Kind() {
	case reflect.Pointer, reflect.Map, reflect.Slice, reflect.Interface:
		return rv.IsNil()
	default:
		return false
	}
}
