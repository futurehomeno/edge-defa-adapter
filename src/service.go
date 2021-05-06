package main

import (
	"flag"
	"fmt"
	"strconv"
	"time"

	"github.com/futurehomeno/fimpgo"
	"github.com/futurehomeno/fimpgo/discovery"
	log "github.com/sirupsen/logrus"
	"github.com/thingsplex/defa/model"
	"github.com/thingsplex/defa/router"
	"github.com/thingsplex/defa/utils"
)

func main() {
	var workDir string
	flag.StringVar(&workDir, "c", "", "Work dir")
	flag.Parse()
	if workDir == "" {
		workDir = "./"
	} else {
		fmt.Println("Work dir ", workDir)
	}
	appLifecycle := model.NewAppLifecycle()
	configs := model.NewConfigs(workDir)
	states := model.NewStates(workDir)

	err := configs.LoadFromFile()
	if err != nil {
		fmt.Print(err)
		panic("Can't load config file.")
	}
	err = states.LoadFromFile()
	if err != nil {
		fmt.Print(err)
		panic("Can't load state file")
	}

	utils.SetupLog(configs.LogFile, configs.LogLevel, configs.LogFormat)
	log.Info("--------------Starting defa----------------")
	log.Info("Work directory : ", configs.WorkDir)
	appLifecycle.PublishEvent(model.EventConfiguring, "main", nil)

	mqtt := fimpgo.NewMqttTransport(configs.MqttServerURI, configs.MqttClientIdPrefix, configs.MqttUsername, configs.MqttPassword, true, 1, 1)
	err = mqtt.Start()
	responder := discovery.NewServiceDiscoveryResponder(mqtt)
	responder.RegisterResource(model.GetDiscoveryResource())
	responder.Start()

	fimpRouter := router.NewFromFimpRouter(mqtt, appLifecycle, configs, states)
	fimpRouter.Start()

	appLifecycle.SetConnectionState(model.ConnStateDisconnected)
	if configs.IsConfigured() && err == nil {
		appLifecycle.SetConfigState(model.ConfigStateConfigured)
		appLifecycle.SetAppState(model.AppStateRunning, nil)
		appLifecycle.SetConnectionState(model.ConnStateConnected)
	} else {
		appLifecycle.SetConfigState(model.ConfigStateNotConfigured)
		appLifecycle.SetAppState(model.AppStateNotConfigured, nil)
		appLifecycle.SetConnectionState(model.ConnStateDisconnected)
	}

	if configs.UserID != "" && configs.AccessToken != "" {
		appLifecycle.SetAuthState(model.AuthStateAuthenticated)
	} else {
		appLifecycle.SetAuthState(model.AuthStateNotAuthenticated)
	}

	// states.Charger, err = model.GetChargers(configs.UserID, configs.AccessToken)
	// states.AliasMap, err = model.GetAliasMap(configs.UserID, configs.AccessToken)
	// states.ChargeSession, err = model.GetCharging(configs.UserID, configs.AccessToken)
	// log.Debug("----------------------------")
	// log.Debug("States.Charger: ", states.Charger.ReceivingAccess[0].ChargePoint)
	// log.Debug("States.AliasMap: ", states.AliasMap)
	// log.Debug("States.Charging: ", states.ChargeSession)
	// states.SaveToFile()

	log.Debug("----------------------------")

	pollString := configs.PollTimeSec
	pollTime, err := strconv.Atoi(pollString)
	hasInitialized := false
	charging := true
	chargingFalseReported := false
	// lastStatusObject := model.StatusObject{}
	// loginToken := *&model.LoginToken{}

	log.Info("Starting ticker...")
	log.Debug("---------------------------------")
	ticker := time.NewTicker(time.Duration(pollTime) * time.Second)

	for range ticker.C {
		log.Debug("")
		counter := 0
		if configs.AccessToken == "" || configs.UserID == "" {
			log.Info("User needs to log in.")
		} else {
			states.Chargers.LastState = nil
			states.Chargers.LastEnergy = nil
			// states.Chargers.LastTime = nil
			for _, selectedCharger := range configs.SelectedChargers {
				for _, chargers := range states.Chargers.Data.ReceivingAccess {
					for _, chargepoint := range chargers.ChargePoint.AliasMap {
						if selectedCharger == chargepoint.Name {
							// if chargepoint.LastStatus != model.SetStatus(chargepoint.Status)
							if hasInitialized {
								states.Chargers.LastState = append(states.Chargers.LastState, model.SetStatus(chargepoint.Status))
								// if states.ChargeSession != nil {
								for _, chargeSession := range *states.ChargeSession {
									if chargeSession.ChargeSession.ChargePointID == chargers.ChargePoint.ID {
										states.Chargers.LastEnergy = append(states.Chargers.LastEnergy, chargepoint.MeterValue-chargeSession.ChargeSession.MeterStart)
									} else {
										states.Chargers.LastEnergy = append(states.Chargers.LastEnergy, 0)
									}
								}
								//}
							} else {
								states.Chargers.LastState = append(states.Chargers.LastState, "initialized")
								if states.ChargeSession != nil {
									for range *states.ChargeSession {
										states.Chargers.LastEnergy = append(states.Chargers.LastEnergy, -1)
									}
								}
								states.Chargers.LastTime = states.Chargers.Data.Timestamp
							}
						}
					}
				}
			}

			var err1 error
			var err2 error
			var newMeterValue float64

			states.Chargers.Data, err1 = model.GetChargers(configs.UserID, configs.AccessToken)
			if err1 != nil {
				log.Error("Error1: ", err1)
			}
			states.ChargeSession, err2 = model.GetCharging(configs.UserID, configs.AccessToken)
			if err2 != nil {
				log.Info("Error due to no active charging session.")
				charging = false
			} else {
				charging = true
			}

			states.SaveToFile()

			for _, selectedCharger := range configs.SelectedChargers {
				for _, chargers := range states.Chargers.Data.ReceivingAccess {
					for _, chargepoint := range chargers.ChargePoint.AliasMap {
						if selectedCharger == chargepoint.Name {
							if len(states.Chargers.LastState) > 0 {
								// if states.Chargers.LastState[counter] != model.SetStatus(chargepoint.Status) {
								val := model.SetStatus(chargepoint.Status)
								if val == "charging" && charging == false { // This should of course not ever happen, but it has.
									val = "error"
								}
								// log.Info("last state: ", states.Chargers.LastState[counter])
								log.Info("new state: ", val)
								msgOperatingMode := fimpgo.NewMessage("evt.state.report", "defa", fimpgo.VTypeString, val, nil, nil, nil)
								msgOperatingMode.Source = "defa"
								adrOperatingMode := &fimpgo.Address{
									MsgType:         fimpgo.MsgTypeEvt,
									ResourceType:    fimpgo.ResourceTypeDevice,
									ResourceName:    model.ServiceName,
									ResourceAddress: "1",
									ServiceName:     "chargepoint",
									ServiceAddress:  selectedCharger}
								mqtt.Publish(adrOperatingMode, msgOperatingMode)
								// }
								var cableConnected bool
								if val == "charging" || val == "ready_to_charge" || val == "finished" {
									cableConnected = true
								} else {
									cableConnected = false
								}
								msgCableLock := fimpgo.NewMessage("evt.cable_lock.report", "defa", fimpgo.VTypeBool, cableConnected, nil, nil, nil)
								msgCableLock.Source = "defa"
								adrCableLock := &fimpgo.Address{
									MsgType:         fimpgo.MsgTypeEvt,
									ResourceType:    fimpgo.ResourceTypeDevice,
									ResourceName:    model.ServiceName,
									ResourceAddress: "1",
									ServiceName:     "chargepoint",
									ServiceAddress:  selectedCharger}
								mqtt.Publish(adrCableLock, msgCableLock)
							}
							if charging == true {
								chargingFalseReported = false
								for _, chargeSession := range *states.ChargeSession {
									// If charging == false, *states.ChargeSession is nil
									if chargeSession.ChargeSession.ChargePointID == chargers.ChargePoint.ID {
										chargeEnergy := chargepoint.MeterValue - chargeSession.ChargeSession.MeterStart
										if len(states.Chargers.LastEnergy) > 0 {
											if states.Chargers.LastEnergy[counter] != chargeEnergy {

												log.Info("new energy 1: ", chargeEnergy)

												// if states.Chargers.LastTime != 0 && states.Chargers.LastEnergy[counter] != -1 {
												if states.Chargers.LastEnergy[counter] != -1 {

													deltaCharge := chargeEnergy - states.Chargers.LastEnergy[counter]
													deltaTime := float64((states.Chargers.Data.Timestamp - states.Chargers.LastTime))
													// log.Debug("deltaCharge: ", deltaCharge)
													// log.Debug("deltaTime: ", deltaTime)
													newMeterValue = (deltaCharge / deltaTime) * 3600000000
													log.Info("New meter value: ", newMeterValue)
													// log.Info("")
													states.Chargers.LastTime = states.Chargers.Data.Timestamp
												} else {
													deltaCharge := chargeEnergy
													deltaTime := float64(states.Chargers.Data.Timestamp - chargeSession.ChargeSession.StartTime)
													newMeterValue = (deltaCharge / deltaTime) * 3600000000
													log.Info("New (first) meter value: ", newMeterValue)
													// log.Info("")
												}
												// }

												msgEnergy := fimpgo.NewMessage("evt.current_session.report", "defa", fimpgo.VTypeFloat, chargeEnergy, nil, nil, nil)
												msgEnergy.Source = "defa"
												adrEnergy := &fimpgo.Address{
													MsgType:         fimpgo.MsgTypeEvt,
													ResourceType:    fimpgo.ResourceTypeDevice,
													ResourceName:    model.ServiceName,
													ResourceAddress: "1",
													ServiceName:     "chargepoint",
													ServiceAddress:  selectedCharger}
												mqtt.Publish(adrEnergy, msgEnergy)

												props := fimpgo.Props{}
												props["unit"] = "W"
												if newMeterValue < 0 {
													newMeterValue = 0
												} else if newMeterValue > chargepoint.Power*1000 {
													newMeterValue = chargepoint.Power * 1000
												}
												msgPower := fimpgo.NewMessage("evt.meter.report", "defa", fimpgo.VTypeFloat, newMeterValue, props, nil, nil)
												msgPower.Source = "defa"
												adrPower := &fimpgo.Address{
													MsgType:         fimpgo.MsgTypeEvt,
													ResourceType:    fimpgo.ResourceTypeDevice,
													ResourceName:    model.ServiceName,
													ResourceAddress: "1",
													ServiceName:     "meter_elec",
													ServiceAddress:  selectedCharger}
												mqtt.Publish(adrPower, msgPower)
											}
										}
									}
								}
							} else if charging == false && chargingFalseReported == false {
								chargeEnergy := 0
								newMeterValue = 0
								msgEnergy := fimpgo.NewMessage("evt.current_session.report", "defa", fimpgo.VTypeFloat, chargeEnergy, nil, nil, nil)
								msgEnergy.Source = "defa"
								adrEnergy := &fimpgo.Address{
									MsgType:         fimpgo.MsgTypeEvt,
									ResourceType:    fimpgo.ResourceTypeDevice,
									ResourceName:    model.ServiceName,
									ResourceAddress: "1",
									ServiceName:     "chargepoint",
									ServiceAddress:  selectedCharger}
								mqtt.Publish(adrEnergy, msgEnergy)

								props := fimpgo.Props{}
								props["unit"] = "W"
								if newMeterValue < 0 {
									newMeterValue = 0
								}
								msgPower := fimpgo.NewMessage("evt.meter.report", "defa", fimpgo.VTypeFloat, newMeterValue, props, nil, nil)
								msgPower.Source = "defa"
								adrPower := &fimpgo.Address{
									MsgType:         fimpgo.MsgTypeEvt,
									ResourceType:    fimpgo.ResourceTypeDevice,
									ResourceName:    model.ServiceName,
									ResourceAddress: "1",
									ServiceName:     "meter_elec",
									ServiceAddress:  selectedCharger}
								mqtt.Publish(adrPower, msgPower)

								chargingFalseReported = true
							}
							counter++
						}
					}
				}
			}
		}
		hasInitialized = true
	}

	for {
		appLifecycle.WaitForState("main", model.AppStateRunning)
		// Configure custom resources here
		//if err := conFimpRouter.Start(); err !=nil {
		//	appLifecycle.PublishEvent(model.EventConfigError,"main",nil)
		//}else {
		//	appLifecycle.WaitForState(model.StateConfiguring,"main")
		//}
		//TODO: Add logic here
		appLifecycle.WaitForState(model.AppStateNotConfigured, "main")
	}

	mqtt.Stop()
	time.Sleep(5 * time.Second)
}
