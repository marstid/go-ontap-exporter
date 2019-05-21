package main

import (
	"github.com/marstid/go-ontap"
	"github.com/prometheus/client_golang/prometheus"
	log "github.com/sirupsen/logrus"
	"os"
	"strconv"
	"sync"
	"time"
)

const (
	namespace = "netapp_ontap_"
)

type Exporter struct {
	url      string
	user     string
	password string
	debug    bool
	useSSL   bool
}

func NewExporter() *Exporter {
	var e *Exporter
	// Get config details
	if os.Getenv("HOST") != "" && os.Getenv("USERID") != "" && os.Getenv("PASSWORD") != "" {
		if os.Getenv("DEBUG") == "True" {
			e = &Exporter{url: os.Getenv("HOST"), user: os.Getenv("USERID"), password: os.Getenv("PASSWORD"), debug: true}
		} else {
			e = &Exporter{url: os.Getenv("HOST"), user: os.Getenv("USERID"), password: os.Getenv("PASSWORD"), debug: false}
		}

	} else {
		log.Fatal("Missing env variables. HOST, USERID or PASSWORD")
	}

	e.useSSL = true

	return e
}

func (e *Exporter) Describe(ch chan<- *prometheus.Desc) {
	metrics := make(chan prometheus.Metric)
	go func() {
		e.Collect(metrics)
		close(metrics)
	}()
	for m := range metrics {
		ch <- m.Desc()
	}
}

