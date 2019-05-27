package main

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/prometheus/common/version"
	log "github.com/sirupsen/logrus"
	"net/http"
	"os"
	"strconv"
	"time"
)

func init() {
	prometheus.MustRegister(version.NewCollector("netapp_exporter"))
	prometheus.MustRegister(NewExporter())

}

func main() {
	defer timeTrack(time.Now(), "Main")

	port := 9099

	if os.Getenv("PORT") != "" {
		i1, err := strconv.Atoi(os.Getenv("PORT"))
		if err == nil {
			port = i1

		}
	}

	http.Handle("/metrics", promhttp.Handler())
	http.HandleFunc("/", redirect)

	log.Info("Serving Netapp metrics on :" + strconv.FormatInt(int64(port), 10))
	log.Fatal(http.ListenAndServe(":"+strconv.FormatInt(int64(port), 10), nil))

}

func redirect(w http.ResponseWriter, r *http.Request) {
	http.Redirect(w, r, "/metrics", 301)
}
