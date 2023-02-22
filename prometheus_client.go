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

type node_stat_comm struct {
	Node_stats node_obj_stat `json:"/dpservices/node_obj_stats"`
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

//json members for /dp_service/node_stats command

type node_obj_stat struct {
	Rx_periodic    float64 `json:"rx_periodic"`
	Cls            float64 `json:"cls"`
	Arp            float64 `json:"arp"`
	Ip             float64 `json:"ip"`
	Ipv6_nd        float64 `json:"ipv6_nd"`
	Ipv4_lookup    float64 `json:"ipv4_lookup"`
	Ipv6_lookup    float64 `json:"ipv6_lookup"`
	Dhcp           float64 `json:"dhcp"`
	Dhcpv6         float64 `json:"dhcpv6"`
	Conntrack      float64 `json:"conntrack"`
	Dnat           float64 `json:"dnat"`
	Firewall       float64 `json:"firewall"`
	Snat           float64 `json:"snat"`
	L2_decap       float64 `json:"l2_decap"`
	Ipv6_encap     float64 `json:"ipv6_encap"`
	Geneve_tunnel  float64 `json:"geneve_tunnel"`
	Ipip_tunnel    float64 `json:"ipip_tunnel"`
	Overlay_switch float64 `json:"overlay_switch"`
	Packet_relay   float64 `json:"packet_relay"`
	Drop           float64 `json:"drop"`
	Lb             float64 `json:"lb"`
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

	rxPeriodic = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "objs_rx_periodic",
			Help: "The total number of objects processed by Rx_periodic graph node",
		},
	)

	ip = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "objs_ip",
			Help: "The total number of objects processed by Ip graph node",
		},
	)

	arp = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "objs_arp",
			Help: "The total number of objects processed by Arp graph node",
		},
	)

	cls = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "objs_cls",
			Help: "The total number of objects processed by Cls graph node",
		},
	)

	ipv4_lookup = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "objs_ipv4_lookup",
			Help: "The total number of objects processed by Ipv4 Lookup graph node",
		},
	)
	ipv6_lookup = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "objs_ipv6_lookup",
			Help: "The total number of objects processed by Ipv6 Loookup graph node",
		},
	)

	ipv6_nd = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "objs_ipv6_nd",
			Help: "The total number of objects processed by Ipv6 Nd graph node",
		},
	)
	ipip_tunnel = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "objs_ipip_tunnel",
			Help: "The total number of objects processed by IPIP Tunnel graph node",
		},
	)
	ipv6_encap = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "rx_periodic_objs_total",
			Help: "The total number of objects processed by Ipv6 Encap graph node",
		},
	)
	dhcp = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "objs_dhcp",
			Help: "The total number of objects processed by DHCP graph node",
		},
	)

	dhcpv6 = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "objs_dhcpv6",
			Help: "The total number of objects processed by DHCPV6 graph node",
		},
	)
	snat = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "objs_snat",
			Help: "The total number of objects processed by snat graph node",
		},
	)

	conntrack = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "objs_conntrack",
			Help: "The total number of objects processed by conntrack graph node",
		},
	)
	l2_decap = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "objs_l2_decap",
			Help: "The total number of objects processed by l2 decap graph node",
		},
	)

	lb = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "objs_lb",
			Help: "The total number of objects processed by lb graph node",
		},
	)
	overlay_switch = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "objs_overlay_switch",
			Help: "The total number of objects processed by overlay switch graph node",
		},
	)
	packet_relay = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "objs_packet_relay",
			Help: "The total number of objects processed by packet relay graph node",
		},
	)

	drop = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "objs_drop",
			Help: "The total number of objects processed by drop graph node",
		},
	)

	firewall = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "firewall",
			Help: "The total number of objects processed by firewall graph node",
		},
	)

	geneveTunnel = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "objs_geneve_tunnel",
			Help: "The total number of objects processed by geneve tunnel graph node",
		},
	)

	dnat = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "objs_dnat",
			Help: "The total number of objects processed by dnat graph node",
		},
	)
)

