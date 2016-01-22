package httpd

import (
    "net"
    "net/http"
    "log"
    "os"
)

type Service struct {
    Config   *Config
    listener *net.Listener
    Handler  *Handler
    logger   *log.Logger
}

func NewService(c Config) *Service {
    s := &Service{
        Config: &c,
        Handler: NewHandler(),
        logger: log.New(os.Stderr, "[httpd] ", log.LstdFlags),
    }
    return s
}

func (s *Service) Open() error {
    listener, err := net.Listen("tcp", s.Config.BindAddress)
    if err != nil {
        s.logger.Fatalf("Bind to %s failed: %s", s.Config.BindAddress, err.Error())
        return err
    }
    s.listener = &listener
    go s.serve()
    if err == nil {
        s.logger.Printf("Service started")
    }
    return err
}

func (s *Service) Close() error {
    return nil
}

func (s *Service) serve() {
    err := http.Serve(*s.listener, s.Handler)
    if err != nil {
        s.logger.Fatalf("Serve failed: %s", err.Error())
    }
}
