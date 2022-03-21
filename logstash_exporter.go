package main

import (
	"github.com/awesomexido/logstash_exporter/collector"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/common/version"
	"gopkg.in/alecthomas/kingpin.v2"
	"net/http"
	_ "net/http/pprof"
	"sync"
	"time"
)

var (
	scrapeDurations = prometheus.NewSummaryVec(
		prometheus.SummaryOpts{
			Namespace: collector.Namespace,
			Subsystem: "exporter",
			Name:      "scrape_duration_seconds",
			Help:      "logstash_exporter: Duration of a scrape job.",
		},
		[]string{"collector", "result"},
	)
)

// LogstashCollector collector type
type LogstashCollector struct {
	collectors map[string]collector.Collector
}

// NewLogstashCollector register a logstash collector
func NewLogstashCollector(logstashEndpoint string) (*LogstashCollector, error) {
	nodeStatsCollector, err := collector.NewNodeStatsCollector(logstashEndpoint)
	if err != nil {
	}

	nodeInfoCollector, err := collector.NewNodeInfoCollector(logstashEndpoint)
	if err != nil {
	}

	return &LogstashCollector{
		collectors: map[string]collector.Collector{
			"node": nodeStatsCollector,
			"info": nodeInfoCollector,
		},
	}, nil
}

func listen(exporterBindAddress string) {
	http.Handle("/metrics", prometheus.Handler())
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/metrics", http.StatusMovedPermanently)
	})

	if err := http.ListenAndServe(exporterBindAddress, nil); err != nil {
	}
}

// Describe logstash metrics
func (coll LogstashCollector) Describe(ch chan<- *prometheus.Desc) {
	scrapeDurations.Describe(ch)
}

// Collect logstash metrics
func (coll LogstashCollector) Collect(ch chan<- prometheus.Metric) {
	wg := sync.WaitGroup{}
	wg.Add(len(coll.collectors))
	for name, c := range coll.collectors {
		go func(name string, c collector.Collector) {
			execute(name, c, ch)
			wg.Done()
		}(name, c)
	}
	wg.Wait()
	scrapeDurations.Collect(ch)
}

func execute(name string, c collector.Collector, ch chan<- prometheus.Metric) {
	begin := time.Now()
	err := c.Collect(ch)
	duration := time.Since(begin)
	var result string

	if err != nil {

		result = "error"
	} else {
		result = "success"
	}
	scrapeDurations.WithLabelValues(name, result).Observe(duration.Seconds())
}

func init() {
	prometheus.MustRegister(version.NewCollector("logstash_exporter"))
}

func main() {
	var (
		logstashEndpoint    = kingpin.Flag("logstash.endpoint", "The protocol, host and port on which logstash metrics API listens").Default("http://localhost:9600").String()
		exporterBindAddress = kingpin.Flag("web.listen-address", "Address on which to expose metrics and web interface.").Default(":9198").String()
	)

	kingpin.Version(version.Print("logstash_exporter"))
	kingpin.HelpFlag.Short('h')
	kingpin.Parse()

	logstashCollector, err := NewLogstashCollector(*logstashEndpoint)
	if err != nil {

	}

	prometheus.MustRegister(logstashCollector)

	listen(*exporterBindAddress)
}
