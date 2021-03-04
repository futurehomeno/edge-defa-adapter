package router

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/futurehomeno/fimpgo"
	log "github.com/sirupsen/logrus"
	"github.com/thingsplex/defa/model"
)

type FromFimpRouter struct {
	inboundMsgCh fimpgo.MessageCh
	mqt          *fimpgo.MqttTransport
	instanceId   string
	appLifecycle *model.Lifecycle
	configs      *model.Configs
	states       *model.States
	resetToken   *model.ResetToken
	loginToken   *model.LoginToken
}

func NewFromFimpRouter(mqt *fimpgo.MqttTransport, appLifecycle *model.Lifecycle, configs *model.Configs, states *model.States) *FromFimpRouter {
	fc := FromFimpRouter{inboundMsgCh: make(fimpgo.MessageCh, 5), mqt: mqt, appLifecycle: appLifecycle, configs: configs, states: states}
	fc.mqt.RegisterChannel("ch1", fc.inboundMsgCh)
	return &fc
}

func (fc *FromFimpRouter) Start() {

	// TODO: Choose either adapter or app topic

	// ------ Adapter topics ---------------------------------------------
	fc.mqt.Subscribe(fmt.Sprintf("pt:j1/+/rt:dev/rn:%s/ad:1/#", model.ServiceName))
	fc.mqt.Subscribe(fmt.Sprintf("pt:j1/+/rt:ad/rn:%s/ad:1", model.ServiceName))

	// ------ Application topic -------------------------------------------
	//fc.mqt.Subscribe(fmt.Sprintf("pt:j1/+/rt:app/rn:%s/ad:1",model.ServiceName))

	go func(msgChan fimpgo.MessageCh) {
		for {
			select {
			case newMsg := <-msgChan:
				fc.routeFimpMessage(newMsg)
			}
		}
	}(fc.inboundMsgCh)
}

