package main

import (
	"fmt"
	"net/http"
	"runtime"

	"github.com/iotaledger/giota"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/prometheus/common/log"
	"gopkg.in/alecthomas/kingpin.v2"
)

// Version is set during build to the git describe version
// (semantic version)-(commitish) form.
var Version = "0.2.0 dev"

var (
	listenAddress = kingpin.Flag("web.listen-address", "Address to listen on for web interface and telemetry.").Default(":9187").String()
	metricPath    = kingpin.Flag("web.telemetry-path", "Path under which to expose metrics.").Default("/metrics").String()
	targetAddress = kingpin.Flag("web.iri-path", "URI of the IOTA IRI Node to scrape.").Default("http://localhost:14265").String()
)

const (
	namespace = "iota-iri"
)

type Exporter struct {
	iriAddress string

	iota_node_info_duration                   prometheus.Gauge
	iota_node_info_available_processors       prometheus.Gauge
	iota_node_info_free_memory                prometheus.Gauge
	iota_node_info_max_memory                 prometheus.Gauge
	iota_node_info_total_memory               prometheus.Gauge
	iota_node_info_latest_milestone           prometheus.Gauge
	iota_node_info_latest_subtangle_milestone prometheus.Gauge
	iota_node_info_total_neighbors            prometheus.Gauge
	iota_node_info_total_tips                 prometheus.Gauge
	iota_node_info_total_transactions_queued  prometheus.Gauge
	iota_node_info_totalScrapes               prometheus.Counter
	iota_neighbors_info_total_neighbors       prometheus.Gauge
	iota_neighbors_info_active_neighbors      prometheus.Gauge
	iota_neighbors_new_transactions           *prometheus.GaugeVec
	iota_neighbors_random_transactions        *prometheus.GaugeVec
	iota_neighbors_all_transactions           *prometheus.GaugeVec
	iota_neighbors_invalid_transactions       *prometheus.GaugeVec
	iota_neighbors_sent_transactions          *prometheus.GaugeVec
	iota_neighbors_active                     *prometheus.GaugeVec
	iota_zmq_seen_tx_count                    prometheus.Gauge
	iota_zmq_txs_with_value_count prometheus.Gauge
	iota_zmq_confirmed_tx_count prometheus.Gauge
	iota_zmq_to_request prometheus.Gauge
	iota_zmq_to_process prometheus.Gauge
	iota_zmq_to_broadcast prometheus.Gauge
	iota_zmq_to_reply prometheus.Gauge
	iota_zmq_total_transactions prometheus.Gauge

}

