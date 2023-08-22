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

func fetchCarbonIntensity() float64 {
	url := "https://api-access.electricitymaps.com/free-tier/carbon-intensity/latest?zone=FR"

	req, _ := http.NewRequest("GET", url, nil)

	req.Header.Add("auth-token", os.Getenv("AUTH_TOKEN"))

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		log.Printf("API call failed: %v", err)
		return 0.0
	}
	defer res.Body.Close()

	var data map[string]interface{}

	if err := json.NewDecoder(res.Body).Decode(&data); err != nil {
		log.Printf("Error decoding JSON: %v", err)
		return 0.0
	}

	latestCI := data["carbonIntensity"].(float64)

	return latestCI
}

func main() {
	http.Handle("/metrics", promhttp.Handler())

	go func() {
		for {
			ci := fetchCarbonIntensity()
			ciGauge.Set(ci)
			time.Sleep(time.Hour)
		}
	}()

	log.Fatal(http.ListenAndServe(":8000", nil))
}
