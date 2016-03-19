package middleware

import (
	"github.com/labstack/echo"
	"github.com/nats-io/nats"
	"net"
	"time"
)

type Nats struct {
	NatsConn *nats.Conn
	LocalIP  string
}

func NewNats() *Nats {
	newNats := &Nats{}
	nc, _ := nats.Connect("nats://192.168.0.5:4222", nats.MaxReconnects(60), nats.ReconnectWait(2*time.Second))
	newNats.NatsConn = nc
	newNats.LocalIP = GetLocalIP()
	return newNats
}

func NatsHandler(nats *Nats) echo.MiddlewareFunc {
	return func(next echo.Handler) echo.Handler {
		return echo.HandlerFunc(func(c echo.Context) error {
			uri := "\"uri\":\"" + c.Request().URI() + "\""
			userAgent := "\"userAgent\":\"" + c.Request().Header().Get("user-agent") + "\""
			localIP := "\"localIP\":\"" + nats.LocalIP + "\""
			time := "\"time\":\"" + time.Now().UTC().String() + "\""

			nats.NatsConn.Publish("click", []byte("{"+time+", "+uri+", "+userAgent+", "+localIP+"}"))
			if err := next.Handle(c); err != nil {
				c.Error(err)
			}
			return nil
		})
	}
}

// GetLocalIP returns the non loopback local IP of the host
func GetLocalIP() string {
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		return ""
	}
	for _, address := range addrs {
		// check the address type and if it is not a loopback the display it
		if ipnet, ok := address.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
			if ipnet.IP.To4() != nil {
				return ipnet.IP.String()
			}
		}
	}
	return ""
}
