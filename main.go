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

	if len(os.Args) <= 2 {
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

	var packetLossSamplesLists [20]deque.Deque[bool]
	var rttSumList [20]float64
	var rttCountList [20]float64
	var hostList [20]string
	var ipAddrList [20]net.IP

	fmt.Printf("Have %d hosts\n", len(os.Args) - 1)

	for host_index := 0; host_index < (len(os.Args) - 1); host_index++ {
		host := os.Args[1+host_index]
		ips, err := net.LookupIP(host)
		if err != nil {
			fmt.Printf("Error resolving host: %v\n", err)
			os.Exit(1)
		}
		hostList[host_index] = host
		ipAddrList[host_index] = ips[0]
	}

	var lock sync.Mutex

	for host_index := 0; host_index < (len(os.Args) - 1); host_index++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			
			fmt.Printf("Pinging %s [%s]:\n", os.Args[1+host_index], ipAddrList[host_index])

			for i := 0; ; i++ {
				lock.Lock()

				rttTime, err := ping.Ping(ipAddrList[host_index], i)

				if err != nil {
					rttSumList[host_index] += 100
					rttCountList[host_index] += 1
					packetLossSamplesLists[host_index].PushBack(true)

					if packetLossSamplesLists[host_index].Len() >= 30 {
						packetLossSamplesLists[host_index].PopFront()
					}

					qualityOfServiceQueueingDelay.With(prometheus.Labels{
						"SourceIP":      hostList[host_index],
						"DestinationIP": ipAddrList[host_index].String(),
					}).Set(100)

					qualityOfServiceSmoothRTT.With(prometheus.Labels{
						"SourceIP":      hostList[host_index],
						"DestinationIP": ipAddrList[host_index].String(),
					}).Set(0.75*rttSumList[host_index]/float64(rttCountList[host_index]) + 0.25*rttTime*1000)

					qualityOfServicePacketLossRate.With(prometheus.Labels{
						"SourceIP":      hostList[host_index],
						"DestinationIP": ipAddrList[host_index].String(),
					}).Set(AverageDeque(packetLossSamplesLists[host_index]))
				} else {
					rttSumList[host_index] += (rttTime * 1000)
					rttCountList[host_index] += 1
					packetLossSamplesLists[host_index].PushBack(false)

					if packetLossSamplesLists[host_index].Len() >= 30 {
						packetLossSamplesLists[host_index].PopFront()
					}

					qualityOfServiceQueueingDelay.With(prometheus.Labels{
						"SourceIP":      hostList[host_index],
						"DestinationIP": ipAddrList[host_index].String(),
					}).Set(rttTime * 1000)

					qualityOfServiceSmoothRTT.With(prometheus.Labels{
						"SourceIP":      hostList[host_index],
						"DestinationIP": ipAddrList[host_index].String(),
					}).Set(0.75*rttSumList[host_index]/float64(rttCountList[host_index]) + 0.25*rttTime*1000)

					qualityOfServicePacketLossRate.With(prometheus.Labels{
						"SourceIP":      hostList[host_index],
						"DestinationIP": ipAddrList[host_index].String(),
					}).Set(AverageDeque(packetLossSamplesLists[host_index]))
				}

				lock.Unlock()

				time.Sleep(100 * time.Millisecond)
			}
		}()
	}

	wg.Wait()
}
