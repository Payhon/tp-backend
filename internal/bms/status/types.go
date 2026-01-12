package status

type ProductionDate struct {
	Raw   uint16 `json:"raw"`
	Year  int    `json:"year"`
	Month int    `json:"month"`
	Day   int    `json:"day"`
}

type TempIndexValue struct {
	Index  int      `json:"index"`
	ValueC *float64 `json:"valueC"`
}

type CellVoltageIndex struct {
	Highest int `json:"highest"`
	Lowest  int `json:"lowest"`
}

type Meta struct {
	SeriesCount     int            `json:"seriesCount"`
	CellTempCount   int            `json:"cellTempCount"`
	HardwareVersion float64        `json:"hardwareVersion"`
	SoftwareVersion float64        `json:"softwareVersion"`
	SpecialID       int            `json:"specialId"`
	ProtocolVersion int            `json:"protocolVersion"`
	ProductionDate  ProductionDate `json:"productionDate"`
}

type Energy struct {
	DesignCapacityMah      uint32  `json:"designCapacityMah"`
	RemainingCapacityMah   uint32  `json:"remainingCapacityMah"`
	FullCapacityMah        uint32  `json:"fullCapacityMah"`
	FullWh                 float64 `json:"fullWh"`
	RemainingWh            float64 `json:"remainingWh"`
	SocPct                 float64 `json:"socPct"`
	SohPct                 float64 `json:"sohPct"`
	CycleCount             uint16  `json:"cycleCount"`
	TotalChargeCapacityRaw uint32  `json:"totalChargeCapacityRaw"`
}

type Timing struct {
	MaxChargeIntervalHours     uint16 `json:"maxChargeIntervalHours"`
	CurrentChargeIntervalHours uint16 `json:"currentChargeIntervalHours"`
	DischargeRemainingMin      uint16 `json:"dischargeRemainingMin"`
	ChargeRemainingMin         uint16 `json:"chargeRemainingMin"`
	ChargeCount                uint16 `json:"chargeCount"`
	DischargeCount             uint16 `json:"dischargeCount"`
	BmsTimestamp               uint32 `json:"bmsTimestamp"`
	PowerOnWorkHours           uint32 `json:"powerOnWorkHours"`
}

type Electrical struct {
	PackCellSumVoltageV  float64          `json:"packCellSumVoltageV"`
	VBatV                float64          `json:"vBatV"`
	VPackV               float64          `json:"vPackV"`
	VLoadV               float64          `json:"vLoadV"`
	CurrentA             float64          `json:"currentA"`
	HighestCellVoltageMv uint16           `json:"highestCellVoltageMv"`
	LowestCellVoltageMv  uint16           `json:"lowestCellVoltageMv"`
	AvgCellVoltageMv     uint16           `json:"avgCellVoltageMv"`
	MaxCellVoltageDiffMv uint16           `json:"maxCellVoltageDiffMv"`
	CellVoltageIndex     CellVoltageIndex `json:"cellVoltageIndex"`
}

type Temperature struct {
	ChargeMosC    *float64       `json:"chargeMosC"`
	DischargeMosC *float64       `json:"dischargeMosC"`
	PrechargeMosC *float64       `json:"prechargeMosC"`
	AmbientC      *float64       `json:"ambientC"`
	HeatingFilmC  *float64       `json:"heatingFilmC"`
	PoleC         *float64       `json:"poleC"`
	HighestTemp   TempIndexValue `json:"highestTemp"`
	LowestTemp    TempIndexValue `json:"lowestTemp"`
	CellTempsC    []*float64     `json:"cellTempsC"`
}

type Cell struct {
	VoltagesMv []uint16 `json:"voltagesMv"`
	Balancing  []bool   `json:"balancing"`
}

type StatusBits struct {
	ProtectionStatus map[string]bool `json:"protectionStatus"`
	IndicatorStatus  map[string]bool `json:"indicatorStatus"`
	AlarmStatus      map[string]bool `json:"alarmStatus"`
	CustomStatus     uint32          `json:"customStatus"`
}

type Identity struct {
	HardwareModel  string  `json:"hardwareModel"`
	BatteryGroupID string  `json:"batteryGroupId"`
	BoardCode      string  `json:"boardCode"`
	BluetoothMac   *string `json:"bluetoothMac"`
}

type BmsStatus struct {
	Meta         Meta        `json:"meta"`
	Energy       Energy      `json:"energy"`
	Timing       Timing      `json:"timing"`
	Electrical   Electrical  `json:"electrical"`
	Temperature  Temperature `json:"temperature"`
	Cell         Cell        `json:"cell"`
	Status       StatusBits  `json:"status"`
	Identity     Identity    `json:"identity"`
	CustomParams []uint16    `json:"customParams"`
}
