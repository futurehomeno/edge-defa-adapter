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
	var chargerObject model.ChargerObject
	loginToken := *&model.LoginToken{}

	log.Info("Starting ticker...")
	log.Debug("---------------------------------")
	ticker := time.NewTicker(time.Duration(pollTime) * time.Second)

	for range ticker.C {
		if states.Chargers.Data == nil {
			if configs.AccessToken == "" || configs.UserID == "" {
				log.Info("User needs to log in.")
			}
		} else {
			log.Debug("Getting chargers...")
			var err1 error
			var err2 error
			var err3 error

			states.Chargers.Data, err1 = model.GetChargers(configs.UserID, configs.AccessToken)
			if err1 != nil {
				log.Error("Error1: ", err1)
			}
			states.AliasMap, err2 = model.GetAliasMap(configs.UserID, configs.AccessToken)
			if err2 != nil {
				log.Error("Error2: ", err2)
			}
			states.ChargeSession, err3 = model.GetCharging(configs.UserID, configs.AccessToken)
			if err3 != nil {
				log.Error("Error3: ", err3)
			}
			if err1 != nil || err2 != nil || err3 != nil {
				log.Error("Something is wrong...: ", err1, err2, err3)

			}

			states.SaveToFile()

			log.Debug("---------------SELECTED CHARGER SECTION BEGINNING---------------")
			for _, selectedCharger := range configs.SelectedChargers {
				for p, charging := range *states.ChargeSession {
					// length := len(states.Chargers.Data.ReceivingAccess)
					// for p, realCharger := range states.Chargers.Data.ReceivingAccess {
					// if charging.ChargeSession.ChargePointID == realCharger.ChargePoint.ID {
					if selectedCharger == charging.ChargeSession.ChargePointID {
						for _, connector := range states.AliasMap.ReceivingAccess[p].ChargePoint.AliasMap {
							if len(states.LastChargerObjects) < p {
								states.LastChargerObjects = append(states.LastChargerObjects, chargerObject)
								states.LastChargerObjects[p-1].Name = selectedCharger
							}
							if len(states.LastChargerObjects) > 0 {
								// if !reflect.DeepEqual(states.LastChargerObjects[q].Status, connector.Status) {
								for s, charger := range states.LastChargerObjects {
									if charger.Name == selectedCharger {
										if states.LastChargerObjects[s].Status != connector.Status {
											log.Debug("Old status: ", states.LastChargerObjects[s].Status)
											log.Debug("New status: ", connector.Status)
											msgOperatingMode := fimpgo.NewMessage("evt.state.report", "defa", fimpgo.VTypeString, connector.Status, nil, nil, nil)
											msgOperatingMode.Source = "defa"
											adrOperatingMode := &fimpgo.Address{
												MsgType:         fimpgo.MsgTypeEvt,
												ResourceType:    fimpgo.ResourceTypeDevice,
												ResourceName:    model.ServiceName,
												ResourceAddress: "1",
												ServiceName:     "chargepoint",
												ServiceAddress:  selectedCharger}
											mqtt.Publish(adrOperatingMode, msgOperatingMode)

											states.LastChargerObjects[s].Status = connector.Status
										}
										if states.LastChargerObjects[s].Power != connector.Power {
											log.Debug("Old power: ", states.LastChargerObjects[s].Power)
											log.Debug("New power: ", connector.Power)
											msgPower := fimpgo.NewMessage("evt.power.report", "defa", fimpgo.VTypeString, connector.Power, nil, nil, nil)
											msgPower.Source = "defa"
											adrPower := &fimpgo.Address{
												MsgType:         fimpgo.MsgTypeEvt,
												ResourceType:    fimpgo.ResourceTypeDevice,
												ResourceName:    model.ServiceName,
												ResourceAddress: "1",
												ServiceName:     "chargepoint",
												ServiceAddress:  selectedCharger}
											mqtt.Publish(adrPower, msgPower)

											states.LastChargerObjects[s].Power = connector.Power
										}
									}
								}
							} else {
								states.LastChargerObjects = append(states.LastChargerObjects, chargerObject)
							}
						}
						// if i >= length {
						// 	continue
					}
					states.SaveToFile()
				}
			}
			log.Debug("---------------SELECTED CHARGER SECTION ENDED---------------")
		}
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
