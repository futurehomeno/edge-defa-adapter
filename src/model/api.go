package model

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"

	log "github.com/sirupsen/logrus"
)

const (
	BASE_URL   = "https://staging.cloudcharge.se/services/user/"
	charging   = BASE_URL + "charging"
	start      = charging + "/start"
	stop       = charging + "/stop"
	mychargers = BASE_URL + "mychargers"
	pwreset    = BASE_URL + "password/reset"
	login      = BASE_URL + "login"
	devToken   = "Fx3Vzr5nUUVHWwhf"
)

type (
	aliasMapValue struct {
		Connector     int
		Power         float64
		MeterValue    float64 `json:"meterValue"`
		ConnectorType string  `json:"connectorType"`
		Status        string
		ErrorCode     string `json:"errorCode"`
		StatusUpdate  int64  `json:"statusUpdated"`
		IsFavorite    bool   `json:"isFavorite"`
	}
	Charger2 struct {
		TimeStamp       int64 `json:"timestamp"`
		ReceivingAccess []struct {
			ChargePoint struct {
				AliasMap map[string]aliasMapValue `json:"aliasMap"`
			} `json:"chargePoint"`
		} `json:"receivingAccess"`
	}
)

type Chargers struct {
	Data []Charger
}

type ResetToken struct {
	Token string `json:"token"`
}

type LoginToken struct {
	Token string `json:"token"`
}

type Charger struct {
	Timestamp       int64 `json:"timestamp"`
	ReceivingAccess []struct {
		ChargePoint struct {
			ID                string `json:"id"`
			Group             string `json:"group"`
			LastHB            int64  `json:"lastHB"`
			HbTimeout         bool   `json:"hbTimeout"`
			ConfigurationKeys struct {
				ChargingScheduleAllowedChargingRateUnit string `json:"ChargingScheduleAllowedChargingRateUnit"`
				SupportedFeatureProfiles                string `json:"SupportedFeatureProfiles"`
			} `json:"configurationKeys"`
			LastSuccesfulChargingRate float64  `json:"lastSuccesfulChargingRate"`
			AliasMap                  struct{} `json:"aliasMap"` // This will be empty, as it contains dynamic keys.
			IsFavorite                bool     `json:"isFavorite"`
			IsReservedForYou          bool     `json:"isReservedForYou"`
			LoadBalancingActive       bool     `json:"loadBalancingActive"`
		} `json:"chargePoint"`
		Token struct {
			Status        string      `json:"status"`
			AccessID      string      `json:"accessId"`
			ChargePointID string      `json:"chargePointId"`
			ConnectorID   interface{} `json:"connectorId"`
			EndTime       interface{} `json:"endTime"`
			StartTime     int64       `json:"startTime"`
			Role          string      `json:"role"`
			MetaString    interface{} `json:"metaString"`
		} `json:"token"`
	} `json:"receivingAccess"`
	GivingAccess []interface{} `json:"givingAccess"`
}

type Charging []struct {
	ChargeSession struct {
		StartTime   int64       `json:"startTime"`
		EndTime     interface{} `json:"endTime"`
		MeterStart  float64     `json:"meterStart"`
		MeterEnd    float64     `json:"meterEnd"`
		PaymentInfo struct {
			Cost     float64     `json:"cost"`
			Vat      float64     `json:"vat"`
			Currency interface{} `json:"currency"`
		} `json:"paymentInfo"`
		TransactionID  int         `json:"transactionId"`
		UserID         string      `json:"userId"`
		ChargePointID  string      `json:"chargePointId"`
		ConnectorID    int         `json:"connectorId"`
		Classification string      `json:"classification"`
		Note           interface{} `json:"note"`
		CarInfo        interface{} `json:"carInfo"`
		CreditCardInfo interface{} `json:"creditCardInfo"`
		IsMyCharger    bool        `json:"isMyCharger"`
		Metadata       struct {
			Cpid                string `json:"cpid"`
			LocationDescription string `json:"locationDescription"`
			Location            string `json:"location"`
		} `json:"metadata"`
	} `json:"chargeSession"`
}

