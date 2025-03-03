package main

import (
	"clyde1811/qosmonitor/ping"
	"fmt"
	"net"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/gammazero/deque"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

func AverageDeque(deque deque.Deque[bool]) float64 {
	sum := 0.0

	for i := 0; i < deque.Len(); i++ {
		if deque.At(i) {
			sum += 1.0
		}
	}

	return sum / float64(deque.Len())
}

func main() {

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
		}, []string{"SourceIP", "DestinationIP"})

		qualityOfServiceQueueingDelay := promauto.NewGaugeVec(prometheus.GaugeOpts{
			Name: "quality_of_service_queueing_delay_millisecond",
			Help: "The duration of queueing delay (in millisecond)",
		}, []string{"SourceIP", "DestinationIP"})

		qualityOfServiceSmoothRTT := promauto.NewGaugeVec(prometheus.GaugeOpts{
			Name: "quality_of_service_smooth_RTT_millisecond",
			Help: "Smooth RTT of queueing delay (in millisecond)",
		}, []string{"SourceIP", "DestinationIP"})

		host := os.Args[1]
		ips, err := net.LookupIP(host)
		if err != nil {
			fmt.Printf("Error resolving host: %v\n", err)
			os.Exit(1)
		}

		ipAddr := ips[0]
		fmt.Printf("Pinging %s [%s]:\n", os.Args[1], ipAddr)

		rttSum := 0.0
		rttCount := 0
		var packet_loss_samples deque.Deque[bool]

		for i := 0; ; i++ {
			rttTime, err := ping.Ping(ipAddr, i)

			if err != nil {
				rttSum += 100
				rttCount += 1
				packet_loss_samples.PushBack(true)

				if packet_loss_samples.Len() >= 30 {
					packet_loss_samples.PopFront()
				}

				qualityOfServiceQueueingDelay.With(prometheus.Labels{
					"SourceIP":      host,
					"DestinationIP": ipAddr.String(),
				}).Set(100)

				qualityOfServiceSmoothRTT.With(prometheus.Labels{
					"SourceIP":      host,
					"DestinationIP": ipAddr.String(),
				}).Set(0.75*rttSum/float64(rttCount) + 0.25*rttTime*1000)

				qualityOfServicePacketLossRate.With(prometheus.Labels{
					"SourceIP":      host,
					"DestinationIP": ipAddr.String(),
				}).Set(AverageDeque(packet_loss_samples))
			} else {
				rttSum += (rttTime * 1000)
				rttCount += 1
				packet_loss_samples.PushBack(false)

				if packet_loss_samples.Len() >= 30 {
					packet_loss_samples.PopFront()
				}

				qualityOfServiceQueueingDelay.With(prometheus.Labels{
					"SourceIP":      host,
					"DestinationIP": ipAddr.String(),
				}).Set(rttTime * 1000)

				qualityOfServiceSmoothRTT.With(prometheus.Labels{
					"SourceIP":      host,
					"DestinationIP": ipAddr.String(),
				}).Set(0.75*rttSum/float64(rttCount) + 0.25*rttTime*1000)

				qualityOfServicePacketLossRate.With(prometheus.Labels{
					"SourceIP":      host,
					"DestinationIP": ipAddr.String(),
				}).Set(AverageDeque(packet_loss_samples))
			}

			time.Sleep(100 * time.Millisecond)
		}
	}()

	wg.Wait()
}
