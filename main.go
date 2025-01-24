package main

import (
	"clyde1811/qosmonitor/ping"
	"fmt"
	"net"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

func main(){
	
	if len(os.Args) != 2 {
		fmt.Println("Usage: go run ping.go <host>")
		os.Exit(1)
	}

	var wg sync.WaitGroup

	wg.Add(1)
	go func() {
		defer wg.Done()
		http.Handle("/metrics", promhttp.Handler())
		http.ListenAndServe(":2112", nil)
	}()
	
	wg.Add(1)
	go func() {
		defer wg.Done()

		qualityOfServicePacketLossRate := promauto.NewGaugeVec(prometheus.GaugeOpts{
			Name: "quality_of_service_packet_loss_rate",
			Help: "The rate of Packet Loss",
		}, []string{"SourceIP", "DestinationIP"});

		qualityOfServiceQueueingDelay := promauto.NewGaugeVec(prometheus.GaugeOpts{
			Name: "quality_of_service_queueing_delay_millisecond",
			Help: "The duration of queueing delay (in millisecond)",
		}, []string{"SourceIP", "DestinationIP"});

		host := os.Args[1]
		ips, err := net.LookupIP(host)
		if err != nil {
			fmt.Printf("Error resolving host: %v\n", err)
			os.Exit(1)
		}

		ipAddr := ips[0]
		fmt.Printf("Pinging %s [%s]:\n", os.Args[1], ipAddr)

		for i := 1; i <= 1000; i += 1{
			rttTime, err := ping.Ping(ipAddr, i)

			if err != nil {
				time.Sleep(500 * time.Millisecond)
				continue
			}

			qualityOfServiceQueueingDelay.With(prometheus.Labels{
				"SourceIP":      "192.168.43.11",
				"DestinationIP":  ipAddr.String(),
			}).Set(rttTime * 1000)
			
			qualityOfServicePacketLossRate.With(prometheus.Labels{
				"SourceIP":      "192.168.43.11",
				"DestinationIP":  ipAddr.String(),
			}).Set(0)

			time.Sleep(500 * time.Millisecond)
		}
	}()

	wg.Wait()
}