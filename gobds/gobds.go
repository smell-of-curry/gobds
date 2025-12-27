package gobds

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/netip"
	"os"
	"os/signal"
	"sync"
	"sync/atomic"
	"syscall"
	"time"
	_ "unsafe"

	"github.com/sandertv/gophertunnel/minecraft"
	"github.com/sandertv/gophertunnel/minecraft/text"
	_ "github.com/smell-of-curry/gobds/gobds/block"
	"github.com/smell-of-curry/gobds/gobds/infra"
	"github.com/smell-of-curry/gobds/gobds/service/authentication"
	"github.com/smell-of-curry/gobds/gobds/service/vpn"
	"github.com/smell-of-curry/gobds/gobds/session"
)

// GoBDS ...
type GoBDS struct {
	conf *Config

	ctx    context.Context
	cancel context.CancelFunc

	started atomic.Pointer[time.Time]

	servers []*Server

	wg sync.WaitGroup
}

// New creates new GoBDS instance.
func (c *Config) New() (*GoBDS, error) {
	c.Log.Info("creating a new instance of gobds")
	ctx, cancel := context.WithCancel(context.Background())
	world_finaliseBlockRegistry()

	gobds := &GoBDS{
		conf:   c,
		ctx:    ctx,
		cancel: cancel,
	}

	if c.PlayerManager == nil {
		return nil, fmt.Errorf("start gobds: player manager is not configured")
	}
	if len(c.Servers) == 0 {
		return nil, fmt.Errorf("start gobds: no servers configured")
	}
	gobds.servers = c.Servers

	go gobds.tick()
	return gobds, nil
}

// tick ...
func (gb *GoBDS) tick() {
	fetch := func() {
		if gb.conf.ClaimService != nil {
			claims, err := gb.conf.ClaimService.FetchClaims()
			if err != nil {
				gb.conf.Log.Error("failed to fetch claims", "err", err)
				return
			}
			infra.SetClaims(claims)
		}
	}
	fetch()

	t := time.NewTicker(time.Minute * 5)
	defer t.Stop()
	for {
		select {
		case <-gb.ctx.Done():
			return
		case <-t.C:
			fetch()
		}
	}
}

// Listen ...
func (gb *GoBDS) Listen() error {
	t := time.Now()
	if !gb.started.CompareAndSwap(nil, &t) {
		return fmt.Errorf("start gobds: already started")
	}

	gb.conf.Log.Info("starting gobds")
	gb.conf.PlayerManager.Start(time.Minute * 3)

	gb.wg.Add(len(gb.servers))
	for _, srv := range gb.servers {
		go gb.listen(srv)
	}

	gb.wg.Wait()
	gb.conf.Log.Info("proxy closed.", "uptime", time.Since(*gb.started.Load()).String())

	return nil
}

// listen handles a server and its sessions.
func (gb *GoBDS) listen(srv *Server) {
	wg := new(sync.WaitGroup)
	ctx, cancel := context.WithCancel(context.Background())
	defer func() { _ = srv.Listener.Close() }()

	go func() {
		<-gb.ctx.Done()
		cancel()
		wg.Wait()
		_ = srv.Listener.Close()
	}()

	for {
		conn, err := srv.Listener.Accept()
		if err != nil {
			gb.wg.Done()
			return
		}

		wg.Add(1)
		go func() {
			defer wg.Done()

			s, err := gb.accept(conn, srv, ctx)
			if err != nil {
				_ = srv.Listener.Disconnect(conn, err.Error())
				return
			}

			srv.Log.Info("player connected", "name", conn.IdentityData().DisplayName)
			s.ReadPackets(ctx)
			srv.Log.Info("player disconnected", "name", conn.IdentityData().DisplayName)
		}()
	}
}