func (e *Exporter) Collect(ch chan<- prometheus.Metric) {
	defer timeTrack(time.Now(), "Total Time")

	client := ontap.NewClient(e.url, e.user, e.password, e.useSSL)
	client.Debug = e.debug

	var clusterName string
	clusterInfo, err := client.GetIdClusterInfo()
	if err != nil {
		clusterName = ""
	} else {
		clusterName = clusterInfo.ClusterName
	}

	wg := sync.WaitGroup{}
	wg.Add(3)

	// Disk
	go func() {
		defer wg.Done()
		//defer timeTrack(time.Now(), "Disk")
		client := ontap.NewClient(e.url, e.user, e.password, e.useSSL)
		client.Debug = e.debug

		// Disk Performance
		dp, err := client.GetDiskPerf()
		if err != nil {
			log.Info(err.Error())
		}
		for _, value := range dp {

			val, err := strconv.ParseFloat(value.Value, 64)
			if err != nil {
				val = 0
			}

			if value.Counter == "base_for_disk_busy" {
				ch <- prometheus.MustNewConstMetric(
					prometheus.NewDesc(namespace+"disk_"+value.Counter, "Disk busy base counter", []string{}, prometheus.Labels{"disk": value.ObjectName, "cluster": clusterName}),
					prometheus.GaugeValue,
					val,
				)
			} else {

				ch <- prometheus.MustNewConstMetric(
					prometheus.NewDesc(namespace+value.Counter, "Disk busy counter", []string{}, prometheus.Labels{"disk": value.ObjectName, "cluster": clusterName}),
					prometheus.GaugeValue,
					val,
				)
			}
		}

		// Disk Info
		di, err := client.GetDiskInfo()
		if err != nil {
			log.Info(err.Error())
		}
		for _, v := range di {

			// Disk Online Status
			if v.Online {
				ch <- prometheus.MustNewConstMetric(
					prometheus.NewDesc(namespace+"disk_online", "Disk Online Status", []string{}, prometheus.Labels{"disk": v.Name, "cluster": clusterName}),
					prometheus.GaugeValue,
					float64(1),
				)
			} else {
				ch <- prometheus.MustNewConstMetric(
					prometheus.NewDesc(namespace+"disk_online", "Disk Online Status", []string{}, prometheus.Labels{"disk": v.Name, "cluster": clusterName}),
					prometheus.GaugeValue,
					float64(0),
				)
			}

			// Disk Spare Status
			if v.Spare {
				ch <- prometheus.MustNewConstMetric(
					prometheus.NewDesc(namespace+"disk_spare", "Disk Spare Status. 1 == Spare Disk", []string{}, prometheus.Labels{"disk": v.Name, "cluster": clusterName}),
					prometheus.GaugeValue,
					float64(1),
				)
			} else {
				ch <- prometheus.MustNewConstMetric(
					prometheus.NewDesc(namespace+"disk_spare", "Disk Spare Status. 1 == Spare Disk", []string{}, prometheus.Labels{"disk": v.Name, "cluster": clusterName}),
					prometheus.GaugeValue,
					float64(0),
				)
			}

			// Disk Prefail Status
			if v.Prefailed {
				ch <- prometheus.MustNewConstMetric(
					prometheus.NewDesc(namespace+"disk_prefailed", "Disk Prefailed Status. 1 == Failed", []string{}, prometheus.Labels{"disk": v.Name, "cluster": clusterName}),
					prometheus.GaugeValue,
					float64(1),
				)
			} else {
				ch <- prometheus.MustNewConstMetric(
					prometheus.NewDesc(namespace+"disk_prefailed", "Disk Prefailed Status. 1 == Failed", []string{}, prometheus.Labels{"disk": v.Name, "cluster": clusterName}),
					prometheus.GaugeValue,
					float64(0),
				)
			}
		}
	}()

	// Volume
	go func() {
		defer wg.Done()
		//defer timeTrack(time.Now(), "Volume")
		client := ontap.NewClient(e.url, e.user, e.password, e.useSSL)
		client.Debug = e.debug

		// Volume Performance
		volMap, _ := client.GetVolumeToAggrMap()

		vp, err := client.GetVolumePerf()
		if err != nil {
			log.Info(err.Error())
		}
		for _, v := range vp {
			aggr := volMap[v.ObjectName]
			val, err := strconv.ParseFloat(v.Value, 64)
			if err != nil {
				val = 0
			}

			ch <- prometheus.MustNewConstMetric(
				prometheus.NewDesc(namespace+"volume_"+v.Counter, "Volume Performance counter", []string{}, prometheus.Labels{"volume": v.ObjectName, "aggr": aggr, "cluster": clusterName}),
				prometheus.CounterValue,
				val,
			)
		}

		// Volume Info

		vi, err := client.GetVolumeInfo(100)
		if err != nil {
			log.Info(err.Error())
		}
		for _, v := range vi {

			// Volume State
			if v.State == "Online" {
				ch <- prometheus.MustNewConstMetric(
					prometheus.NewDesc(namespace+"volume_state", "Volume State. 1 == Online", []string{}, prometheus.Labels{"volume": v.Name, "aggr": v.Aggr, "cluster": clusterName}),
					prometheus.GaugeValue,
					float64(1),
				)
			} else {
				ch <- prometheus.MustNewConstMetric(
					prometheus.NewDesc(namespace+"volume_state", "Volume State. 1 == Online", []string{}, prometheus.Labels{"volume": v.Name, "aggr": v.Aggr, "cluster": clusterName}),
					prometheus.GaugeValue,
					float64(0),
				)
			}

			// Volume size
			val, err := strconv.ParseFloat(v.SizeTotal, 64)
			if err != nil {
				val = 0
			}

			ch <- prometheus.MustNewConstMetric(
				prometheus.NewDesc(namespace+"volume_size_total", "Volume size total", []string{}, prometheus.Labels{"volume": v.Name, "aggr": v.Aggr, "cluster": clusterName}),
				prometheus.GaugeValue,
				float64(val),
			)

			val, err = strconv.ParseFloat(v.SizeUsed, 64)
			if err != nil {
				val = 0
			}

			ch <- prometheus.MustNewConstMetric(
				prometheus.NewDesc(namespace+"volume_size_used", "Volume size used", []string{}, prometheus.Labels{"volume": v.Name, "aggr": v.Aggr, "cluster": clusterName}),
				prometheus.GaugeValue,
				float64(val),
			)

			val, err = strconv.ParseFloat(v.SizeFree, 64)
			if err != nil {
				val = 0
			}

			ch <- prometheus.MustNewConstMetric(
				prometheus.NewDesc(namespace+"volume_size_free", "Volume size free", []string{}, prometheus.Labels{"volume": v.Name, "aggr": v.Aggr, "cluster": clusterName}),
				prometheus.GaugeValue,
				float64(val),
			)

			// Snapshot usage
			val, err = strconv.ParseFloat(v.SnapPercentUsed, 64)
			if err != nil {
				val = 0
			}

			ch <- prometheus.MustNewConstMetric(
				prometheus.NewDesc(namespace+"volume_snap_used", "Volume percent used snapshot", []string{}, prometheus.Labels{"volume": v.Name, "aggr": v.Aggr, "cluster": clusterName}),
				prometheus.GaugeValue,
				float64(val),
			)

			val, err = strconv.ParseFloat(v.SnapPercentReserve, 64)
			if err != nil {
				val = 0
			}

			ch <- prometheus.MustNewConstMetric(
				prometheus.NewDesc(namespace+"volume_snap_reserved", "Volume percent reserved snapshot", []string{}, prometheus.Labels{"volume": v.Name, "aggr": v.Aggr, "cluster": clusterName}),
				prometheus.GaugeValue,
				float64(val),
			)

		}
	}()

	// Aggregate
	go func() {
		defer wg.Done()
		//defer timeTrack(time.Now(), "Aggregates")
		client := ontap.NewClient(e.url, e.user, e.password, e.useSSL)
		client.Debug = e.debug

		aggr, err := client.GetAggrPerf()
		if err != nil {
			log.Info(err.Error())
			return
		}

		for _, v := range aggr {

			val, err := strconv.ParseFloat(v.Value, 64)
			if err != nil {
				val = 0
			}

			ch <- prometheus.MustNewConstMetric(
				prometheus.NewDesc(namespace+"aggr_"+v.Counter, "Aggregate Performance counter "+v.Counter, []string{}, prometheus.Labels{"aggr": v.ObjectName, "cluster": clusterName}),
				prometheus.CounterValue,
				val,
			)
		}

	}()

	wg.Wait()

}
