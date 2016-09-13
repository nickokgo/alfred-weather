package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

var fioIconNames = map[string]string{
	"clear-day":           "clear",
	"clear-night":         "nt_clear",
	"partly-cloudy-day":   "partlycloudy",
	"partly-cloudy-night": "nt_partlycloudy",
	"wind":                "hazy",
}

const fioAPI = "https://api.forecast.io/forecast"

// ForecastIO is a Forecast.io service handle
type ForecastIO struct {
	apiKey string
}

type fioConditions struct {
	Temperature         float64 `json:"temperature"`
	Icon                string  `json:"icon"`
	Humidity            float64 `json:"humidity"`
	Summary             string  `json:"summary"`
	ApparentTemperature float64 `json:"apparentTemperature"`
	PrecipProbability   float64 `json:"precipProbability"`
	Time                int64   `json:"time"`
}

type fioWeather struct {
	Daily struct {
		Icon string `json:"icon"`
		Data []struct {
			PrecipType        string  `json:"precipType"`
			TempMin           float64 `json:"temperatureMin"`
			TempMax           float64 `json:"temperatureMax"`
			Summary           string  `json:"summary"`
			SunsetTime        int64   `json:"sunsetTime"`
			SunriseTime       int64   `json:"sunriseTime"`
			PrecipProbability float64 `json:"precipProbability"`
			Icon              string  `json:"icon"`
			Time              int64   `json:"time"`
		} `json:"data"`
		Summary string `json:"summary"`
	} `json:"daily"`
	Hourly struct {
		Icon    string `json:"icon"`
		Summary string `json:"summary"`
		Data    []struct {
			ApparentTemp      float64 `json:"apparentTemperature"`
			Humidity          float64 `json:"humidity"`
			Icon              string  `json:"icon"`
			PrecipProbability float64 `json:"precipProbability"`
			Summary           string  `json:"summary"`
			Temp              float64 `json:"temperature"`
			Time              int64   `json:"time"`
		} `json:"data"`
	} `json:"hourly"`
	Currently fioConditions `json:"currently"`
	Flags     struct {
		Units string `json:"units"`
	} `json:"flags"`
	Alerts []struct {
		Title   string `json:"title"`
		Expires int64  `json:"exipres"`
		URI     string `json:"uri"`
	} `json:"alerts"`
}

// NewForecastIO returns a new ForecastIO handle
func NewForecastIO(apiKey string) ForecastIO {
	return ForecastIO{apiKey: apiKey}
}

// Forecast returns the forecast for a given location
func (f *ForecastIO) Forecast(l Location) (weather Weather, err error) {
	dlog.Printf("getting forecast for %#v", l)

	url := fmt.Sprintf("%s/%s/%f,%f", fioAPI, f.apiKey, l.Latitude, l.Longitude)
	dlog.Printf("getting URL %s", url)

	var request *http.Request
	if request, err = http.NewRequest("GET", url, nil); err != nil {
		return
	}

	var resp *http.Response
	if resp, err = client.Do(request); err != nil {
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return weather, fmt.Errorf(resp.Status)
	}

	var w fioWeather
	if err = json.NewDecoder(resp.Body).Decode(&w); err != nil {
		return
	}

	for _, a := range w.Alerts {
		alert := alert{
			Description: a.Title,
			Expires:     time.Unix(a.Expires, 0),
			URI:         a.URI,
		}
		weather.Alerts = append(weather.Alerts, alert)
	}

	var units units
	if w.Flags.Units == "us" {
		units = unitsUS
	} else {
		units = unitsMetric
	}

	weather.Current.Summary = w.Currently.Summary
	weather.Current.Icon = fromFioIconName(w.Currently.Icon)
	weather.Current.Humidity = w.Currently.Humidity * 100
	weather.Current.Temp = Temperature{
		Value: w.Currently.Temperature,
		Units: units,
	}

	for _, d := range w.Daily.Data {
		f := dailyForecast{
			Date:    time.Unix(d.Time, 0),
			Icon:    fromFioIconName(d.Icon),
			Precip:  int(d.PrecipProbability * 100),
			Summary: d.Summary,
			HighTemp: Temperature{
				Value: d.TempMax,
				Units: units,
			},
			LowTemp: Temperature{
				Value: d.TempMin,
				Units: units,
			},
			Sunrise: time.Unix(d.SunriseTime, 0),
			Sunset:  time.Unix(d.SunsetTime, 0),
		}
		weather.Daily = append(weather.Daily, f)
	}

	for _, d := range w.Hourly.Data {
		f := hourlyForecast{
			Date:    time.Unix(d.Time, 0),
			Icon:    fromFioIconName(d.Icon),
			Precip:  int(d.PrecipProbability * 100),
			Summary: d.Summary,
			Temp: Temperature{
				Value: d.Temp,
				Units: units,
			},
		}
		weather.Hourly = append(weather.Hourly, f)
	}

	return
}

func fromFioIconName(name string) string {
	if n, ok := fioIconNames[name]; ok {
		return n
	}
	return name
}
