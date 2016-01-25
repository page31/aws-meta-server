package main

import (
    "log"
    "os"
    "os/signal"
    "syscall"
    "math/rand"
    "time"
    "flag"

    "github.com/page31/aws-meta-server/cmd/run"
)

type Main struct {
    Logger       *log.Logger
    Server       *run.Server
    ServerConfig *run.ServerConfig
}

func (m *Main) Run() error {
    rand.Seed(time.Now().UTC().UnixNano())

    m.Logger.Println("Starting service")
    err := m.Server.Open()
    if err != nil {
        m.Logger.Fatalf("Start server failed: %s\n", err.Error())
        return err
    } else {
        m.Logger.Println("Service started")
    }

    signalCh := make(chan os.Signal, 1)
    signal.Notify(signalCh, os.Interrupt, syscall.SIGTERM)
    m.Logger.Println("Listening for signals")

    // Block until one of the signals above is received
    select {
    case <-signalCh:
        m.Logger.Println("Signal received, shuttingdown...")
        go func() {
            m.Close()
        }()
    }
    return nil
}

func (m *Main) Close() {
    m.Server.Close()
}

func main() {
    configFile := flag.String("config", "/etc/aws-meta-server/conf.ini", "config file")
    flag.Parse()
    err, c := run.NewConfig(*configFile)
    m := &Main{
        Logger:log.New(os.Stderr, "[main] ", log.LstdFlags),
    }
    if err != nil {
        m.Logger.Fatalf("Read config file failed %s", err.Error())
        return
    }
    m.ServerConfig = c
    m.Server = run.NewServer(c)
    m.Run()
}
