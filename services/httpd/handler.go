package httpd
import (
    "log"
    "os"
    "net/http"
    "github.com/bmizerany/pat"

    "github.com/page31/aws-meta-server/services/aws"
    "net"
    "strings"
)

type route struct {
    name        string
    method      string
    pattern     string
    handlerFunc interface{}
}

type Handler struct {
    mux        *pat.PatternServeMux
    //PushService *push.Service
    routes     []route
    logger     *log.Logger
    Version    string
    AWSService *aws.Service
}


func NewHandler() *Handler {
    h := &Handler{
        mux : pat.New(),
        logger:log.New(os.Stderr, "[HttpHandler]", log.LstdFlags),
        Version: "1.0",
    }
    h.SetRoutes([]route{
        route{"list", "GET", "/ec2/name", h.serveEC2NameFromIP},
    })
    return h
}

func versionHeader(inner http.Handler, h *Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        w.Header().Add("X-Push-Version", h.Version)
        inner.ServeHTTP(w, r)
    })
}

func (h *Handler) SetRoutes(routes []route) {
    for _, r := range routes {
        var handler http.Handler
        if hf, ok := r.handlerFunc.(func(http.ResponseWriter, *http.Request)); ok {
            handler = http.HandlerFunc(hf)
        }
        handler = versionHeader(handler, h)
        h.mux.Add(r.method, r.pattern, handler)
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
    err, name := h.AWSService.EC2NameFromIP(ip)
    if err != nil {
        w.WriteHeader(500)
        w.Write([]byte(err.Error()))
    } else {
        w.WriteHeader(200)
        w.Write([]byte(name))
    }
}