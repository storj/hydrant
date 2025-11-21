package process

import (
	"net"
	"os"
	"time"

	"storj.io/hydrant"
	"storj.io/hydrant/value"
)

var (
	processStart = time.Now()
)

// MustRegisterProcessAnnotations is run on the default store at init.
func MustRegisterProcessAnnotations(s *Store) {
	s.MustRegisterAnnotationThunk(AnnotationThunk{
		Key: "proc.uptime",
		Value: func() (value.Value, bool) {
			return value.Duration(time.Since(processStart)), true
		},
	})
}

// MustRegisterOSAnnotations is run on the default store at init.
func MustRegisterOSAnnotations(s *Store) {
	if hostname, err := os.Hostname(); err == nil {
		s.MustRegisterAnnotation(hydrant.String("os.hostname", hostname))
	}

	if outboundIP, ok := getOutboundIP(); ok {
		s.MustRegisterAnnotation(hydrant.String("os.ip", outboundIP.String()))
	}
}

func init() {
	MustRegisterRuntimeAnnotations(DefaultStore)
	MustRegisterOSAnnotations(DefaultStore)
}

func getOutboundIP() (net.IP, bool) {
	conn, err := net.Dial("udp", "8.8.8.8:80")
	if err != nil {
		return net.IP{}, false
	}
	defer conn.Close()
	if addr, ok := conn.LocalAddr().(*net.UDPAddr); ok {
		return addr.IP, true
	}
	return net.IP{}, false
}
