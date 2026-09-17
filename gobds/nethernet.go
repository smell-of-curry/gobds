package gobds

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/df-mc/go-nethernet"
	"github.com/df-mc/go-nethernet/endpoint"
	"github.com/pion/ice/v4"
	"github.com/pion/webrtc/v4"
	"github.com/sandertv/gophertunnel/minecraft"
)

// netherNetConnTimeout bounds ICE + DTLS + SCTP setup for one NetherNet connection.
const netherNetConnTimeout = 20 * time.Second

// netherNetListenOpts is the unexported NetherNet listen stack.
// allowAnonymous is test-only: production config never sets it.
type netherNetListenOpts struct {
	tcpAddr        string
	udpPort        int
	udpIP          net.IP
	publicIP       string
	allowAnonymous bool
	// keyPath persists the server identity key across restarts (stable fingerprint, no repeated TOFU prompt).
	keyPath string
	log     *slog.Logger
}

type netherNetEndpoint struct {
	handler    *endpoint.Handler
	httpServer *http.Server
	tcp        net.Listener
	udp        *net.UDPConn
	api        *webrtc.API
}

func (e *netherNetEndpoint) close() error {
	var errs []error
	if e.httpServer != nil {
		errs = append(errs, e.httpServer.Close())
	}
	if e.udp != nil {
		errs = append(errs, e.udp.Close())
	}
	return errors.Join(errs...)
}

func (e *netherNetEndpoint) udpPort() int {
	if e.udp == nil {
		return 0
	}
	a, ok := e.udp.LocalAddr().(*net.UDPAddr)
	if !ok {
		return 0
	}
	return a.Port
}

type netherNetListener struct {
	listener
	endpoint *netherNetEndpoint
}

func (l netherNetListener) Close() error {
	return errors.Join(l.listener.Close(), l.endpoint.close())
}

func newNetherNetAPI(udp *net.UDPConn, publicIP string) *webrtc.API {
	var se webrtc.SettingEngine
	se.SetICEUDPMux(webrtc.NewICEUDPMux(nil, udp))
	se.SetNetworkTypes([]webrtc.NetworkType{webrtc.NetworkTypeUDP4, webrtc.NetworkTypeUDP6})
	// Public server: no LAN peers to resolve, and pion otherwise opens a
	// multicast :5353 socket per connection.
	se.SetICEMulticastDNSMode(ice.MulticastDNSModeDisabled)
	// Only advertise 127.0.0.1 when the mux itself is bound to loopback (tests);
	// production answers to remote players must not carry a loopback candidate.
	if a, ok := udp.LocalAddr().(*net.UDPAddr); ok && a.IP.IsLoopback() {
		se.SetIncludeLoopbackCandidate(true)
	}
	if publicIP != "" {
		se.SetNAT1To1IPs([]string{publicIP}, webrtc.ICECandidateTypeHost)
	}
	return webrtc.NewAPI(webrtc.WithSettingEngine(se))
}

func newNetherNetStack(opts netherNetListenOpts) (*netherNetEndpoint, nethernet.ListenConfig, error) {
	if opts.log == nil {
		opts.log = slog.Default()
	}
	h := endpoint.HandlerConfig{Logger: opts.log}.New()
	tcp, err := net.Listen("tcp", opts.tcpAddr)
	if err != nil {
		return nil, nethernet.ListenConfig{}, fmt.Errorf("listen nethernet signaling: %w", err)
	}
	udpIP := opts.udpIP
	if udpIP == nil {
		udpIP = net.IPv4zero
	}
	udp, err := net.ListenUDP("udp", &net.UDPAddr{IP: udpIP, Port: opts.udpPort})
	if err != nil {
		_ = tcp.Close()
		return nil, nethernet.ListenConfig{}, fmt.Errorf("listen nethernet ice mux: %w", err)
	}
	api := newNetherNetAPI(udp, opts.publicIP)
	serverKey, err := loadOrCreateServerKey(opts.keyPath)
	if err != nil {
		_ = tcp.Close()
		_ = udp.Close()
		return nil, nethernet.ListenConfig{}, fmt.Errorf("nethernet server key: %w", err)
	}
	httpServer := &http.Server{
		Handler:           h,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       10 * time.Second,
		IdleTimeout:       30 * time.Second,
	}
	go func() {
		if err := httpServer.Serve(tcp); err != nil && !errors.Is(err, http.ErrServerClosed) {
			opts.log.Error("nethernet signaling server", "err", err)
		}
	}()
	return &netherNetEndpoint{
			handler:    h,
			httpServer: httpServer,
			tcp:        tcp,
			udp:        udp,
			api:        api,
		}, nethernet.ListenConfig{
			IssueServerIdentity: serverIdentityIssuer(serverKey),
			DisableTrickleICE:   true,
			// go-nethernet defaults to 5s for ICE+DTLS+SCTP; real players behind
			// NAT hit "start ICE: context deadline exceeded" at that budget.
			ConnContext: func(parent context.Context, _ *nethernet.Conn) (context.Context, context.CancelFunc) {
				return context.WithTimeout(parent, netherNetConnTimeout)
			},
			AllowAnonymous: opts.allowAnonymous,
			API:            api,
			Log:            opts.log,
		}, nil
}

// newNetherNetDialAPI returns the WebRTC API used when dialing an upstream BDS.
// BDS runs on the same host (RemoteAddress 0.0.0.0:<port> → 127.0.0.1), and pion
// drops loopback host candidates by default, so include them for loopback targets
// to keep the proxy↔BDS ICE path off the public interface.
func newNetherNetDialAPI(url string) *webrtc.API {
	var se webrtc.SettingEngine
	se.SetNetworkTypes([]webrtc.NetworkType{webrtc.NetworkTypeUDP4, webrtc.NetworkTypeUDP6})
	if host, _, err := net.SplitHostPort(strings.TrimPrefix(url, "http://")); err == nil {
		if ip := net.ParseIP(host); ip != nil && ip.IsLoopback() {
			se.SetIncludeLoopbackCandidate(true)
		}
	}
	return webrtc.NewAPI(webrtc.WithSettingEngine(se))
}

func netherNetDialAddress(remoteAddress string) string {
	host, port, err := net.SplitHostPort(remoteAddress)
	if err != nil {
		return "http://" + remoteAddress
	}
	switch host {
	case "0.0.0.0", "", "::":
		host = "127.0.0.1"
	}
	return "http://" + net.JoinHostPort(host, port)
}

func (c UserConfig) netherNetListener(srv *Server, cfg minecraft.ListenConfig, nn NetherNetConfig, opts netherNetListenOpts) (Listener, error) {
	opts.tcpAddr = srv.LocalAddress
	opts.udpPort = nn.UDPPort
	opts.publicIP = nn.PublicIP
	opts.log = srv.Log
	if opts.keyPath == "" {
		opts.keyPath = fmt.Sprintf("nethernet_key_%s.pem", strings.ToLower(srv.Name))
	}
	ep, listenCfg, err := newNetherNetStack(opts)
	if err != nil {
		return nil, fmt.Errorf("create nethernet listener: %w", err)
	}
	l, err := cfg.ListenNetwork(minecraft.NetherNet{
		Signaling:    ep.handler,
		ListenConfig: listenCfg,
		Log:          srv.Log,
	}, ep.handler.NetworkID())
	if err != nil {
		_ = ep.close()
		return nil, fmt.Errorf("create nethernet listener: %w", err)
	}
	srv.Log.Info("listening on NetherNet", "tcp", ep.tcp.Addr().String(), "udp", ep.udpPort())
	return netherNetListener{listener: listener{l}, endpoint: ep}, nil
}
