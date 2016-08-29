package zonewatcher

import (
	"fmt"
	"log"
	"os"
	"os/signal"
	"runtime"
	"syscall"

	"github.com/miekg/dns"
)

func Listen() {
	SigQuit := make(chan os.Signal)
	signal.Notify(SigQuit, syscall.SIGINT, syscall.SIGTERM)
	SigStat := make(chan os.Signal)
	signal.Notify(SigStat, syscall.SIGUSR1)

forever:
	for {
		select {
		case s := <-SigQuit:
			log.Printf("Signal (%d) received, stopping", s)
			break forever
		case _ = <-SigStat:
			log.Printf("Goroutines: %d", runtime.NumGoroutine())
		}
	}
}

type DnsHandler struct {
	State     bool
	Once      bool
	TcpServer *dns.Server
	UdpServer *dns.Server
}

func (d *DnsHandler) NewServer(net string, host string, port string) *dns.Server {
	addr := fmt.Sprintf("%s:%s", host, port)
	server := &dns.Server{
		Addr:    addr,
		Net:     net,
		Handler: d,
	}

	return server
}

func (d *DnsHandler) Serve(s *dns.Server) {

	log.Printf("starting dns %s listener on %s", s.Net, s.Addr)

	err := s.ListenAndServe()
	if err != nil {
		panic(fmt.Sprintf("Failed to set up the %s server: %s", s.Net, err.Error()))
	}
}

func (d *DnsHandler) ServeDNS(writer dns.ResponseWriter, request *dns.Msg) {
	var message *dns.Msg

	switch request.Opcode {
	case dns.OpcodeQuery:
		// In case we want run tests based on the zone nmae
		// zone := request.Question[0].Name

		log.Printf("%#v", request)
		zone := request.Question[0].Name

		if zone == "zone_deleted.com." {
			log.Print("missing zone")
			message = missingZone(request)
			d.State = false
		} else {
			log.Print("zone exists")
			message = zoneExists(request)
			d.State = true
		}
	default:
		log.Printf("ERROR %s : unsupported opcode %d", request.Question[0].Name, request.Opcode)
		message = handleError(request)
	}

	writer.WriteMsg(message)

	if d.Once && d.TcpServer != nil {
		d.TcpServer.Shutdown()
	}

	if d.Once && d.UdpServer != nil {
		d.UdpServer.Shutdown()
	}
}

func missingZone(request *dns.Msg) *dns.Msg {
	log.Print("zone missing")
	message := new(dns.Msg)
	message.SetReply(request)
	message.RecursionDesired = false
	return message
}

func zoneExists(request *dns.Msg) *dns.Msg {
	// SOA Format Reference
	// example.com.    IN    SOA   ns.example.com. hostmaster.example.com. (
	//                           2003080800 ; sn = serial number
	//                           172800     ; ref = refresh = 2d
	//                           900        ; ret = update retry = 15m
	//                           1209600    ; ex = expiry = 2w
	//                           3600       ; nx = nxdomain ttl = 1h
	//                           )

	message := new(dns.Msg)

	message.SetReply(request)

	name := request.Question[0].Name

	record := fmt.Sprintf("%s IN SOA ns.example.com. admin.example.com. 100 21600 3600 1814400 300", name)
	rr, err := dns.NewRR(record)
	if err != nil {
		panic(err)
	}

	message.Answer = append(message.Answer, rr)
	message.RecursionDesired = false

	log.Printf("%#v", message)

	return message
}

func handleError(request *dns.Msg) *dns.Msg {
	message := new(dns.Msg)
	message.SetReply(request)
	message.SetRcode(message, dns.RcodeRefused)

	// Add the question back
	message.Question[0] = request.Question[0]

	// Send an authoritative answer
	message.MsgHdr.Authoritative = true

	return message
}
