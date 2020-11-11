package main

import (
	"flag"
	"fmt"
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

	states.Charger, err = model.GetChargers(configs.UserID, configs.AccessToken)
	states.AliasMap, err = model.GetAliasMap(configs.UserID, configs.AccessToken)
	states.ChargeSession, err = model.GetCharging(configs.UserID, configs.AccessToken)
	log.Debug("----------------------------")
	log.Debug("States.Charger: ", states.Charger.ReceivingAccess[0].ChargePoint)
	log.Debug("States.AliasMap: ", states.AliasMap)
	log.Debug("States.Charging: ", states.ChargeSession)
	states.SaveToFile()

	log.Debug("----------------------------")

	// pollString := configs.PollTimeSec
	// pollTime, err := strconv.Atoi(pollString)

	// log.Info("Starting ticker...")
	// log.Debug("---------------------------------")
	// ticker := time.NewTicker(time.Duration(pollTime) * time.Second)

	// var chargeObject model.ChargerObject

	// for range ticker.C {
	// 	if configs.AccessToken == "" && configs.UserID == "" {
	// 		log.Info("User needs to log in.")
	// 	} else {
	// 		if len(states.Chargers.Data) == 0 {
	// 			log.Debug("Getting cargers...")
	// 			log.Debug("-----------------------------------------")
	// 			states.Chargers, err = model.GetChargers(configs.UserID, configs.AccessToken)
	// 			if err != nil {
	// 				log.Error("Error: ", err)
	// 				if err.Error() == "401" {
	// 					configs.AccessToken = ""
	// 				}
	// 			} else {
	// 				log.Debug("Charger count: ", len(states.Chargers.Data))
	// 			}
	// 		}

	// 	}
	// }

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
