package run
import (
    "github.com/page31/aws-meta-server/services/aws"
    "github.com/page31/aws-meta-server/services/httpd"
    "github.com/page31/aws-meta-server/services/named"
    "gopkg.in/gcfg.v1"
)

type ServerConfig struct {
    AWS  aws.Config
    HTTP httpd.Config
    DNS  named.Config
}

func NewConfig(file string) (error, *ServerConfig) {
    c := &ServerConfig{}
    err := c.init(file)
    return err, c
}

func (c *ServerConfig) init(file string) error {
    return gcfg.ReadFileInto(c, file)
}
