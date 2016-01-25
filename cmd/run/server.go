package run

import (
    "errors"
    "github.com/page31/aws-meta-server/services/aws"
    "github.com/page31/aws-meta-server/services/named"
    "github.com/page31/aws-meta-server/services/httpd"
)

type Server struct {
    Config      ServerConfig
    Services    []Service
    awsService  *aws.Service
    dnsService  *named.Service
    httpService *httpd.Service
}

type Service interface {
    Open() error
    Close() error
}


func NewServer(c *ServerConfig) *Server {
    s := &Server{
        Config: *c,
    }
    s.appendAWSService(c.AWS)
    s.appendDNSService(c.DNS)
    s.appendHTTPService(c.HTTP)
    return s
}

func (s *Server) Open() error {
    var service Service
    for _, service = range s.Services {
        if err := service.Open(); err != nil {
            return err
        }
    }
    return nil
}

func (s *Server) Close() error {
    var service Service
    success := true
    for _, service = range s.Services {
        if err := service.Close(); err != nil {
            success = false
        }
    }
    if success {
        return nil
    } else {
        return errors.New("some services failed to close")
    }
}


func (s *Server) appendAWSService(c aws.Config) {
    s.awsService = aws.NewService(c)
    s.Services = append(s.Services, s.awsService)
}

func (s *Server) appendDNSService(c named.Config) {
    if c.Enabled {
        s.dnsService = named.NewService(c)
        s.dnsService.AWSService = s.awsService
        s.Services = append(s.Services, s.dnsService)
    }
}

func (s *Server) appendHTTPService(c httpd.Config) {
    if c.Enabled {
        s.httpService = httpd.NewService(c)
        s.httpService.Handler.AWSService = s.awsService
        s.Services = append(s.Services, s.httpService)
    }
}
