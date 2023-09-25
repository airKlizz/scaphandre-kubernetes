package main

import (
	"encoding/json"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
	ciGauge = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "carbon_intensity_fr",
		Help: "gCO₂eq/kWh of France",
	})
)

func init() {
	prometheus.MustRegister(ciGauge)
}

type EmissionHistory struct {
	Zone               string    `json:"zone"`
	CarbonIntensity    int       `json:"carbonIntensity"`
	Datetime           time.Time `json:"datetime"`
	UpdatedAt          time.Time `json:"updatedAt"`
	CreatedAt          time.Time `json:"createdAt"`
	EmissionFactorType string    `json:"emissionFactorType"`
	IsEstimated        bool      `json:"isEstimated"`
	EstimationMethod   *string   `json:"estimationMethod"`
}

type EmissionData struct {
	Zone    string            `json:"zone"`
	History []EmissionHistory `json:"history"`
}

func fetchCarbonIntensity() float64 {
	url := "https://api-access.electricitymaps.com/free-tier/carbon-intensity/history?zone=FR"

	req, _ := http.NewRequest("GET", url, nil)

	req.Header.Add("auth-token", os.Getenv("AUTH_TOKEN"))

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		log.Printf("API call failed: %v", err)
		return 0.0
	}
	defer res.Body.Close()

	var data EmissionData

	if err := json.NewDecoder(res.Body).Decode(&data); err != nil {
		log.Printf("Error decoding JSON: %v", err)
		return 0.0
	}

	// We don't take the latest value because estimated, so we take the 4th value starting from the end to make sure is not estimated.
	// It makes a difference of ~6 hours between the metric time and the carbon intensity time
	timeOffset := 4
	if len(data.History) < timeOffset {
		log.Printf("No history: %+v", data)
		return 0.0
	}
	carbonIntensity := data.History[len(data.History)-timeOffset].CarbonIntensity

	return float64(carbonIntensity)
}

func main() {
	http.Handle("/metrics", promhttp.Handler())

	go func() {
		for {
			ci := fetchCarbonIntensity()
			ciGauge.Set(ci)
			time.Sleep(10 * time.Minute)
		}
	}()

	log.Fatal(http.ListenAndServe(":8000", nil))
}