type AliasMap []struct {
	NumXXX struct {
		Connector     int         `json:"connector"`
		Power         float64     `json:"power"`
		MeterValue    float64     `json:"meterValue"`
		Info          interface{} `json:"info"`
		ConnectorType string      `json:"connectorType"`
		Tariff        interface{} `json:"tariff"`
		Status        string      `json:"status"`
		ErrorCode     string      `json:"errorCode"`
		ErrorInfo     interface{} `json:"errorInfo"`
		StatusUpdated int64       `json:"statusUpdated"`
		CustomerID    interface{} `json:"customerId"`
		IsFavorite    bool        `json:"isFavorite"`
	}
}

func processHTTPResponse(resp *http.Response, err error, holder interface{}) error {
	if err != nil {
		log.Error(fmt.Errorf("API does not respond"))
		return err
	}
	defer resp.Body.Close()
	// check http return code
	if resp.StatusCode != 200 {
		//bytes, _ := ioutil.ReadAll(resp.Body)
		log.Error("Bad HTTP return code ", resp.StatusCode)
		return fmt.Errorf("Bad HTTP return code %d", resp.StatusCode)
	}

	if err = json.NewDecoder(resp.Body).Decode(holder); err != nil {
		return err
	}
	return nil
}

func (rt *ResetToken) ResetPassword(phonenr string) (string, error) {
	log.Debug("Resetting password")

	type Payload struct {
		Phonenr string `json:"token"`
	}
	data := Payload{
		Phonenr: phonenr,
	}
	payloadBytes, err := json.Marshal(data)
	if err != nil {
		log.Debug("issue with payloadBytes")
	}
	body := bytes.NewReader(payloadBytes)

	req, err := http.NewRequest("POST", pwreset, body)

	if err != nil {
		log.Error(fmt.Errorf("Can't post pwreset request, error: %v", err))
		return "", err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("devToken", devToken)
	resp, err := http.DefaultClient.Do(req)
	processHTTPResponse(resp, err, rt)
	log.Debug("pwreset token: ", rt.Token)
	return rt.Token, nil
}

func (lt *LoginToken) Login(resetToken string, smspw string) (string, error) {
	log.Debug("Logging in")

	type Payload struct {
		UserId   string `json:"userId"`
		Password string `"json:"password"`
	}
	data := Payload{
		UserId:   resetToken,
		Password: smspw,
	}
	payloadBytes, err := json.Marshal(data)
	if err != nil {
		log.Debug("issue with payloadBytes")
	}
	body := bytes.NewReader(payloadBytes)

	req, err := http.NewRequest("POST", login, body)
	log.Debug("req: ", req)

	if err != nil {
		log.Error(fmt.Errorf("Can't post login request, error: %v", err))
		return "", err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("devToken", devToken)

	resp, err := http.DefaultClient.Do(req)

	processHTTPResponse(resp, err, lt)
	log.Debug("login token: ", lt.Token)
	return lt.Token, nil
}

func GetChargers(userId string, accessToken string) (*Charger, error) {
	charger := new(Charger)
	err := get(userId, accessToken, mychargers, charger) // get charger
	if err != nil {
		return charger, err
	}
	// err = get(userId, accessToken, mychargers, charger, true) // get aliasMap

	return charger, err
}

func GetAliasMap(userId string, accessToken string) (*Charger2, error) {
	// aliasMap := new(AliasMap)
	var c *Charger2
	// var am AliasMap
	log.Debug("Getting from ", mychargers)

	req, err := http.NewRequest("GET", mychargers, nil)

	if err != nil {
		log.Error(fmt.Errorf("Can't GET from ", mychargers))
	}

	req.Header.Set("Accept", "*/*")
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-User", userId)
	req.Header.Set("X-Authorization", accessToken)
	req.Header.Set("devToken", devToken)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	if err := json.NewDecoder(resp.Body).Decode(&c); err != nil {
		log.Error(err)
	}
	log.Info(c)

	return c, err
}

func GetCharging(userId string, accessToken string) (*Charging, error) {
	chargeSession := new(Charging)
	// chargePoint := new(ChargePoint)
	err := get(userId, accessToken, charging, chargeSession)

	return chargeSession, err
}

func get(userId string, accessToken string, url string, target interface{}) error {
	log.Debug("Getting from ", url)

	req, err := http.NewRequest("GET", url, nil)

	if err != nil {
		log.Error(fmt.Errorf("Can't GET from ", url))
	}

	req.Header.Set("Accept", "*/*")
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-User", userId)
	req.Header.Set("X-Authorization", accessToken)
	req.Header.Set("devToken", devToken)
	resp, err := http.DefaultClient.Do(req)
	processHTTPResponse(resp, err, target)
	return err
}