func NewExporter(iriAddress string) *Exporter {
	return &Exporter{
		iriAddress: iriAddress,

		iota_node_info_duration: prometheus.NewGauge(
			prometheus.GaugeOpts{
				//Namespace: namespace,
				//Subsystem: "exporter",
				//Name: "duration",
				Name: "iota_node_info_duration",
				Help: "Response time of getting Node Info.",
			}),

		iota_node_info_available_processors: prometheus.NewGauge(
			prometheus.GaugeOpts{
				//Namespace: namespace,
				//Subsystem: "exporter",
				//Name: "available_processors",
				Name: "iota_node_info_available_processors",
				Help: "Number of cores available in this Node.",
			}),

		iota_node_info_free_memory: prometheus.NewGauge(
			prometheus.GaugeOpts{
				//Namespace: namespace,
				//Subsystem: "exporter",
				//Name: "free_memory",
				Name: "iota_node_info_free_memory",
				Help: "Free Memory in this IRI instance.",
			}),

		iota_node_info_max_memory: prometheus.NewGauge(
			prometheus.GaugeOpts{
				//Namespace: namespace,
				//Subsystem: "exporter",
				//Name: "max_memory",
				Name: "iota_node_info_max_memory",
				Help: "Max Memory in this IRI instance.",
			}),

		iota_node_info_total_memory: prometheus.NewGauge(
			prometheus.GaugeOpts{
				//Namespace: namespace,
				//Subsystem: "exporter",
				//Name: "total_memory",
				Name: "iota_node_info_total_memory",
				Help: "Total Memory in this IRI instance.",
			}),

		iota_node_info_latest_milestone: prometheus.NewGauge(
			prometheus.GaugeOpts{
				//Namespace: namespace,
				//Subsystem: "exporter",
				//Name: "latest_milestone",
				Name: "iota_node_info_latest_milestone",
				Help: "Tangle milestone at the interval.",
			}),

		iota_node_info_latest_subtangle_milestone: prometheus.NewGauge(
			prometheus.GaugeOpts{
				//Namespace: namespace,
				//Subsystem: "exporter",
				//Name: "latest_subtangle_milestone",
				Name: "iota_node_info_latest_subtangle_milestone",
				Help: "Subtangle milestone at the interval.",
			}),

		iota_node_info_total_neighbors: prometheus.NewGauge(
			prometheus.GaugeOpts{
				//Namespace: namespace,
				//Subsystem: "exporter",
				//Name: "total_neighbors",
				Name: "iota_node_info_total_neighbors",
				Help: "Total neighbors at the interval.",
			}),

		iota_node_info_total_tips: prometheus.NewGauge(
			prometheus.GaugeOpts{
				//Namespace: namespace,
				//Subsystem: "exporter",
				//Name: "total_tips",
				Name: "iota_node_info_total_tips",
				Help: "Total tips at the interval.",
			}),

		iota_node_info_total_transactions_queued: prometheus.NewGauge(
			prometheus.GaugeOpts{
				//Namespace: namespace,
				//Subsystem: "exporter",
				//Name: "total_transactions_queued",
				Name: "iota_node_info_total_transactions_queued",
				Help: "Total open txs at the interval.",
			}),

		iota_node_info_totalScrapes: prometheus.NewCounter(
			prometheus.CounterOpts{
				//Namespace: namespace,
				//Subsystem: "exporter",
				//Name: "scrapes_total",
				Name: "iota_node_info_scrapes_total",
				Help: "Total number of scrapes.",
			}),

		iota_neighbors_info_total_neighbors: prometheus.NewGauge(
			prometheus.GaugeOpts{
				//Namespace: namespace,
				//Subsystem: "exporter",
				//Name: "total_neighbors_ws",
				Name: "iota_neighbors_info_total_neighbors",
				Help: "Total number of neighbors as received in the getNeighbors ws call.",
			}),

		iota_neighbors_info_active_neighbors: prometheus.NewGauge(
			prometheus.GaugeOpts{
				//Namespace: namespace,
				//Subsystem: "exporter",
				//Name: "total_neighbors_ws",
				Name: "iota_neighbors_info_active_neighbors",
				Help: "Total number of neighbors that are active.",
			}),

		iota_neighbors_new_transactions: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				//Namespace: namespace,
				//Subsystem: "exporter",
				//Name: "neighbors_new_transactions",
				Name: "iota_neighbors_new_transactions",
				Help: "Number of New Transactions for a specific Neighbor.",
			},
			[]string{"id"},
		),

		iota_neighbors_random_transactions: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				//Namespace: namespace,
				//Subsystem: "exporter",
				//Name: "neighbors_random_transactions",
				Name: "iota_neighbors_random_transactions",
				Help: "Number of Random Transactions for a specific Neighbor.",
			},
			[]string{"id"},
		),

		iota_neighbors_all_transactions: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				//Namespace: namespace,
				//Subsystem: "exporter",
				//Name: "neighbors_all_transactions",
				Name: "iota_neighbors_all_transactions",
				Help: "Number of All Transaction Types for a specific Neighbor.",
			},
			[]string{"id"},
		),

		iota_neighbors_invalid_transactions: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				//Namespace: namespace,
				//Subsystem: "exporter",
				//Name: "neighbors_invalid_transactions",
				Name: "iota_neighbors_invalid_transactions",
				Help: "Number of Invalid Transactions for a specific Neighbor.",
			},
			[]string{"id"},
		),

		iota_neighbors_sent_transactions: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				//Namespace: namespace,
				//Subsystem: "exporter",
				//Name: "neighbors_sent_transactions",
				Name: "iota_neighbors_sent_transactions",
				Help: "Number of Invalid Transactions for a specific Neighbor.",
			},
			[]string{"id"},
		),

		iota_neighbors_active: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				//Namespace: namespace,
				//Subsystem: "exporter",
				//Name: "neighbors_sent_transactions",
				Name: "iota_neighbors_active",
				Help: "Report if the Neighbor Active based on incoming transactions.",
			},
			[]string{"id"},
		),

		iota_zmq_seen_tx_count: prometheus.NewGauge(
			prometheus.GaugeOpts{
				//Namespace: namespace,
				//Subsystem: "zmq",
				//Name: "zmq_seen_tx_count",
				Name: "iota_zmq_seen_tx_count",
				Help: "Count of transactions seen by zeroMQ.",
			}),

		iota_zmq_txs_with_value_count: prometheus.NewGauge(
			prometheus.GaugeOpts{
				//Namespace: namespace,
				//Subsystem: "zmq",
				//Name: "zmq_txs_with_value_count",
				Name: "iota_zmq_txs_with_value_count",
				Help: "Count of transactions seen by zeroMQ that have a non-zero value.",
			}),

		iota_zmq_confirmed_tx_count: prometheus.NewGauge(
			prometheus.GaugeOpts{
				//Namespace: namespace,
				//Subsystem: "zmq",
				//Name: "zmq_confirmed_tx_count",
				Name: "iota_zmq_confirmed_tx_count",
				Help: "Count of transactions confirmed by zeroMQ.",
			}),

		iota_zmq_to_process: prometheus.NewGauge(
			prometheus.GaugeOpts{
				//Namespace: namespace,
				//Subsystem: "zmq",
				//Name: "zmq_to_process",
				Name: "iota_zmq_to_process",
				Help: "toProcess from RSTAT output of ZMQ.",
			}),

		iota_zmq_to_broadcast: prometheus.NewGauge(
			prometheus.GaugeOpts{
				//Namespace: namespace,
				//Subsystem: "zmq",
				//Name: "zmq_to_broadcast",
				Name: "iota_zmq_to_broadcast",
				Help: "toBroadcast from RSTAT output of ZMQ.",
			}),

		iota_zmq_to_request: prometheus.NewGauge(
			prometheus.GaugeOpts{
				//Namespace: namespace,
				//Subsystem: "zmq",
				//Name: "zmq_to_request",
				Name: "iota_zmq_to_request",
				Help: "toRequest from RSTAT output of ZMQ.",
			}),

		iota_zmq_to_reply: prometheus.NewGauge(
			prometheus.GaugeOpts{
				//Namespace: namespace,
				//Subsystem: "zmq",
				//Name: "zmq_to_reply",
				Name: "iota_zmq_to_reply",
				Help: "toReply from RSTAT output of ZMQ.",
			}),

		iota_zmq_total_transactions: prometheus.NewGauge(
			prometheus.GaugeOpts{
				//Namespace: namespace,
				//Subsystem: "zmq",
				//Name: "zmq_total_transactions",
				Name: "iota_zmq_total_transactions",
				Help: "totalTransactions from RSTAT output of ZMQ.",
			}),
	}
}

