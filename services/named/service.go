package named

import (
    "log"
    "os"
    "github.com/page31/aws-meta-server/services/aws"
)

type Service struct {
    Config     *Config
    logger     *log.Logger
    AWSService *aws.Service
}

func NewService(c Config) *Service {
    s := &Service{
        Config: &c,
        logger: log.New(os.Stderr, "[named] ", log.LstdFlags),
    }
    return s
}

func (s *Service) Open() error {
    return nil;
}

func (s *Service) Close() error {
    return nil;
}

