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
	// lastStatusObject := model.StatusObject{}
	// loginToken := *&model.LoginToken{}

	log.Info("Starting ticker...")
	log.Debug("---------------------------------")
	ticker := time.NewTicker(time.Duration(pollTime) * time.Second)

	for range ticker.C {
		counter := 0
		if states.Chargers.Data == nil && (configs.AccessToken == "" || configs.UserID == "") {
			log.Info("User needs to log in.")
		} else {
			states.Chargers.LastState = nil
			states.Chargers.LastEnergy = nil
			states.Chargers.LastTime = nil
			for _, selectedCharger := range configs.SelectedChargers {
				for _, chargers := range states.Chargers.Data.ReceivingAccess {
					for _, chargepoint := range chargers.ChargePoint.AliasMap {
						if selectedCharger == chargepoint.Name {
							// if chargepoint.LastStatus != model.SetStatus(chargepoint.Status)
							if hasInitialized {
								states.Chargers.LastState = append(states.Chargers.LastState, model.SetStatus(chargepoint.Status))
								for _, chargeSession := range *states.ChargeSession {
									if chargeSession.ChargeSession.ChargePointID == chargers.ChargePoint.ID {
										states.Chargers.LastEnergy = append(states.Chargers.LastEnergy, chargepoint.MeterValue-chargeSession.ChargeSession.MeterStart)
									} else {
										states.Chargers.LastEnergy = append(states.Chargers.LastEnergy, 0)
									}
									states.Chargers.LastTime = append(states.Chargers.LastTime, chargepoint.StatusUpdated)
								}
							} else {
								states.Chargers.LastState = append(states.Chargers.LastState, "initialized")
								for range *states.ChargeSession {
									states.Chargers.LastEnergy = append(states.Chargers.LastEnergy, -1)
								}
							}
						}
					}
				}
			}

			log.Debug("Getting chargers...")
			var err1 error
			// var err2 error
			var err3 error

			states.Chargers.Data, err1 = model.GetChargers(configs.UserID, configs.AccessToken)
			if err1 != nil {
				log.Error("Error1: ", err1)
			}
			// states.AliasMap, err2 = model.GetAliasMap(configs.UserID, configs.AccessToken)
			// if err2 != nil {
			// 	log.Error("Error2: ", err2)
			// }
			states.ChargeSession, err3 = model.GetCharging(configs.UserID, configs.AccessToken)
			if err3 != nil {
				log.Error("Error3: ", err3)
			}
			if err1 != nil || err3 != nil { // || err 2
				log.Error("Something is wrong...: ", err1, err3) // , err2
			}

			states.SaveToFile()

			log.Debug("---------------SELECTED CHARGER SECTION BEGINNING---------------")
			//
			hours := time.Now().UnixNano() / 3600000

			log.Debug(hours / 1000000)

			//

			for _, selectedCharger := range configs.SelectedChargers {
				for _, chargers := range states.Chargers.Data.ReceivingAccess {
					for _, chargepoint := range chargers.ChargePoint.AliasMap {
						if selectedCharger == chargepoint.Name {
							if len(states.Chargers.LastEnergy) > 0 {
								log.Debug("last state 0: ", states.Chargers.LastState[counter])
								log.Debug("defa state 0: ", chargepoint.Status)
								log.Debug("new state 0: ", model.SetStatus(chargepoint.Status))
								if states.Chargers.LastState[counter] != model.SetStatus(chargepoint.Status) {
									log.Debug("last state 1: ", states.Chargers.LastState[counter])
									log.Debug("new state 1: ", model.SetStatus(chargepoint.Status))
									msgOperatingMode := fimpgo.NewMessage("evt.state.report", "defa", fimpgo.VTypeString, model.SetStatus((chargepoint.Status)), nil, nil, nil)
									msgOperatingMode.Source = "defa"
									adrOperatingMode := &fimpgo.Address{
										MsgType:         fimpgo.MsgTypeEvt,
										ResourceType:    fimpgo.ResourceTypeDevice,
										ResourceName:    model.ServiceName,
										ResourceAddress: "1",
										ServiceName:     "chargepoint",
										ServiceAddress:  selectedCharger}
									mqtt.Publish(adrOperatingMode, msgOperatingMode)
								}
							}
							for _, chargeSession := range *states.ChargeSession {
								if chargeSession.ChargeSession.ChargePointID == chargers.ChargePoint.ID {
									chargeEnergy := chargepoint.MeterValue - chargeSession.ChargeSession.MeterStart
									if len(states.Chargers.LastEnergy) > 0 {
										if states.Chargers.LastEnergy[counter] != chargeEnergy {
											log.Debug("last energy 1: ", states.Chargers.LastEnergy[counter])
											log.Debug("new energy 1: ", chargeEnergy)
											msgPower := fimpgo.NewMessage("evt.current_session.report", "defa", fimpgo.VTypeFloat, chargeEnergy, nil, nil, nil)
											msgPower.Source = "defa"
											adrPower := &fimpgo.Address{
												MsgType:         fimpgo.MsgTypeEvt,
												ResourceType:    fimpgo.ResourceTypeDevice,
												ResourceName:    model.ServiceName,
												ResourceAddress: "1",
												ServiceName:     "chargepoint",
												ServiceAddress:  selectedCharger}
											mqtt.Publish(adrPower, msgPower)

											newMeterValue := (states.Chargers.LastEnergy[counter] - chargeEnergy) / float64((chargepoint.StatusUpdated-states.Chargers.LastTime[counter])/1000000)
											log.Debug("New meter value: ", newMeterValue)
										}
									}
								}
							}
							counter++
						}
					}
				}
			}
			log.Debug("---------------SELECTED CHARGER SECTION ENDED---------------")
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
