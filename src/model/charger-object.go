package model

type ChargerObject struct {
	Id            string
	DeviceId      string
	Name          string
	IsOnline      bool
	OperatingMode string
	Power         float64
	Energy        float64
}
