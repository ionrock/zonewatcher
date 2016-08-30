package zonewatcher

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"runtime"
	// "strings"
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

type DnsBackend interface {
	Add(string)
	Del(string)
	Find(string) bool
}

func NewFileDnsBackend(dirname string) *FileDnsBackend {
	_, err := os.Stat(dirname)
	if err != nil {
		err = os.Mkdir(dirname, 0755)
		if err != nil {
			panic(err)
		}
	}

	return &FileDnsBackend{dirname: dirname}
}

type FileDnsBackend struct {
	dirname string
}

func (b FileDnsBackend) Fname(zone string) string {
	base, _ := filepath.Abs(b.dirname)
	return filepath.Join(base, zone)
}

func (b FileDnsBackend) Find(zone string) bool {
	_, err := os.Stat(b.Fname(zone))
	if err != nil {
		log.Print(err)
		return false
	}
	return true
}

func (b FileDnsBackend) Add(zone string) {
	log.Print("Add zone: ", zone, b.Fname(zone))
	if b.Find(zone) {
		return
	}

	err := ioutil.WriteFile(b.Fname(zone), []byte("true"), 0644)
	if err != nil {
		panic(err)
	}

	log.Print("Added zone: ", zone, b.Find(zone))

}

func (b FileDnsBackend) Del(zone string) {

	log.Print("Del zone: ", zone, b.Fname(zone))
	if b.Find(zone) == false {
		return
	}

	err := os.Remove(b.Fname(zone))
	if err != nil {
		log.Print(err)
	}
}

type DnsHandler struct {
	State   bool
	Servers []*dns.Server
	Zones   DnsBackend
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

	d.Servers = append(d.Servers, s)
}

func (d *DnsHandler) Shutdown() {
	for _, s := range d.Servers {
		err := s.Shutdown()
		if err != nil {
			panic(err)
		}
	}
}

func (d *DnsHandler) ServeDNS(writer dns.ResponseWriter, request *dns.Msg) {
	var message *dns.Msg

	switch request.Opcode {
	case dns.OpcodeQuery:
		// In case we want run tests based on the zone nmae
		// zone := request.Question[0].Name

		zone := request.Question[0].Name

		exists := d.Zones.Find(zone)

		if exists {
			log.Print("exists: ", zone)
			message = zoneExists(request)
			d.State = true
		} else {
			log.Print("missing: ", zone)
			message = missingZone(request)
			d.State = false
		}
	default:
		log.Printf("ERROR %s : unsupported opcode %d", request.Question[0].Name, request.Opcode)
		message = handleError(request)
	}

	writer.WriteMsg(message)
}

func missingZone(request *dns.Msg) *dns.Msg {
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
