package httpd
import (
    "log"
    "os"
    "net/http"
    "github.com/bmizerany/pat"

    "github.com/page31/aws-meta-server/services/aws"
    "net"
    "strings"
    "reflect"
)

type route struct {
    name        string
    method      string
    pattern     string
    handlerFunc interface{}
}

type Handler struct {
    mux        *pat.PatternServeMux
    routes     []route
    logger     *log.Logger
    Version    string
    AWSService *aws.Service
}

type HTTPHandler func(http.ResponseWriter, *http.Request)

func NewHandler() *Handler {
    h := &Handler{
        mux : pat.New(),
        logger:log.New(os.Stderr, "[HttpHandler]", log.LstdFlags),
        Version: "1.0",
    }
    h.SetRoutes([]route{
        route{"IP2EC2name", "GET", "/ec2/name", h.serveEC2NameFromIP},
        route{"Names", "GET", "/ec2/names", h.serveEC2Names},
        route{"Update", "GET", "/update", h.serveUpdate},
        route{"Name2EC2IP", "GET", "/ec2/ip", h.serveEC2IPFromName},
    })
    return h
}

func (h *Handler) SetRoutes(routes []route) {
    for _, r := range routes {
        var handler http.Handler
        if hf, ok := r.handlerFunc.(func(http.ResponseWriter, *http.Request)); ok {
            handler = http.HandlerFunc(hf)
        } else if hf, ok := r.handlerFunc.(func() error); ok {
            handler = http.HandlerFunc(wrapNoContentHandler(hf))
        } else {
            handler = http.HandlerFunc(wrapBindHandler(r.handlerFunc))
        }
        handler = versionHeader(handler, h)
        h.mux.Add(r.method, r.pattern, handler)
    }
}

func versionHeader(inner http.Handler, h *Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        w.Header().Add("X-AWS-META-SERVER", h.Version)
        inner.ServeHTTP(w, r)
    })
}

func wrapNoContentHandler(inner func() error) HTTPHandler {
    return func(w http.ResponseWriter, r *http.Request) {
        err := inner()
        if err != nil {
            writeError(w, err)
        } else {
            writeOK(w)
        }
    }
}

func wrapBindHandler(inner interface{}) HTTPHandler {
    methodType := reflect.TypeOf(inner)
    methodValue := reflect.ValueOf(inner)
    pType := methodType.In(1)
    return func(w http.ResponseWriter, r *http.Request) {
        p := reflect.New(pType);
        err := BindQuery(r.URL.Query(), pType, p.Elem())
        if err != nil {
            w.WriteHeader(400)
            w.Write([]byte(err.Error() + "\n"))
        } else {
            in := []reflect.Value{reflect.ValueOf(w), p.Elem()}
            methodValue.Call(in)
        }
    }
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
    h.mux.ServeHTTP(w, r)
}

func (h *Handler) serveEC2NameFromIP(w http.ResponseWriter, r *http.Request) {
    ip := r.URL.Query().Get("ip")
    if ip == "" {
        ip = strings.TrimSpace(strings.Split(r.Header.Get("X-Forwarded-For"), ",")[0])
    }
    if ip == "" {
        clientIp, _, err := net.SplitHostPort(r.RemoteAddr)
        if err == nil {
            ip = clientIp
        }
    }
    err, name := h.AWSService.GetEC2NameFromIP(ip)
    if err != nil {
        writeError(w, err)
    } else {
        writeString(w, name)
    }
}

func (h *Handler) serveEC2Names(w http.ResponseWriter, r *http.Request) {
    w.WriteHeader(200)
    w.Write([]byte(strings.Join(h.AWSService.GetAllEC2Names(), "\n")))
}

type ec2IPFromNameRequest struct {
    Name    string `bind:"name" required:"true"`
    Public  bool `bind:"public"`
    Private bool `bind:"private" default:"true"`
}

func (h *Handler) serveEC2IPFromName(w http.ResponseWriter, request ec2IPFromNameRequest) {
    ec2 := aws.EC2Instance{}
    err := h.AWSService.GetEC2FromName(request.Name, &ec2)
    if err != nil {
        writeError(w, err)
    } else {
        var value string
        if request.Private {
            value = ec2.PrivateIP
        }
        if request.Public {
            value += "|" + ec2.PublicIP
        }
        writeString(w, value)
    }
}

func (h *Handler) serveUpdate() error {
    return h.AWSService.UpdateCache()
}

func writeError(w http.ResponseWriter, err error) {
    w.WriteHeader(500)
    w.Write([]byte(err.Error() + "\n"))
}

func writeOK(w http.ResponseWriter) {
    w.WriteHeader(200)
    w.Write([]byte("OK\n"))
}

func writeString(w http.ResponseWriter, content string) {
    w.WriteHeader(200)
    w.Write([]byte(content))
}