func (e *Exporter) Describe(ch chan<- *prometheus.Desc) {

	ch <- e.iota_node_info_duration.Desc()
	ch <- e.iota_node_info_available_processors.Desc()
	ch <- e.iota_node_info_free_memory.Desc()
	ch <- e.iota_node_info_max_memory.Desc()
	ch <- e.iota_node_info_total_memory.Desc()
	ch <- e.iota_node_info_latest_milestone.Desc()
	ch <- e.iota_node_info_latest_subtangle_milestone.Desc()
	ch <- e.iota_node_info_total_neighbors.Desc()
	ch <- e.iota_node_info_total_tips.Desc()
	ch <- e.iota_node_info_total_transactions_queued.Desc()
	ch <- e.iota_node_info_totalScrapes.Desc()

	ch <- e.iota_neighbors_info_total_neighbors.Desc()
	ch <- e.iota_neighbors_info_active_neighbors.Desc()

	e.iota_neighbors_new_transactions.Describe(ch)
	e.iota_neighbors_random_transactions.Describe(ch)
	e.iota_neighbors_all_transactions.Describe(ch)
	e.iota_neighbors_invalid_transactions.Describe(ch)
	e.iota_neighbors_sent_transactions.Describe(ch)
	e.iota_neighbors_active.Describe(ch)

	ch <- e.iota_zmq_seen_tx_count.Desc()
	ch <- e.iota_zmq_txs_with_value_count.Desc()
	ch <- e.iota_zmq_confirmed_tx_count.Desc()
	ch <- e.iota_zmq_to_process.Desc()
	ch <- e.iota_zmq_to_broadcast.Desc()
	ch <- e.iota_zmq_to_request.Desc()
	ch <- e.iota_zmq_to_reply.Desc()
	ch <- e.iota_zmq_total_transactions.Desc()
}

