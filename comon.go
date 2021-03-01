package main

import (
	"context"
	"encoding/json"
	"net/http"
	"sync"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/client"

	//"github.com/kr/pretty"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/prometheus/common/log"
)

func main() {
	log.Info("Starting...")
	http.HandleFunc("/metrics", responder())
	log.Fatal(http.ListenAndServe(":9099", nil))
}

func responder() func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		startTime := time.Now()
		var registry = prometheus.NewRegistry()

		log.Info("Collecting metrics from docker client...")
		containerMetrics(registry)
		log.Info("Done.")

		//Record time took to gather metrics
		registry.Register(prometheus.NewGaugeFunc(
			prometheus.GaugeOpts{
				Name: "process_metrics_gathering_time",
				Help: "Time took gathering metrics from docker",
			}, func() float64 {
				return float64(time.Since(startTime).Milliseconds())
			}))
		promhttp.HandlerFor(registry, promhttp.HandlerOpts{
			ErrorLog:      log.NewErrorLogger(),
			ErrorHandling: promhttp.ContinueOnError,
		}).ServeHTTP(w, r)
	}
}

func containerMetrics(registry *prometheus.Registry) error {
	cli, err := client.NewClientWithOpts(client.WithAPIVersionNegotiation())
	if check(err) {
		return err
	}
	clist, err := cli.ContainerList(context.Background(), types.ContainerListOptions{})
	if check(err) {
		return err
	}
	var wg sync.WaitGroup

	for _, c := range clist {
		log.Infoln("Getting stats for", c.ID[0:12], c.Names)
		container := c

		wg.Add(1)
		go func() {
			processContainer(cli, container, registry)
			wg.Done()
		}()
	}
	wg.Wait()
	cli.Close()
	return nil
}

func processContainer(cli *client.Client, c types.Container, registry *prometheus.Registry) {
	var cstat types.StatsJSON
	truncatedID := c.ID[0:12]

	stats, err := cli.ContainerStats(context.Background(), c.ID, false)
	if check(err) {
		return
	}
	err = json.NewDecoder(stats.Body).Decode(&cstat)
	if check(err) {
		return
	}
	container_registry := prometheus.WrapRegistererWith(
		prometheus.Labels{
			"id":             truncatedID,
			"container_name": c.Names[0],
		}, registry,
	)

	/* Container CPU Usage--------------------------------------------------\
	\*---------------------------------------------------------------------*/
	cpuc := prometheus.NewCounterVec(prometheus.CounterOpts{
		Name:        "cpu_usage",
		Help:        "Total cpu usage in seconds",
		ConstLabels: prometheus.Labels{},
	}, []string{"mode", "cpu"})
	container_registry.Register(cpuc)

	cpuc.WithLabelValues("user", "all").Add((float64)(cstat.CPUStats.CPUUsage.UsageInUsermode))
	cpuc.WithLabelValues("kernel", "all").Add((float64)(cstat.CPUStats.CPUUsage.UsageInKernelmode))
	cpuc.WithLabelValues("total", "all").Add((float64)(cstat.CPUStats.CPUUsage.TotalUsage))

	/* Container Memory Usage-----------------------------------------------\
	\*---------------------------------------------------------------------*/
	memory := prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name:        "memory_usage_bytes",
		Help:        "Total memory usage in bytes",
		ConstLabels: prometheus.Labels{},
	}, []string{"type"})
	container_registry.Register(memory)

	//for s := range cstat.Stats.MemoryStats.Stats {
	memory.WithLabelValues("active").Set((float64)(cstat.MemoryStats.Stats["active_anon"]))
	memory.WithLabelValues("total").Set((float64)(cstat.MemoryStats.Stats["hierarchical_memory_limit"]))
	memory.WithLabelValues("max").Set((float64)(cstat.MemoryStats.MaxUsage))
	memory.WithLabelValues("limit").Set((float64)(cstat.MemoryStats.Limit))

	/* Container Network Usage----------------------------------------------\
	\*---------------------------------------------------------------------*/
	networkTxBytes := prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name:        "network_transmit_bytes",
		Help:        "Total bytes transmitted",
		ConstLabels: prometheus.Labels{},
	}, []string{"name"})
	container_registry.Register(networkTxBytes)
	networkTxPackets := prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name:        "network_transmit_packets",
		Help:        "Total packets transmitted",
		ConstLabels: prometheus.Labels{},
	}, []string{"name"})
	container_registry.Register(networkTxPackets)
	networkTxDropped := prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name:        "network_transmit_dropped_packets",
		Help:        "Total packets dropped on transmit",
		ConstLabels: prometheus.Labels{},
	}, []string{"name"})
	container_registry.Register(networkTxDropped)
	networkTxError := prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name:        "network_transmit_errors",
		Help:        "Total transmit errors",
		ConstLabels: prometheus.Labels{},
	}, []string{"name"})
	container_registry.Register(networkTxError)
	networkRxBytes := prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name:        "network_receive_bytes",
		Help:        "Total received bytes",
		ConstLabels: prometheus.Labels{},
	}, []string{"name"})
	container_registry.Register(networkRxBytes)
	networkRxPackets := prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name:        "network_receive_packets",
		Help:        "Total received packets",
		ConstLabels: prometheus.Labels{},
	}, []string{"name"})
	container_registry.Register(networkRxPackets)
	networkRxDropped := prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name:        "network_receive_dropped_packets",
		Help:        "Total dropped packets on receive",
		ConstLabels: prometheus.Labels{},
	}, []string{"name"})
	container_registry.Register(networkRxDropped)
	networkRxError := prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name:        "network_receive_errors",
		Help:        "Total receive errors",
		ConstLabels: prometheus.Labels{},
	}, []string{"name"})
	container_registry.Register(networkRxError)

	for name := range cstat.Networks {
		//TX
		networkTxBytes.WithLabelValues(name).Set((float64)(cstat.Networks[name].TxBytes))
		networkTxPackets.WithLabelValues(name).Set((float64)(cstat.Networks[name].TxPackets))
		networkTxDropped.WithLabelValues(name).Set((float64)(cstat.Networks[name].TxDropped))
		networkTxError.WithLabelValues(name).Set((float64)(cstat.Networks[name].TxErrors))
		//RX
		networkRxBytes.WithLabelValues(name).Set((float64)(cstat.Networks[name].RxBytes))
		networkRxPackets.WithLabelValues(name).Set((float64)(cstat.Networks[name].RxPackets))
		networkRxDropped.WithLabelValues(name).Set((float64)(cstat.Networks[name].RxDropped))
		networkRxError.WithLabelValues(name).Set((float64)(cstat.Networks[name].RxErrors))
	}

}
