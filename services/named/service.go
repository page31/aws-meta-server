package named

import (
    "log"
    "os"
    "strings"
    "net"
    "crypto/tls"
    "errors"

    "github.com/miekg/dns"
    "github.com/page31/aws-meta-server/services/aws"
    "math/rand"
    "sync/atomic"
)

var (
    errBadNetType = errors.New("Bad net type")
    seq = rand.Uint32()
)

type Service struct {
    Config     *Config
    logger     *log.Logger
    AWSService *aws.Service
    server     *dns.Server
}

func NewService(c Config) *Service {
    mux := dns.NewServeMux()
    if !strings.HasSuffix(c.Domain, ".") {
        c.Domain += "."
    }
    if !strings.HasSuffix(c.Mbox, ".") {
        c.Mbox += "."
    }
    if !strings.HasSuffix(c.Host, ".") {
        c.Host += "."
    }
    s := &Service{
        Config: &c,
        logger: log.New(os.Stderr, "[named] ", log.LstdFlags),
        server: &dns.Server{Addr: c.Addr, Net: strings.ToLower(c.Net), Handler: mux},
    }
    mux.HandleFunc(c.Domain, s.handle)
    return s
}

func (s *Service) Open() error {
    if err := s.listen(); err != nil {
        return err
    }
    go func() {
        err := s.server.ActivateAndServe()
        if err != nil {
            s.logger.Fatalf("dns serve failed: %s", err.Error())
        }
    }()
    return nil
}

func (s *Service) listen() error {
    srv := s.server
    addr := srv.Addr
    if addr == "" {
        addr = ":domain"
    }
    switch srv.Net {
    case "tcp", "tcp4", "tcp6":
        a, e := net.ResolveTCPAddr(srv.Net, addr)
        if e != nil {
            return e
        }
        l, e := net.ListenTCP(srv.Net, a)
        if e != nil {
            return e
        }
        srv.Listener = l
        return e
    case "tcp-tls", "tcp4-tls", "tcp6-tls":
        network := "tcp"
        if srv.Net == "tcp4-tls" {
            network = "tcp4"
        } else if srv.Net == "tcp6" {
            network = "tcp6"
        }

        l, e := tls.Listen(network, addr, srv.TLSConfig)
        if e != nil {
            return e
        }
        srv.Listener = l
        return e
    case "udp", "udp4", "udp6":
        a, e := net.ResolveUDPAddr(srv.Net, addr)
        if e != nil {
            return e
        }
        l, e := net.ListenUDP(srv.Net, a)
        if e != nil {
            return e
        }
        srv.PacketConn = l
        return e
    }
    return errBadNetType
}

func (s *Service) Close() error {
    return s.server.Shutdown()
}

func (s *Service) handle(w dns.ResponseWriter, r *dns.Msg) {
    reply := new(dns.Msg)
    reply.SetReply(r)
    reply.Authoritative = true
    for _, q := range r.Question {
        answers := s.answer(q)
        if len(answers) > 0 {
            reply.Answer = append(reply.Answer, answers...)
        } else {
            reply.Ns = append(reply.Ns, s.soa(q))
        }
    }
    w.WriteMsg(reply)
}

func (s *Service) answer(q dns.Question) (answers []dns.RR) {
    name := q.Name[0:len(q.Name) - len(s.Config.Domain) - 1]
    instances := s.AWSService.GetEC2FromName(name)
    for _, inst := range instances {
        var target string
        hdr := dns.RR_Header{
            Name: q.Name,
            Class: dns.ClassINET,
            Rrtype: q.Qtype,
            Ttl: inst.Ttl(),
        }
        if inst.PubicDNS != "" {
            hdr.Rrtype = dns.TypeCNAME
            target = inst.PubicDNS
            if !strings.HasSuffix(target, ".") {
                target += "."
            }
        } else if q.Qtype == dns.TypeA {
            target = inst.PublicIP
            if target == "" {
                target = inst.PrivateIP
            }
        }
        if target == "" {
            continue
        }
        if hdr.Rrtype == dns.TypeCNAME {
            answers = append(answers, &dns.CNAME{
                Hdr: hdr,
                Target: target,
            })
        } else if (hdr.Rrtype == dns.TypeA) {
            answers = append(answers, &dns.A{
                Hdr: hdr,
                A: net.ParseIP(target),
            })
        }
    }
    return answers
}

func (s *Service) soa(q dns.Question) dns.RR {
    return &dns.SOA{
        Hdr:     dns.RR_Header{
            Name: s.Config.Domain,
            Rrtype: dns.TypeSOA,
            Class: dns.ClassINET,
            Ttl: 60,
        },
        Ns:      s.Config.Host,
        Mbox:    s.Config.Mbox,
        Serial:  atomic.AddUint32(&seq, 1),
        Refresh: 300,
        Retry:   300,
        Expire:  300,
        Minttl:  60,
    }
}
