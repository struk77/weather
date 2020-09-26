package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"time"

	owm "github.com/briandowns/openweathermap"
)

type sendMessageReqBody struct {
	ChatID int64  `json:"chat_id"`
	Text   string `json:"text"`
}

type owmStruct struct {
	APIkey   string
	Location string
	Units    string
	Language string
}

type tgStruct struct {
	Bottoken string
	Chatid   int64
}

type dataJSON struct {
	OWM owmStruct
	TG  tgStruct
}

func sendMessage(url, message string, chatId int64) error {
	log.SetOutput(os.Stdout)
	m := &sendMessageReqBody{
		Text:   message,
		ChatID: chatId,
	}
	jsonStr, err := json.Marshal(m)
	if err != nil {
		return err
	}
	req, err := http.Post(url, "application/json", bytes.NewBuffer(jsonStr))
	if err != nil {
		return err
	}
	if req.StatusCode != http.StatusOK {
		return errors.New("unexpected status" + req.Status)
	}
	return err
}

func getCurrent(owmData owmStruct) (*owm.CurrentWeatherData, error) {
	w, err := owm.NewCurrent(owmData.Units, owmData.Language, owmData.APIkey)
	if err != nil {
		return nil, err
	}
	err = w.CurrentByName(owmData.Location)
	if err != nil {
		return nil, err
	}
	return w, nil
}

func getForecast5(owmData owmStruct) (*owm.Forecast5WeatherData, error) {
	w, err := owm.NewForecast("5", owmData.Units, owmData.Language, owmData.APIkey)
	if err != nil {
		return nil, err
	}
	err = w.DailyByName(owmData.Location, 5)
	if err != nil {
		return nil, err
	}
	forecast := w.ForecastWeatherJson.(*owm.Forecast5WeatherData)
	return forecast, err
}

func main() {
	log.SetOutput(os.Stdout)
	dataFile := os.Args[1]
	jsonFile, err := os.Open(dataFile)
	// if we os.Open returns an error then handle it
	if err != nil {
		log.Fatalln(err)
	}
	log.Println("Successfully opened data json")
	// defer the closing of our jsonFile so that we can parse it later on
	defer jsonFile.Close()

	byteValue, _ := ioutil.ReadAll(jsonFile)
	var data dataJSON

	err = json.Unmarshal(byteValue, &data)
	if err != nil {
		log.Fatalln(err)
	}

	current, err := getCurrent(data.OWM)
	if err != nil {
		log.Fatalln(err)
	}
	var langString [3]string
	switch data.OWM.Language {
	case "RU":
		langString = [3]string{"Сейчас в г.", "Температура воздуха", "Прогноз"}
	default:
		langString = [3]string{"Now in ", "Air temperature", "Forecast"}
	}
	message := fmt.Sprintf("%s%s %s. %s %.0f°C\n", langString[0], current.Name, current.Weather[0].Description, langString[1], current.Main.Temp)

	w, err := getForecast5(data.OWM)
	if err != nil {
		log.Fatalln(err)
	}
	message = message + langString[2] + ":\n"
	for _, v := range w.List {
		tm := fmt.Sprintf("%d:00", time.Unix(int64(v.Dt), 0).Hour())
		m := fmt.Sprintf("%-5s %.0f°C, %s\n", tm, v.Main.Temp, v.Weather[0].Description)
		message = message + m
	}

	//log.Println(message)
	err = sendMessage("https://api.telegram.org/bot"+data.TG.Bottoken+"/sendMessage", message, data.TG.Chatid)
	if err != nil {
		fmt.Println(err)
	}
	os.Exit(0)
}