func readSocket(conn net.Conn) {
	buf := make([]byte, 1024)
	var label string
	for {
		n, err := conn.Read(buf)
		if err != nil {
			print("error 1")
			return
		}
		buf_r := buf[:n]

		if bytes.Contains(buf_r, []byte("/ethdev/list")) {
			var abc ethdev_list
			fmt.Println("list command")
			err = json.Unmarshal(buf_r, &abc)
			if err != nil {
				fmt.Println("Error in reply: ", string(buf_r), err)
				conn.Close()
				return
			}
			fmt.Println(abc)
			num_interface = len(abc.Num_interface)

		}
		if bytes.Contains(buf_r, []byte("/ethdev/info")) {
			var abc info_comm
			fmt.Println("info command")
			err = json.Unmarshal(buf_r, &abc)
			if err != nil {
				fmt.Println("Error in reply: ", string(buf_r), err)
				conn.Close()
				return
			}
			fmt.Println(abc)
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
			fmt.Println(abc)
			rxGoodPackets.WithLabelValues(label).Set(abc.Xstats.Rx_good_packets)
			txGoodPackets.WithLabelValues(label).Set(abc.Xstats.Tx_good_packets)
			rxErrors.WithLabelValues(label).Set(abc.Xstats.Rx_errors)
			txErrors.WithLabelValues(label).Set(abc.Xstats.Tx_errors)
			txGoodBytes.WithLabelValues(label).Set(abc.Xstats.Tx_good_bytes)
			rxGoodBytes.WithLabelValues(label).Set(abc.Xstats.Rx_good_bytes)
			rxMbuffAllocErrors.WithLabelValues(label).Set(abc.Xstats.Rx_mbuf_alloc_errors)
			rxMissedErrors.WithLabelValues(label).Set(abc.Xstats.Rx_missed_errors)
		}

		if bytes.Contains(buf_r, []byte("/dpservices/node_obj_stats")) {
			var abc node_stat_comm
			err = json.Unmarshal(buf_r, &abc)
			if err != nil {
				fmt.Println("Error in reply: ", string(buf_r), err)
				conn.Close()
				return
			}
			fmt.Println(abc)
			rxPeriodic.Set(abc.Node_stats.Rx_periodic)
			ip.Set(abc.Node_stats.Ip)
			cls.Set(abc.Node_stats.Cls)
			conntrack.Set(abc.Node_stats.Conntrack)
			dhcp.Set(abc.Node_stats.Dhcp)
			dhcpv6.Set(abc.Node_stats.Dhcpv6)
			snat.Set(abc.Node_stats.Snat)
			dnat.Set(abc.Node_stats.Dnat)
			ipv4_lookup.Set(abc.Node_stats.Ipv4_lookup)
			ipv6_lookup.Set(abc.Node_stats.Ipv4_lookup)
			ipv6_encap.Set(abc.Node_stats.Ipv6_encap)
			ipv6_nd.Set(abc.Node_stats.Ipv6_nd)
			l2_decap.Set(abc.Node_stats.L2_decap)
			drop.Set(abc.Node_stats.Drop)
			arp.Set(abc.Node_stats.Arp)
			overlay_switch.Set(abc.Node_stats.Overlay_switch)
			lb.Set(abc.Node_stats.Lb)
			packet_relay.Set(abc.Node_stats.Packet_relay)
			firewall.Set(abc.Node_stats.Firewall)
			geneveTunnel.Set(abc.Node_stats.Geneve_tunnel)
			ipip_tunnel.Set(abc.Node_stats.Ipip_tunnel)

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
		rxErrors, rxMissedErrors, rxPeriodic, cls, arp, ip, ipv4_lookup, ipv6_lookup, ipip_tunnel,
		ipv6_nd, l2_decap, drop, dnat, snat, conntrack, firewall, overlay_switch, packet_relay, geneveTunnel, lb,
		dhcp, dhcpv6, ipv6_encap)

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

		_, err = c.Write([]byte("/dpservices/node_obj_stats"))
		if err != nil {
			log.Fatal("write error:", err)
			break
		}
		time.Sleep(10 * time.Second)
	}

}
