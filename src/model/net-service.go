package model

import (
	"fmt"

	"github.com/futurehomeno/fimpgo/fimptype"
)

type NetworkService struct {
}

func (ns *NetworkService) MakeInclusionReport(deviceId string, name string) fimptype.ThingInclusionReport {

	var manufacturer string
	services := []fimptype.Service{}

	carchargerInterfaces := []fimptype.Interface{{
		Type:      "in",
		MsgType:   "cmd.state.get_report",
		ValueType: "null",
		Version:   "1",
	}, {
		Type:      "out",
		MsgType:   "evt.state.report",
		ValueType: "string",
		Version:   "1",
	}, {
		Type:      "in",
		MsgType:   "cmd.current_session.get_report",
		ValueType: "null",
		Version:   "1",
	}, {
		Type:      "out",
		MsgType:   "evt.current_session.report",
		ValueType: "float",
		Version:   "1",
	}, {
		Type:      "in",
		MsgType:   "cmd.charge.start",
		ValueType: "null",
		Version:   "1",
	}, {
		Type:      "out",
		MsgType:   "cmd.charge.stop",
		ValueType: "bool",
		Version:   "1",
	}}

	powerInterfaces := []fimptype.Interface{{
		Type:      "in",
		MsgType:   "cmd.meter.get_report",
		ValueType: "null",
		Version:   "1",
	}, {
		Type:      "out",
		MsgType:   "evt.meter.report",
		ValueType: "float",
		Version:   "1",
	}}

	carchargerService := fimptype.Service{
		Name:    "chargepoint",
		Alias:   "chargepoint",
		Address: "/rt:dev/rn:defa/ad:1/sv:chargepoint/ad:",
		Enabled: true,
		Groups:  []string{"ch_0"},
		Props: map[string]interface{}{
			"sup_states": []string{"disconnected", "requesting", "charging", "finished", "unknown"},
		},
		Interfaces: carchargerInterfaces,
	}

	powerService := fimptype.Service{
		Name:    "meter_elec",
		Alias:   "meter_elec",
		Address: "/rt:dev/rn:defa/ad:1/sv:meter_elec/ad:",
		Enabled: true,
		Groups:  []string{"ch_0"},
		Props: map[string]interface{}{
			"sup_units": []string{"W"},
		},
		Interfaces: powerInterfaces,
	}

	serviceAddress := fmt.Sprintf("%s", deviceId)
	carchargerService.Address = carchargerService.Address + serviceAddress
	powerService.Address = powerService.Address + serviceAddress
	services = append(services, carchargerService, powerService)
	deviceAddr := fmt.Sprintf("%s", deviceId)
	powerSource := "ac"

	inclReport := fimptype.ThingInclusionReport{
		IntegrationId:     "",
		Address:           deviceAddr,
		Type:              "",
		ProductHash:       manufacturer,
		CommTechnology:    "wifi",
		ProductName:       name,
		ManufacturerId:    manufacturer,
		DeviceId:          deviceId,
		HwVersion:         "1",
		SwVersion:         "1",
		PowerSource:       powerSource,
		WakeUpInterval:    "-1",
		Security:          "",
		Tags:              nil,
		Groups:            []string{"ch_0"},
		PropSets:          nil,
		TechSpecificProps: nil,
		Services:          services,
	}

	return inclReport

}