func (e *Exporter) Collect(ch chan<- prometheus.Metric) {
	e.scrape(ch)
	ch <- e.iota_node_info_duration
	ch <- e.iota_node_info_available_processors
	ch <- e.iota_node_info_free_memory
	ch <- e.iota_node_info_max_memory
	ch <- e.iota_node_info_total_memory
	ch <- e.iota_node_info_latest_milestone
	ch <- e.iota_node_info_latest_subtangle_milestone
	ch <- e.iota_node_info_total_neighbors
	ch <- e.iota_node_info_total_tips
	ch <- e.iota_node_info_total_transactions_queued
	ch <- e.iota_node_info_totalScrapes

	ch <- e.iota_neighbors_info_total_neighbors
	ch <- e.iota_neighbors_info_active_neighbors

	e.iota_neighbors_new_transactions.Collect(ch)
	e.iota_neighbors_random_transactions.Collect(ch)
	e.iota_neighbors_all_transactions.Collect(ch)
	e.iota_neighbors_invalid_transactions.Collect(ch)
	e.iota_neighbors_sent_transactions.Collect(ch)
	e.iota_neighbors_active.Collect(ch)

	ch <- e.iota_zmq_seen_tx_count
	ch <- e.iota_zmq_txs_with_value_count
	ch <- e.iota_zmq_confirmed_tx_count
	ch <- e.iota_zmq_to_process
	ch <- e.iota_zmq_to_broadcast
	ch <- e.iota_zmq_to_request
	ch <- e.iota_zmq_to_reply
	ch <- e.iota_zmq_total_transactions
}

func (e *Exporter) scrape(ch chan<- prometheus.Metric) {

	api := giota.NewAPI(e.iriAddress, nil)

	scrape_nodeinfo(e, api)
	scrape_neighbors(e, api)	
	scrape_zmq(e)
}

func main() {
	kingpin.Version(fmt.Sprintf("iota-iri_exporter %s (built with %s)\n", Version, runtime.Version()))
	log.AddFlags(kingpin.CommandLine)
	kingpin.Parse()

	// landingPage contains the HTML served at '/'.
	// TODO: Make this nicer and more informative.
	var landingPage = []byte(`<html>
	<head><title>Iota-IRI Exporter</title></head>
	<body>
	<h1>Iota-IRI Node Exporter</h1>
	<p><a href='` + *metricPath + `'>Metrics</a></p>
	</body>
	</html>
	`)

	exporter := NewExporter(*targetAddress)
	prometheus.MustRegister(exporter)

	http.Handle(*metricPath, promhttp.Handler())
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Write(landingPage) // nolint: errcheck
	})

	log.Infof("Starting %s_exporter Server on port %s monitoring %s", namespace, *listenAddress, *targetAddress)
	log.Fatal(http.ListenAndServe(*listenAddress, nil))
}
