package ping

import (
	"errors"
	"fmt"
	"math/rand"
	"net"
	"time"

	"golang.org/x/net/icmp"
	"golang.org/x/net/ipv4"
)

func Ping(addr net.IP, sequenceID int) (queueingDelay float64, e error) {
	// Resolve the address
	

	conn, err := icmp.ListenPacket("ip4:icmp", "0.0.0.0")
	if err != nil {
		fmt.Printf("Error creating connection: %v\n", err)
		return -1, errors.New("error on creating connection")
	}
	defer conn.Close()

	// Set timeout for the connection
	conn.SetDeadline(time.Now().Add(5 * time.Second))

	// Generate a random ID for the ICMP packet
	id := rand.Intn(0xffff)

	// Create an ICMP Echo Request message
	echo := &icmp.Message{
		Type: ipv4.ICMPTypeEcho,
		Code: 0,
		Body: &icmp.Echo{
			ID:   id,
			Seq:  sequenceID,
			Data: []byte("Go Ping!"),
		},
	}

	// Marshal the ICMP message into binary form
	messageBytes, err := echo.Marshal(nil)
	if err != nil {
		fmt.Printf("Error marshaling ICMP message: %v\n", err)
		return -1, errors.New("error on marshaling ICMP message")
	}

	// Send the ICMP packet
	start := time.Now()
	_, err = conn.WriteTo(messageBytes, &net.IPAddr{IP: addr})
	if err != nil {
		fmt.Printf("Error sending ICMP message: %v\n", err)
		return -1, errors.New("error on sending ICMP message")
	}

	// Buffer to read the reply
	reply := make([]byte, 1500)
	n, peer, err := conn.ReadFrom(reply)
	if err != nil {
		fmt.Printf("Error reading ICMP reply: %v\n", err)
		return -1,  errors.New("error on reading ICMP reply")
	}

	duration := time.Since(start)

	// Parse the reply
	parsedMessage, err := icmp.ParseMessage(1, reply[:n]) // Protocol number 1 = ICMP
	if err != nil {
		fmt.Printf("Error parsing ICMP reply: %v\n", err)
		return -1, errors.New("error on parsing ICMP reply")
	}

	switch parsedMessage.Type {
	case ipv4.ICMPTypeEchoReply:
		fmt.Printf("Reply from %s: sequence=%v time=%v\n", peer, sequenceID, duration)
		return float64(duration.Seconds()), nil
	default:
		fmt.Printf("Unexpected ICMP message type: %v\n", parsedMessage.Type)
		return -1, errors.New("unexpected ICMP message type")
	}
}