// accept accepts new connection.
func (gb *GoBDS) accept(conn session.Conn, srv *Server, ctx context.Context) (*session.Session, error) {
	identityData := conn.IdentityData()
	if gb.conf.VPNService != nil {
		if reason, allowed := gb.handleVPN(conn.LocalAddr(), ctx); !allowed {
			return nil, errors.New(reason)
		}
	}
	if gb.conf.AuthenticationService != nil {
		response, err := gb.conf.AuthenticationService.AuthenticationOf(identityData.XUID, ctx)
		if err != nil {
			disconnectionMessage := err.Error()
			if errors.Is(err, authentication.ErrRecordNotFound) {
				disconnectionMessage = text.Colourf("<red>you must join through the server hub.</red>")
			}
			return nil, errors.New(disconnectionMessage)
		}
		if !response.Allowed {
			return nil, fmt.Errorf("you must join through the server hub to play")
		}
	}

	identityData = gb.conf.PlayerManager.IdentityDataOf(conn)
	clientData := gb.conf.PlayerManager.ClientDataOf(conn)

	displayName := identityData.DisplayName
	if !gb.handleWhitelisted(displayName) {
		return nil, fmt.Errorf("you're not whitelisted")
	}
	if auth := gb.conf.AuthenticationService; auth != nil && !auth.Enabled {
		if !gb.handleSecureSlots(srv, displayName) {
			return nil, fmt.Errorf("the server is at full capacity")
		}
	}

	ctx2, cancel := context.WithTimeout(ctx, time.Minute)

	defer cancel()
	serverConn, err := srv.DialerFunc(identityData, clientData, ctx2)
	if err != nil {
		srv.Log.Error("error dialing connection", "err", err)
		return nil, fmt.Errorf("error dialing connection")
	}

	return gb.startGame(conn, serverConn, ctx)
}

// handleWhitelisted ensures that only whitelisted players can join.
func (gb *GoBDS) handleWhitelisted(displayName string) bool {
	if gb.conf.Whitelist == nil {
		return true
	}
	return gb.conf.Whitelist.Has(displayName)
}

// handleSecureSlots secures slots for some whitelisted players.
func (gb *GoBDS) handleSecureSlots(srv *Server, displayName string) bool {
	if gb.conf.Whitelist == nil {
		// We depend on the whitelist config to handle secured slots.
		return true
	}

	status := srv.StatusProvider.ServerStatus(-1, -1)
	current, limit := status.PlayerCount, status.MaxPlayers
	if current >= limit {
		return false
	}

	securedSlots := gb.conf.SecuredSlots
	securedLimit := limit - securedSlots
	if current < securedLimit {
		return true
	}
	return gb.conf.Whitelist.Has(displayName)
}

// handleVPN protects proxy from vpn/proxy users.
func (gb *GoBDS) handleVPN(netAddr net.Addr, ctx context.Context) (reason string, allowed bool) {
	addr, _ := netip.ParseAddrPort(netAddr.String())
	addrString := addr.Addr().String()
	if addrString == "127.0.0.1" || addrString == "0.0.0.0" || addrString == "localhost" {
		return "", true
	}

	m, err := gb.conf.VPNService.CheckIP(addrString, ctx)
	if err != nil {
		return err.Error(), false
	}
	if m.Status != vpn.StatusSuccess {
		return m.Message, false
	}
	return "VPN/Proxy connections are not allowed.", !m.Proxy
}

// startGame starts game for new connection.
func (gb *GoBDS) startGame(conn, serverConn session.Conn, ctx context.Context) (*session.Session, error) {
	gameData := serverConn.GameData()
	gameData.WorldSeed = 0
	gameData.ClientSideGeneration = false

	var failed bool
	var g sync.WaitGroup
	g.Add(2)
	go func() {
		if err := conn.StartGameContext(ctx, gameData); err != nil {
			var disc minecraft.DisconnectError
			if ok := errors.As(err, &disc); !ok {
				gb.conf.Log.Error("start game failed", "err", err)
			}
			failed = true
		}
		g.Done()
	}()
	go func() {
		if err := serverConn.DoSpawnContext(ctx); err != nil {
			var disc minecraft.DisconnectError
			if ok := errors.As(err, &disc); !ok {
				gb.conf.Log.Error("spawn failed", "err", err)
			}
			failed = true
		}
		g.Done()
	}()
	g.Wait()

	if failed {
		_ = serverConn.Close()
		return nil, fmt.Errorf("failed to start game")
	}

	s := session.Config{
		Client:        conn,
		Server:        serverConn,
		PingIndicator: gb.conf.PingIndicator,
		AfkTimer:      gb.conf.AFKTimer,
		Border:        gb.conf.Border,
		Log:           gb.conf.Log,
	}.New()

	s.ForwardXUID(gb.conf.EncryptionKey)
	return s, nil
}

// Close closes all listeners.
func (gb *GoBDS) Close() error {
	gb.cancel()
	return nil
}

// CloseOnProgramEnd ...
func (gb *GoBDS) CloseOnProgramEnd() {
	c := make(chan os.Signal, 2)
	signal.Notify(c, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-c
		_ = gb.Close()
	}()
}

// noinspection ALL
//
//go:linkname world_finaliseBlockRegistry github.com/df-mc/dragonfly/server/world.finaliseBlockRegistry
func world_finaliseBlockRegistry()
