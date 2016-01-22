package aws
import (
    "errors"
    "os"
    "log"

    "github.com/aws/aws-sdk-go/aws"
    "github.com/aws/aws-sdk-go/service/ec2"
    "github.com/aws/aws-sdk-go/aws/credentials"
    "github.com/aws/aws-sdk-go/aws/session"
)

type Service struct {
    Config       *Config
    logger       *log.Logger
    awsConfig    *aws.Config
    ec2Instances []*EC2Instance
}

type EC2Instance struct {
    Address    string
    PublicIP   string
    PrivateIP  string
    PubicDNS   string
    PrivateDNS string
    Name       string
}

var (
    notFoundError = errors.New("no matching resource found")
)

func NewService(c Config) *Service {
    awsConfig := aws.NewConfig().WithCredentials(
        credentials.NewStaticCredentials(c.AccessKeyID, c.SecretAccessKey, "")).WithRegion(c.Region)

    s := &Service{
        Config: &c,
        logger: log.New(os.Stderr, "[aws] ", log.LstdFlags),
        awsConfig: awsConfig,
    }
    return s
}

func (s *Service) Open() error {
    err := s.UpdateCache()
    return err;
}

func (s *Service) Close() error {
    return nil;
}

func (s *Service) GetAllEC2Names() []string {
    names := make([]string, 0, len(s.ec2Instances));
    s.eachEC2Instance(func(idx int, instance *EC2Instance) bool {
        names = append(names, instance.Name)
        return true
    })
    return names
}

func (s *Service) GetEC2FromName(name string, output *EC2Instance) error {
    instance := s.findEC2Instance(func(inst *EC2Instance) bool {
        return inst.Name == name
    })
    if instance == nil {
        return notFoundError
    } else if output != nil {
        *output = *instance
    }
    return nil
}

func (s *Service) GetEC2NameFromIP(ip string) (error, string) {
    instance := s.findEC2Instance(func(ec2 *EC2Instance) bool {
        return ip != "" && (ec2.PrivateIP == ip || ec2.PublicIP == ip)
    })
    if instance == nil {
        return notFoundError, ""
    } else {
        return nil, instance.Name
    }
}

func (s *Service) UpdateCache() error {
    ec2 := ec2.New(session.New(s.awsConfig))
    resp, err := ec2.DescribeInstances(nil)
    if err != nil {
        return err;
    }
    instances := make([]*EC2Instance, 0, 10)
    for _, rev := range resp.Reservations {
        for _, inst := range rev.Instances {
            ec2 := newEC2(inst)
            instances = append(instances, ec2)
        }
    }
    s.ec2Instances = instances
    s.logger.Printf("got %d ec2 instances", len(instances))
    return nil
}

func newEC2(inst *ec2.Instance) *EC2Instance {
    ec2 := &EC2Instance{}
    for _, tag := range inst.Tags {
        if tag.Key != nil && *tag.Key == "Name" {
            ec2.Name = *tag.Value
            break
        }
    }
    if inst.PublicDnsName != nil {
        ec2.PubicDNS = *inst.PublicDnsName
        ec2.Address = ec2.PubicDNS
    }
    if inst.PublicIpAddress != nil {
        ec2.PublicIP = *inst.PublicIpAddress
    }
    if inst.PrivateDnsName != nil {
        ec2.PrivateDNS = *inst.PrivateDnsName
        if ec2.Address == "" {
            ec2.Address = ec2.PrivateDNS
        }
    }
    if inst.PrivateIpAddress != nil {
        ec2.PrivateIP = *inst.PrivateIpAddress
    }
    return ec2
}

func (s *Service) findEC2Instances(filterFunc func(*EC2Instance) bool, limit int) []*EC2Instance {
    instances := s.ec2Instances
    filtered := make([]*EC2Instance, 0, limit)
    for _, inst := range instances {
        if filterFunc(inst) {
            filtered = append(filtered, inst)
        }
        if limit > 0 && len(filtered) == limit {
            break
        }
    }
    return filtered
}

func (s *Service) findEC2Instance(filterFunc func(*EC2Instance) bool) *EC2Instance {
    instances := s.findEC2Instances(filterFunc, 1)
    if len(instances) == 0 {
        return nil
    } else {
        return instances[0]
    }
}

func (s *Service) eachEC2Instance(iterFunc func(index int, instance *EC2Instance) bool) int {
    loopCount := 0
    for idx, inst := range s.ec2Instances {
        loopCount += 1
        if !iterFunc(idx, inst) {
            break
        }
    }
    return loopCount
}