func (fc *FromFimpRouter) routeFimpMessage(newMsg *fimpgo.Message) {
	// resetToken := model.ResetToken{}
	// loginToken := model.LoginToken{}
	ns := model.NetworkService{}
	// var chargerObject model.ChargerObject
	log.Debug("New fimp msg")
	addr := strings.Replace(newMsg.Addr.ServiceAddress, "_0", "", 1)
	switch newMsg.Payload.Service {
	case "chargepoint":
		addr = strings.Replace(addr, "l", "", 1)
		switch newMsg.Payload.Type {
		case "cmd.charge.start":
			// get address
			for _, chargers := range fc.states.Chargers.Data.ReceivingAccess {
				for _, chargepoint := range chargers.ChargePoint.AliasMap {
					if addr == chargepoint.Name {
						err := model.StartCharging(chargers.ChargePoint.ID, chargepoint.Connector, fc.configs.UserID, fc.configs.AccessToken)
						if err != nil {
							log.Error(err)
						} else {
							msgOperatingMode := fimpgo.NewMessage("evt.state.report", "defa", fimpgo.VTypeString, model.SetStatus(("charging")), nil, nil, nil)
							msgOperatingMode.Source = "defa"
							adrOperatingMode := &fimpgo.Address{
								MsgType:         fimpgo.MsgTypeEvt,
								ResourceType:    fimpgo.ResourceTypeDevice,
								ResourceName:    model.ServiceName,
								ResourceAddress: "1",
								ServiceName:     "chargepoint",
								ServiceAddress:  addr}
							fc.mqt.Publish(adrOperatingMode, msgOperatingMode)
						}
						// fc.mqt.Publish(adrOperatingMode, msgOperatingMode)
					}
				}
			}

			// send ChargeStart to that address
		case "cmd.charge.stop":
			// get address
			for _, chargers := range fc.states.Chargers.Data.ReceivingAccess {
				for _, chargepoint := range chargers.ChargePoint.AliasMap {
					if addr == chargepoint.Name {
						err := model.StopCharging(chargers.ChargePoint.ID, chargepoint.Connector, fc.configs.UserID, fc.configs.AccessToken)
						if err != nil {
							log.Error(err)
						} else {
							msgOperatingMode := fimpgo.NewMessage("evt.state.report", "defa", fimpgo.VTypeString, model.SetStatus(("ready_to_charge")), nil, nil, nil)
							msgOperatingMode.Source = "defa"
							adrOperatingMode := &fimpgo.Address{
								MsgType:         fimpgo.MsgTypeEvt,
								ResourceType:    fimpgo.ResourceTypeDevice,
								ResourceName:    model.ServiceName,
								ResourceAddress: "1",
								ServiceName:     "chargepoint",
								ServiceAddress:  addr}
							fc.mqt.Publish(adrOperatingMode, msgOperatingMode)
						}
						// fc.mqt.Publish(adrOperatingMode, msgOperatingMode)
					}
				}
			}

		case "cmd.state.get_report":
			// TODO
		case "cmd.smart_charge.set":
			// TODO
		}

	case model.ServiceName:
		adr := &fimpgo.Address{MsgType: fimpgo.MsgTypeEvt, ResourceType: fimpgo.ResourceTypeAdapter, ResourceName: model.ServiceName, ResourceAddress: "1"}
		switch newMsg.Payload.Type {
		case "cmd.auth.login":
			authReq := model.Login{}
			err := newMsg.Payload.GetObjectValue(&authReq)
			if err != nil {
				log.Error("Incorrect login message ")
				return
			}
			status := model.AuthStatus{
				Status:    model.AuthStateAuthenticated,
				ErrorText: "",
				ErrorCode: "",
			}
			if authReq.Username != "" && authReq.Password != "" {
				// TODO: This is an example . Add your logic here or remove
			} else {
				status.Status = "ERROR"
				status.ErrorText = "Empty username or password"
			}
			fc.appLifecycle.SetAuthState(model.AuthStateAuthenticated)
			msg := fimpgo.NewMessage("evt.auth.status_report", model.ServiceName, fimpgo.VTypeObject, status, nil, nil, newMsg.Payload)
			if err := fc.mqt.RespondToRequest(newMsg.Payload, msg); err != nil {
				// if response topic is not set , sending back to default application event topic
				fc.mqt.Publish(adr, msg)
			}

		case "cmd.auth.set_tokens":
			authReq := model.SetTokens{}
			err := newMsg.Payload.GetObjectValue(&authReq)
			if err != nil {
				log.Error("Incorrect login message ")
				return
			}
			status := model.AuthStatus{
				Status:    model.AuthStateAuthenticated,
				ErrorText: "",
				ErrorCode: "",
			}
			if authReq.AccessToken != "" && authReq.RefreshToken != "" {
				// TODO: This is an example . Add your logic here or remove
			} else {
				status.Status = "ERROR"
				status.ErrorText = "Empty username or password"
			}
			fc.appLifecycle.SetAuthState(model.AuthStateAuthenticated)
			msg := fimpgo.NewMessage("evt.auth.status_report", model.ServiceName, fimpgo.VTypeObject, status, nil, nil, newMsg.Payload)
			if err := fc.mqt.RespondToRequest(newMsg.Payload, msg); err != nil {
				// if response topic is not set , sending back to default application event topic
				fc.mqt.Publish(adr, msg)
			}

		case "cmd.app.get_manifest":
			mode, err := newMsg.Payload.GetStringValue()
			if err != nil {
				log.Error("Incorrect request format ")
				return
			}
			manifest := model.NewManifest()
			err = manifest.LoadFromFile(filepath.Join(fc.configs.GetDefaultDir(), "app-manifest.json"))
			if err != nil {
				log.Error("Failed to load manifest file .Error :", err.Error())
				return
			}
			if mode == "manifest_state" {
				manifest.AppState = *fc.appLifecycle.GetAllStates()
				manifest.ConfigState = fc.configs
			}
			if fc.configs.AccessToken != "" {
				if fc.states.IsConfigured() {
					var chargerSelect []interface{}
					manifest.Configs[0].ValT = "str_map"
					manifest.Configs[0].UI.Type = "list_checkbox"
					// for _, charger := range fc.states.Chargers.Data.ReceivingAccess {
					// 	// for _, charger := range data.ReceivingAccess {
					// 	ChargerID := fmt.Sprintf("%v", charger.ChargePoint.ID)
					// 	ChargerName := ChargerID
					// 	chargerSelect = append(chargerSelect, map[string]interface{}{"val": ChargerID, "label": map[string]interface{}{"en": ChargerName}})
					// 	// }
					// }
					for _, chargepoint := range fc.states.Chargers.Data.ReceivingAccess {
						for _, connector := range chargepoint.ChargePoint.AliasMap {
							log.Debug("Found new connector, name: ", connector.Name)
							ChargerName := fmt.Sprintf("%v", connector.Name)
							chargerSelect = append(chargerSelect, map[string]interface{}{"val": ChargerName, "label": map[string]interface{}{"en": ChargerName}})
						}
					}

					// for _, chargePoint := range fc.states.AliasMap.ReceivingAccess {
					// 	for _, connector := range chargePoint.ChargePoint.AliasMap {
					// 		log.Debug("Found new connector, name: ", connector.Name)
					// 		ChargerName := fmt.Sprintf("%v", connector.Name)
					// 		chargerSelect = append(chargerSelect, map[string]interface{}{"val": ChargerName, "label": map[string]interface{}{"en": ChargerName}})
					// 	}
					// }

					manifest.Configs[0].UI.Select = chargerSelect
				} else {
					manifest.Configs[0].ValT = "string"
					manifest.Configs[0].UI.Type = "input_readonly"
					manifest.Configs[0].UI.Select = nil
					var val model.Value
					val.Default = "Please refresh this page..."
					manifest.Configs[0].Val = val
				}
			} else {
				manifest.Configs[0].ValT = "string"
				manifest.Configs[0].UI.Type = "input_readonly"
				manifest.Configs[0].UI.Select = nil
				var val model.Value
				val.Default = "You need to login first"
				manifest.Configs[0].Val = val
			}
			msg := fimpgo.NewMessage("evt.app.manifest_report", model.ServiceName, fimpgo.VTypeObject, manifest, nil, nil, newMsg.Payload)
			msg.Source = "defa"
			if err := fc.mqt.RespondToRequest(newMsg.Payload, msg); err != nil {
				// if response topic is not set , sending back to default application event topic
				fc.mqt.Publish(adr, msg)
			}

		case "cmd.app.get_state":
			msg := fimpgo.NewMessage("evt.app.manifest_report", model.ServiceName, fimpgo.VTypeObject, fc.appLifecycle.GetAllStates(), nil, nil, newMsg.Payload)
			if err := fc.mqt.RespondToRequest(newMsg.Payload, msg); err != nil {
				// if response topic is not set , sending back to default application event topic
				fc.mqt.Publish(adr, msg)
			}

		case "cmd.config.get_extended_report":

			msg := fimpgo.NewMessage("evt.config.extended_report", model.ServiceName, fimpgo.VTypeObject, fc.configs, nil, nil, newMsg.Payload)
			if err := fc.mqt.RespondToRequest(newMsg.Payload, msg); err != nil {
				fc.mqt.Publish(adr, msg)
			}

		case "cmd.config.extended_set":
			conf := model.Configs{}
			err := newMsg.Payload.GetObjectValue(&conf)
			if err != nil {
				log.Error("Can't parse configuration object")
				return
			}

			if fc.configs.PhoneNr != conf.PhoneNr {
				log.Debug("Old phoneNr: ", fc.configs.PhoneNr)
				log.Debug("New phoneNr: ", conf.PhoneNr)
				fc.configs.PhoneNr = conf.PhoneNr
				fc.configs.UserID, err = fc.resetToken.ResetPassword(fc.configs.PhoneNr)
				if err != nil {
					log.Error(err)
				}
			}

			if fc.configs.SMSCode != conf.SMSCode {
				log.Debug("Old smsCode: ", fc.configs.SMSCode)
				log.Debug("New smsCode: ", conf.SMSCode)
				fc.configs.SMSCode = conf.SMSCode
				fc.configs.AccessToken, err = fc.loginToken.Login(fc.configs.UserID, fc.configs.SMSCode)
				if err != nil {
					log.Error("Error: ", err)
				}
				log.Debug("Getting chargers...")
				fc.states.Chargers.Data, err = model.GetChargers(fc.configs.UserID, fc.configs.AccessToken)
				if err != nil {
					log.Error("Error: ", err)
				}
				// fc.states.AliasMap, err = model.GetAliasMap(fc.configs.UserID, fc.configs.AccessToken)
				// if err != nil {
				// 	log.Error("Error: ", err)
				// }
				fc.states.ChargeSession, err = model.GetCharging(fc.configs.UserID, fc.configs.AccessToken)
				if err != nil {
					log.Error("Error: ", err)
				}
				fc.states.SaveToFile()
			}
			log.Debug("conf: ", conf)
			fc.configs.SelectedChargers = conf.SelectedChargers
			if len(fc.configs.SelectedChargers) != 0 {
				fc.appLifecycle.SetConfigState(model.ConfigStateConfigured)
				fc.appLifecycle.SetConnectionState(model.ConnStateConnected)
				fc.appLifecycle.SetAppState(model.AppStateRunning, nil)
				fc.appLifecycle.SetAuthState(model.AuthStateAuthenticated)
			} else {
				fc.appLifecycle.SetConfigState(model.ConfigStateNotConfigured)
				fc.appLifecycle.SetConnectionState(model.ConnStateDisconnected)
				fc.appLifecycle.SetAppState(model.AppStateNotConfigured, nil)
				fc.appLifecycle.SetAuthState(model.AuthStateNotAuthenticated)
			}
			if err = fc.configs.SaveToFile(); err != nil {
				log.Error(err)
			}
			if err = fc.states.SaveToFile(); err != nil {
				log.Error(err)
			}
			log.Debugf("App reconfigured . New parameters : %v", fc.configs)
			// TODO: This is an example . Add your logic here or remove
			configReport := model.ConfigReport{
				OpStatus: "ok",
				AppState: *fc.appLifecycle.GetAllStates(),
			}
			msg := fimpgo.NewMessage("evt.app.config_report", model.ServiceName, fimpgo.VTypeObject, configReport, nil, nil, newMsg.Payload)
			msg.Source = "defa"
			if err := fc.mqt.RespondToRequest(newMsg.Payload, msg); err != nil {
				fc.mqt.Publish(adr, msg)
			}

			for _, selectedCharger := range fc.configs.SelectedChargers {
				inclReport := ns.MakeInclusionReport(selectedCharger, selectedCharger)
				msg := fimpgo.NewMessage("evt.thing.inclusion_report", "defa", fimpgo.VTypeObject, inclReport, nil, nil, nil)
				msg.Source = "defa"
				adr := fimpgo.Address{MsgType: fimpgo.MsgTypeEvt, ResourceType: fimpgo.ResourceTypeAdapter, ResourceName: "defa", ResourceAddress: "1"}
				fc.mqt.Publish(&adr, msg)

				for _, chargers := range fc.states.Chargers.Data.ReceivingAccess {
					for _, chargepoint := range chargers.ChargePoint.AliasMap {
						if selectedCharger == chargepoint.Name {
							msgOperatingMode := fimpgo.NewMessage("evt.state.report", "defa", fimpgo.VTypeString, model.SetStatus((chargepoint.Status)), nil, nil, nil)
							msgOperatingMode.Source = "defa"
							adrOperatingMode := &fimpgo.Address{
								MsgType:         fimpgo.MsgTypeEvt,
								ResourceType:    fimpgo.ResourceTypeDevice,
								ResourceName:    model.ServiceName,
								ResourceAddress: "1",
								ServiceName:     "chargepoint",
								ServiceAddress:  selectedCharger}
							fc.mqt.Publish(adrOperatingMode, msgOperatingMode)

							for _, chargeSession := range *fc.states.ChargeSession {
								if chargeSession.ChargeSession.ChargePointID == chargers.ChargePoint.ID {
									chargeEnergy := chargepoint.MeterValue - chargeSession.ChargeSession.MeterStart
									msgPower := fimpgo.NewMessage("evt.current_session.report", "defa", fimpgo.VTypeFloat, chargeEnergy, nil, nil, nil)
									msgPower.Source = "defa"
									adrPower := &fimpgo.Address{
										MsgType:         fimpgo.MsgTypeEvt,
										ResourceType:    fimpgo.ResourceTypeDevice,
										ResourceName:    model.ServiceName,
										ResourceAddress: "1",
										ServiceName:     "chargepoint",
										ServiceAddress:  selectedCharger}
									fc.mqt.Publish(adrPower, msgPower)
								} else {
									chargeEnergy := 0
									msgPower := fimpgo.NewMessage("evt.current_session.report", "defa", fimpgo.VTypeFloat, chargeEnergy, nil, nil, nil)
									msgPower.Source = "defa"
									adrPower := &fimpgo.Address{
										MsgType:         fimpgo.MsgTypeEvt,
										ResourceType:    fimpgo.ResourceTypeDevice,
										ResourceName:    model.ServiceName,
										ResourceAddress: "1",
										ServiceName:     "chargepoint",
										ServiceAddress:  selectedCharger}
									fc.mqt.Publish(adrPower, msgPower)
								}
							}
						}
					}
				}
			}

		case "cmd.log.set_level":
			// Configure log level
			level, err := newMsg.Payload.GetStringValue()
			if err != nil {
				return
			}
			logLevel, err := log.ParseLevel(level)
			if err == nil {
				log.SetLevel(logLevel)
				fc.configs.LogLevel = level
				fc.configs.SaveToFile()
			}
			log.Info("Log level updated to = ", logLevel)

		case "cmd.system.reconnect":
			// This is optional operation.
			fc.appLifecycle.PublishEvent(model.EventConfigured, "from-fimp-router", nil)
			//val := map[string]string{"status":status,"error":errStr}
			val := model.ButtonActionResponse{
				Operation:       "cmd.system.reconnect",
				OperationStatus: "ok",
				Next:            "config",
				ErrorCode:       "",
				ErrorText:       "",
			}
			msg := fimpgo.NewMessage("evt.app.config_action_report", model.ServiceName, fimpgo.VTypeObject, val, nil, nil, newMsg.Payload)
			if err := fc.mqt.RespondToRequest(newMsg.Payload, msg); err != nil {
				fc.mqt.Publish(adr, msg)
			}

		case "cmd.app.factory_reset":
			val := model.ButtonActionResponse{
				Operation:       "cmd.app.factory_reset",
				OperationStatus: "ok",
				Next:            "config",
				ErrorCode:       "",
				ErrorText:       "",
			}
			fc.appLifecycle.SetConfigState(model.ConfigStateNotConfigured)
			fc.appLifecycle.SetAppState(model.AppStateNotConfigured, nil)
			fc.appLifecycle.SetAuthState(model.AuthStateNotAuthenticated)
			msg := fimpgo.NewMessage("evt.app.config_action_report", model.ServiceName, fimpgo.VTypeObject, val, nil, nil, newMsg.Payload)
			if err := fc.mqt.RespondToRequest(newMsg.Payload, msg); err != nil {
				fc.mqt.Publish(adr, msg)
			}

		case "cmd.network.get_all_nodes":
			// TODO: This is an example . Add your logic here or remove
		case "cmd.thing.get_inclusion_report":
			//nodeId , _ := newMsg.Payload.GetStringValue()
			// TODO: This is an example . Add your logic here or remove
		case "cmd.thing.inclusion":
			//flag , _ := newMsg.Payload.GetBoolValue()
			// TODO: This is an example . Add your logic here or remove
		case "cmd.thing.delete":
			val, err := newMsg.Payload.GetStrMapValue()
			if err != nil {
				log.Error("Wrong msg format")
				return
			}
			deviceID, ok := val["address"]
			if ok {
				val := map[string]interface{}{
					"address": deviceID,
				}
				adr := &fimpgo.Address{MsgType: fimpgo.MsgTypeEvt, ResourceType: fimpgo.ResourceTypeAdapter, ResourceName: "defa", ResourceAddress: "1"}
				msg := fimpgo.NewMessage("evt.thing.exclusion_report", "defa", fimpgo.VTypeObject, val, nil, nil, newMsg.Payload)
				fc.mqt.Publish(adr, msg)
				log.Info("Device with deviceID: ", deviceID, " has been removed from network.")
			} else {
				log.Error("Incorrect address")
			}
		case "cmd.app.uninstall":
			for _, chargers := range fc.states.Chargers.Data.ReceivingAccess {
				for _, chargepoint := range chargers.ChargePoint.AliasMap {
					log.Info("Exluding device: ", chargepoint.Name)
					val := map[string]interface{}{
						"address": chargepoint.Name,
					}
					msg := fimpgo.NewMessage("evt.thing.exclusion_report", "defa", fimpgo.VTypeObject, val, nil, nil, newMsg.Payload)
					msg.Source = "defa"
					adr := fimpgo.Address{MsgType: fimpgo.MsgTypeEvt, ResourceType: fimpgo.ResourceTypeAdapter, ResourceName: "defa", ResourceAddress: "1"}
					fc.mqt.Publish(&adr, msg)
				}
			}
		}
	}
}
