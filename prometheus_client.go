package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// xstat command
type xstat_comm struct {
	Xstats ethdev_exstat `json:"/ethdev/xstats"`
}

// info command
type info_comm struct {
	Info ethdev_info `json:"/ethdev/info"`
}

// list command
type ethdev_list struct {
	Num_interface []int16 `json:"/ethdev/list"`
}

// json members for xstat command
type ethdev_exstat struct {
	Rx_good_packets      float64 `json:"rx_good_packets"`
	Tx_good_packets      float64 `json:"tx_good_packets"`
	Rx_good_bytes        float64 `json:"rx_good_bytes"`
	Tx_good_bytes        float64 `json:"tx_good_bytes"`
	Rx_missed_errors     float64 `json:"rx_missed_errors"`
	Rx_errors            float64 `json:"rx_errors"`
	Tx_errors            float64 `json:"tx_errors"`
	Rx_mbuf_alloc_errors float64 `json:"rx_mbuf_allocation_errors"`
}

// json members for info command
type ethdev_info struct {
	Interface_name string `json:"name"`
}

var num_interface int = 0

var (
	rxGoodPackets = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "rx_good_packets_total",
			Help: "The total number of rx packets on an interface",
		},
		[]string{"interface"},
	)
	txGoodPackets = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "tx_good_packets_total",
			Help: "The total number of tx packets on an interface",
		},
		[]string{"interface"},
	)
	rxGoodBytes = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "rx_good_bytes_total",
			Help: "The total number of rx bytes on an interface",
		},
		[]string{"interface"},
	)
	txGoodBytes = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "tx_good_bytes_total",
			Help: "The total number of tx bytes on an interface",
		},
		[]string{"interface"},
	)
	rxErrors = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "rx_errors_total",
			Help: "The total number of rx errors on an interface",
		},
		[]string{"interface"},
	)
	txErrors = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "tx_errors_total",
			Help: "The total number of tx errors on an interface",
		},
		[]string{"interface"},
	)
	rxMissedErrors = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "rx_missed_errors_total",
			Help: "The total number of rx missed errors on an interface",
		},
		[]string{"interface"},
	)
	rxMbuffAllocErrors = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "rx_mbuf_alloc_errors_total",
			Help: "The total number of rx missed errors on an interface",
		},
		[]string{"interface"},
	)

	// Graph node objects stats

	objs_num = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "obj_count",
			Help: "The total number of objects processed by Rx_periodic graph node",
		},
		[]string{"graph_node"},
	)
)

func readSocket(conn net.Conn) {
	buf := make([]byte, 1024*16)
	var label string
	for {
		n, err := conn.Read(buf)
		if err != nil {
			print("Socket read error")
			return
		}
		buf_r := buf[:n]

		if bytes.Contains(buf_r, []byte("/ethdev/list")) {
			var abc ethdev_list
			err = json.Unmarshal(buf_r, &abc)
			if err != nil {
				fmt.Println("Error in reply: ", string(buf_r), err)
				conn.Close()
				return
			}
			num_interface = len(abc.Num_interface)

		}
		if bytes.Contains(buf_r, []byte("/ethdev/info")) {
			var abc info_comm
			err = json.Unmarshal(buf_r, &abc)
			if err != nil {
				fmt.Println("Error in reply: ", string(buf_r), err)
				conn.Close()
				return
			}
			label = abc.Info.Interface_name

		}

		if bytes.Contains(buf_r, []byte("/ethdev/xstats")) {
			var abc xstat_comm
			err = json.Unmarshal(buf_r, &abc)
			if err != nil {
				fmt.Println("Error in reply: ", string(buf_r), err)
				conn.Close()
				return
			}
			rxGoodPackets.WithLabelValues(label).Set(abc.Xstats.Rx_good_packets)
			txGoodPackets.WithLabelValues(label).Set(abc.Xstats.Tx_good_packets)
			rxErrors.WithLabelValues(label).Set(abc.Xstats.Rx_errors)
			txErrors.WithLabelValues(label).Set(abc.Xstats.Tx_errors)
			txGoodBytes.WithLabelValues(label).Set(abc.Xstats.Tx_good_bytes)
			rxGoodBytes.WithLabelValues(label).Set(abc.Xstats.Rx_good_bytes)
			rxMbuffAllocErrors.WithLabelValues(label).Set(abc.Xstats.Rx_mbuf_alloc_errors)
			rxMissedErrors.WithLabelValues(label).Set(abc.Xstats.Rx_missed_errors)
		}

		if bytes.Contains(buf_r, []byte("/dp_service/graph/obj_count")) {
			var results map[string]interface{}
			err = json.Unmarshal(buf_r, &results)
			if err != nil {
				fmt.Println("Error in reply: ", string(buf_r), err)
				conn.Close()
				return
			}
			for k := range results {
				switch outer_map := results[k].(type) {
				case map[string]interface{}:
					for l := range outer_map {
						switch inner_map := outer_map[l].(type) {
						case map[string]interface{}:
							for m := range inner_map {
								switch last_element := inner_map[m].(type) {
								case float64:
									objs_num.WithLabelValues(m).Set(last_element)

								}
							}

						}

					}
				}
			}

		}

	}
}

func main() {
	bind := ""
	flagset := flag.NewFlagSet(os.Args[0], flag.ExitOnError)
	flagset.StringVar(&bind, "bind", ":2112", "The socket to bind to.")

	r := prometheus.NewRegistry()
	r.MustRegister(rxGoodPackets, txGoodPackets, txGoodBytes,
		rxGoodBytes, rxMbuffAllocErrors, txErrors,
		rxErrors, rxMissedErrors, objs_num)

	mux := http.NewServeMux()
	mux.Handle("/metrics", promhttp.HandlerFor(r, promhttp.HandlerOpts{}))

	c, err := net.Dial("unixpacket", "/var/run/dpdk/rte/dpdk_telemetry.v2")
	if err != nil {
		panic(err)
	}
	defer c.Close()

	go readSocket(c)

	go func() {
		log.Fatal(http.ListenAndServe(":2112", mux))
	}()

	_, err = c.Write([]byte("/ethdev/list"))
	if err != nil {
		log.Fatal("write error:", err)
	}

	for {
		for j := 0; j < num_interface; j++ {

			_, err = c.Write([]byte("/ethdev/info," + strconv.Itoa(j)))
			if err != nil {
				log.Fatal("write error:", err)
				break
			}

			_, err = c.Write([]byte("/ethdev/xstats," + strconv.Itoa(j)))
			if err != nil {
				log.Fatal("write error:", err)
				break
			}
			time.Sleep(100 * time.Millisecond)
		}

		_, err = c.Write([]byte("/dp_service/graph/obj_count"))
		if err != nil {
			log.Fatal("write error:", err)
			break
		}
		time.Sleep(10 * time.Second)
	}

}
