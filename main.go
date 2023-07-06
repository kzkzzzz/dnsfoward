package main

import (
	"flag"
	"fmt"
	"github.com/miekg/dns"
	"github.com/spf13/viper"
	"log"
	"net"
	"strings"
)

type (
	Config struct {
		ListenAddr string
		RemoteDns  string
		Record     *Record
	}

	Record struct {
		A  map[string]string
		MX map[string]string
	}
)

var (
	confFile string
	conf     = &Config{}
)

func main() {
	flag.StringVar(&confFile, "conf", "config.yaml", "specify config file")
	flag.Parse()

	vp := viper.NewWithOptions(viper.KeyDelimiter("!"))
	vp.SetConfigFile(confFile)
	err := vp.ReadInConfig()
	if err != nil {
		log.Fatal(err)
	}

	err = vp.Unmarshal(conf)
	if err != nil {
		log.Fatal(err)
	}

	mux := dns.NewServeMux()

	mux.HandleFunc(".", func(writer dns.ResponseWriter, msg *dns.Msg) {
		if len(msg.Question) != 1 {
			return
		}
		ques := msg.Question[0]

		domain := strings.TrimRight(ques.Name, ".")

		log.Printf("query: %s - domain: %s\n", dns.Type(ques.Qtype).String(), domain)

		switch ques.Qtype {
		case dns.TypeA:

			if v, ok := conf.Record.A[domain]; ok && v != "" {
				err = replyA(writer, msg, v)
				if err != nil {
					log.Printf("reply A err: %s\n", err)
				}
				return
			}

		case dns.TypeMX:
			if v, ok := conf.Record.MX[domain]; ok && v != "" {
				err = replyMX(writer, msg, v)
				if err != nil {
					log.Printf("reply MX err: %s\n", err)
				}
				return
			}
		}

		err := forwardRemote(writer, msg)
		if err != nil {
			log.Printf("forward remote err: %s\n", err)
		}

	})

	s := &dns.Server{
		Addr:    conf.ListenAddr,
		Net:     "udp",
		Handler: mux,
	}

	log.Printf("listen on %s", conf.ListenAddr)

	log.Fatal(s.ListenAndServe())

}

func forwardRemote(writer dns.ResponseWriter, msg *dns.Msg) error {
	udpAddr, err := net.ResolveUDPAddr("udp", conf.RemoteDns)
	if err != nil {
		return err
	}
	conn, err := net.DialUDP("udp", nil, udpAddr)
	if err != nil {
		return err
	}

	pack, err := msg.Pack()
	if err != nil {
		return err
	}

	_, err = conn.Write(pack)
	if err != nil {
		return err
	}

	res := make([]byte, 1024)
	n, err := conn.Read(res)
	if err != nil {
		return err
	}

	if n <= 0 {
		return fmt.Errorf("remote server response empty")
	}

	_, err = writer.Write(res[:n])
	if err != nil {
		return err
	}

	return nil
}

func setMsgResp(msg *dns.Msg, rr ...dns.RR) {
	msg.Authoritative = true
	msg.RecursionAvailable = true
	msg.Response = true
	msg.AuthenticatedData = false
	msg.Answer = append(msg.Answer, rr...)
}

func replyA(writer dns.ResponseWriter, msg *dns.Msg, ip string) error {
	rr := &dns.A{
		Hdr: dns.RR_Header{
			Name:   msg.Question[0].Name,
			Rrtype: dns.TypeA,
			Class:  dns.ClassINET,
			Ttl:    60,
		},
		A: net.ParseIP(ip),
	}
	setMsgResp(msg, rr)

	err := writer.WriteMsg(msg)
	if err != nil {
		return err
	}

	return nil
}

func replyMX(writer dns.ResponseWriter, msg *dns.Msg, mxServer string) error {
	rr := &dns.MX{
		Hdr: dns.RR_Header{
			Name:   msg.Question[0].Name,
			Rrtype: dns.TypeMX,
			Class:  dns.ClassINET,
			Ttl:    60,
		},
		Mx:         mxServer + ".",
		Preference: 10,
	}

	setMsgResp(msg, rr)
	err := writer.WriteMsg(msg)
	if err != nil {
		return err
	}

	return nil
}
