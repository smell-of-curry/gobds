package gobds

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/netip"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
	_ "unsafe"

	"github.com/go-jose/go-jose/v4/json"
	"github.com/google/uuid"
	"github.com/sandertv/go-raknet"
	"github.com/sandertv/gophertunnel/minecraft"
	"github.com/sandertv/gophertunnel/minecraft/protocol/packet"
	"github.com/sandertv/gophertunnel/minecraft/resource"
	"github.com/sandertv/gophertunnel/minecraft/text"
	"github.com/smell-of-curry/gobds/gobds/cmd"
	"github.com/smell-of-curry/gobds/gobds/infra"
	"github.com/smell-of-curry/gobds/gobds/interceptor"
	"github.com/smell-of-curry/gobds/gobds/service/authentication"
	"github.com/smell-of-curry/gobds/gobds/service/claim"
	"github.com/smell-of-curry/gobds/gobds/service/vpn"
	"github.com/smell-of-curry/gobds/gobds/session"
	"github.com/smell-of-curry/gobds/gobds/session/handlers"
	"github.com/smell-of-curry/gobds/gobds/util/area"
	"github.com/smell-of-curry/gobds/gobds/util/translator"
	"github.com/smell-of-curry/gobds/gobds/whitelist"

	_ "github.com/smell-of-curry/gobds/gobds/block"
)

// GoBDS ...
type GoBDS struct {
	conf *Config

	provider *minecraft.ForeignStatusProvider
	listener *minecraft.Listener

	resources []*resource.Pack

	interceptor   *interceptor.Interceptor
	playerManager *PlayerManager

	whitelist *whitelist.Whitelist

	ctx    context.Context
	cancel context.CancelFunc

	log *slog.Logger
}

// NewGoBDS ...
func NewGoBDS(conf Config, log *slog.Logger) *GoBDS {
	log.Info("creating a new instance of gobds")

	ctx, cancel := context.WithCancel(context.Background())
	gobds := &GoBDS{
		conf: &conf,

		resources: make([]*resource.Pack, 0),

		ctx:    ctx,
		cancel: cancel,

		log: log,
	}
	gobds.setup()
	return gobds
}

// setup ...
func (gb *GoBDS) setup() {
	world_finaliseBlockRegistry()

	gb.setupResources()
	gb.setupServices()
	gb.setupInterceptor()
	gb.setupWhitelists()
	gb.setupBorder()

	gb.log.Info("completed initial setup")

	go gb.tick()
}

// setupResources ...
func (gb *GoBDS) setupResources() {
	defer gb.setupCommands()

	var packs []*resource.Pack
	for _, url := range gb.conf.Resources.URLResources {
		pack, err := resource.ReadURL(url)
		if err != nil {
			gb.log.Error("failed to load url pack", "err", err)
			continue
		}
		packs = append(packs, pack)
	}
	for _, path := range gb.conf.Resources.PathResources {
		pack, err := resource.ReadPath(path)
		if err != nil {
			gb.log.Error("failed to load path pack", "err", err)
			continue
		}
		packs = append(packs, pack)
	}

	gb.resources = packs
	if len(gb.resources) <= 0 {
		gb.log.Warn("no resources found, skipping translator setup")
		return
	}

	err := translator.Setup(gb.resources[0]) // TODO: Is this behaviour correct?
	if err != nil {
		gb.log.Error("failed to setup translator", "err", err)
	}
}

// setupCommands ...
func (gb *GoBDS) setupCommands() {
	rawBytes, err := os.ReadFile(gb.conf.Resources.CommandPath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			dir := filepath.Dir(gb.conf.Resources.CommandPath)
			if err = os.MkdirAll(dir, os.ModePerm); err != nil {
				panic(err)
			}
			if createErr := os.WriteFile(gb.conf.Resources.CommandPath, []byte("{}"), os.ModePerm); createErr != nil {
				panic(createErr)
			}
			rawBytes, err = os.ReadFile(gb.conf.Resources.CommandPath)
			if err != nil {
				panic(err)
			}
		} else {
			gb.log.Error("failed to read commands", "err", err)
			return
		}
	}

	var commands map[string]cmd.EngineResponseCommand
	if err = json.Unmarshal(rawBytes, &commands); err != nil {
		gb.log.Error("failed to unmarshal commands", "err", err)
		return
	}

	cmd.LoadFrom(commands)
	gb.log.Info("loaded commands", "count", len(commands))
}

// setupServices ...
func (gb *GoBDS) setupServices() {
	infra.AuthenticationService = authentication.NewService(gb.log, gb.conf.AuthenticationService)
	infra.ClaimService = claim.NewService(gb.log, gb.conf.ClaimService)
	infra.VPNService = vpn.NewService(gb.log, gb.conf.VPNService)
}

// setupInterceptor ...
func (gb *GoBDS) setupInterceptor() {
	intercept := interceptor.NewInterceptor()

	for id, h := range map[int]interceptor.Handler{
		packet.IDAddActor:             handlers.AddActor{},
		packet.IDAvailableCommands:    handlers.AvailableCommands{},
		packet.IDCommandRequest:       handlers.CommandRequest{},
		packet.IDInventoryTransaction: handlers.InventoryTransaction{},
		packet.IDItemRegistry:         handlers.ItemRegistry{},
		packet.IDItemStackRequest:     handlers.ItemStackRequest{},
		packet.IDLevelChunk:           handlers.LevelChunk{},
		packet.IDPlayerAuthInput:      handlers.PlayerAuthInput{},
		packet.IDRemoveActor:          handlers.RemoveActor{},
		packet.IDSetActorData:         handlers.SetActorData{},
		packet.IDSubChunk:             handlers.SubChunk{},
		packet.IDText:                 handlers.CustomCommandRegisterHandler{},
	} {
		intercept.AddHandler(id, h)
	}

	// Set the command path for the custom command register handler
	handlers.SetCommandPath(gb.conf.Resources.CommandPath)

	gb.interceptor = intercept
}

// setupWhitelists ...
func (gb *GoBDS) setupWhitelists() {
	conf, err := whitelist.ReadConfig(gb.conf.Network.WhitelistPath)
	if err != nil {
		gb.log.Error("failed to read whitelist config", "err", err)
		return
	}
	gb.whitelist = whitelist.NewWhitelist(conf.Entries)
}

// setupBorder ...
func (gb *GoBDS) setupBorder() {
	b := gb.conf.Border
	if !b.Enabled {
		return // Border is disabled, just return.
	}
	infra.WorldBorder = area.NewArea2D(b.MinX, b.MinZ, b.MaxX, b.MaxZ)
}

// tick ...
func (gb *GoBDS) tick() {
	t := time.NewTicker(time.Minute * 5)
	defer t.Stop()
	for {
		select {
		case <-gb.ctx.Done():
			return
		case <-t.C:
			if infra.ClaimService.Enabled {
				claims, err := infra.ClaimService.FetchClaims()
				if err != nil {
					gb.log.Error("failed to fetch claims", "err", err)
					continue
				}
				infra.SetClaims(claims)
			}
		}
	}
}

// Start ...
func (gb *GoBDS) Start() error {
	gb.log.Info("starting gobds")

	playerManager, err := NewPlayerManager(gb.conf.Network.PlayerManagerPath, gb.log)
	if err != nil {
		return err
	}
	gb.playerManager = playerManager

	remoteAddr := gb.conf.Network.RemoteAddress
	_, err = raknet.Ping(remoteAddr)
	if err != nil {
		return err
	}

	gb.provider, err = minecraft.NewForeignStatusProvider(remoteAddr)
	if err != nil {
		return err
	}

	cfg := minecraft.ListenConfig{
		StatusProvider: gb.provider,

		TexturePacksRequired: gb.conf.Resources.PacksRequired,
		ResourcePacks:        gb.resources,

		FlushRate: time.Millisecond * time.Duration(gb.conf.Network.FlushRate),
		ErrorLog:  gb.log,
	}
	l, err := cfg.Listen("raknet", gb.conf.Network.LocalAddress)
	if err != nil {
		return err
	}
	gb.log.Info("listener established, gobds is now running",
		"on", gb.conf.Network.LocalAddress, "to", gb.conf.Network.RemoteAddress)

	gb.listener = l
	gb.playerManager.Start(time.Minute * 3)
	defer func(l *minecraft.Listener) {
		gb.log.Info("shutting down gobds")
		gb.playerManager.Close()
		_ = l.Close()
	}(gb.listener)

	for {
		select {
		case <-gb.ctx.Done():
			return gb.ctx.Err()
		default:
			conn, err := gb.listener.Accept()
			if err != nil {
				gb.log.Error("error accepting connection", "err", err)
				continue
			}

			go func(conn *minecraft.Conn) {
				gb.log.Info("accepted new connection", "name", conn.IdentityData().DisplayName)
				gb.accept(conn)
			}(conn.(*minecraft.Conn))
		}
	}
}

// accept ...
func (gb *GoBDS) accept(conn *minecraft.Conn) {
	identityData := conn.IdentityData()
	if reason, allowed := gb.handleVPN(conn.LocalAddr()); !allowed {
		_ = gb.listener.Disconnect(conn, reason)
		return
	}
	if infra.AuthenticationService.Enabled {
		response, err := infra.AuthenticationService.AuthenticationOf(identityData.XUID)
		if err != nil {
			disconnectionMessage := err.Error()
			if errors.Is(err, authentication.RecordNotFound) {
				disconnectionMessage = text.Colourf("<red>You must join through the server hub.</red>")
			}
			_ = gb.listener.Disconnect(conn, disconnectionMessage)
			return
		}
		if !response.Allowed {
			_ = gb.listener.Disconnect(conn, "You must join through the server hub to play.")
			return
		}
	}

	identityData = gb.playerManager.IdentityDataOf(conn)
	clientData := gb.playerManager.ClientDataOf(conn)

	displayName := identityData.DisplayName
	if !gb.handleWhitelisted(displayName) {
		_ = gb.listener.Disconnect(conn, "You're not whitelisted.")
		return
	}
	if !gb.handleSecureSlots(displayName) {
		_ = gb.listener.Disconnect(conn, "The server is at full capacity.")
		return
	}

	d := minecraft.Dialer{
		ClientData:   clientData,
		IdentityData: identityData,

		DownloadResourcePack: func(id uuid.UUID, version string, current, total int) bool { return false },

		FlushRate:           time.Millisecond * time.Duration(gb.conf.Network.FlushRate),
		ErrorLog:            gb.log,
		KeepXBLIdentityData: true,
	}
	serverConn, err := d.DialTimeout("raknet", gb.conf.Network.RemoteAddress, time.Second*60)
	if err != nil {
		gb.log.Error("error dialing connection", "err", err)
		_ = gb.listener.Disconnect(conn, "error dialing connection")
		return
	}

	gb.startGame(conn, serverConn)
}

// handleWhitelisted ...
func (gb *GoBDS) handleWhitelisted(displayName string) bool {
	if gb.whitelist == nil || !gb.conf.Network.Whitelisted {
		return true
	}
	return gb.whitelist.Has(displayName)
}

// handleSecureSlots ...
func (gb *GoBDS) handleSecureSlots(displayName string) bool {
	if gb.whitelist == nil {
		// We depend on the whitelist config to handle secured slots.
		return true
	}
	status := gb.provider.ServerStatus(-1, -1)
	current, limit := status.PlayerCount, status.MaxPlayers
	if current >= limit {
		return false
	}

	securedSlots := gb.conf.Network.SecuredSlots
	securedLimit := limit - securedSlots
	if current < securedLimit {
		return true
	}
	return gb.whitelist.Has(displayName)
}

// handleVPN ...
func (gb *GoBDS) handleVPN(netAddr net.Addr) (reason string, allowed bool) {
	addr, _ := netip.ParseAddrPort(netAddr.String())
	addrString := addr.Addr().String()
	if addrString == "127.0.0.1" || addrString == "0.0.0.0" || addrString == "localhost" {
		return "", true
	}

	m, err := infra.VPNService.CheckIP(addrString)
	if err != nil {
		return err.Error(), false
	}
	if m.Status != vpn.StatusSuccess {
		return m.Message, false
	}
	return "VPN/Proxy connections are not allowed.", !m.Proxy
}

// startGame ...
func (gb *GoBDS) startGame(conn *minecraft.Conn, serverConn *minecraft.Conn) {
	gameData := serverConn.GameData()
	gameData.WorldSeed = 0
	gameData.ClientSideGeneration = false

	var failed bool
	var g sync.WaitGroup
	g.Add(2)
	go func() {
		if err := conn.StartGame(gameData); err != nil {
			var disc minecraft.DisconnectError
			if ok := errors.As(err, &disc); !ok {
				gb.log.Error("start game failed", "err", err)
			}
			failed = true
		}
		g.Done()
	}()
	go func() {
		if err := serverConn.DoSpawn(); err != nil {
			var disc minecraft.DisconnectError
			if ok := errors.As(err, &disc); !ok {
				gb.log.Error("spawn failed", "err", err)
			}
			failed = true
		}
		g.Done()
	}()
	g.Wait()

	if failed {
		_ = serverConn.Close()
		_ = gb.listener.Disconnect(conn, "failed to start game")
		return
	}

	s := session.NewSession(conn, serverConn, gb.log)
	s.ForwardXUID(gb.conf.Encryption.Key)
	gb.startListening(s)
}

// startListening ...
func (gb *GoBDS) startListening(s *session.Session) {
	client, server := s.Client(), s.Server()
	go gb.readFrom(s, client, server, client, server, "reading from client", "server")
	go gb.readFrom(s, client, server, server, client, "reading from server", "client")
}

// readFrom ...
func (gb *GoBDS) readFrom(
	s *session.Session,

	client, server *minecraft.Conn,
	src, dst *minecraft.Conn,
	readName, writeName string,
) {
	defer func() {
		if err := gb.listener.Disconnect(client, "connection closed"); err != nil {
			if closedConnectionErr(err) || errors.Is(err, context.Canceled) {
				return
			}
			gb.log.Error("error disconnecting client", "err", err)
		}
	}()
	defer server.Close()

	for {
		select {
		case <-gb.ctx.Done():
			return
		default:
		}

		pkt, err := src.ReadPacket()
		if err != nil {
			if closedConnectionErr(err) || errors.Is(err, context.Canceled) {
				return
			}
			var disc minecraft.DisconnectError
			if errors.As(err, &disc) {
				_ = gb.listener.Disconnect(client, disc.Error())
				return
			}
			gb.log.Error(fmt.Sprintf("error %s", readName), "err", err)
			return
		}

		ctx := session.NewContext()
		gb.interceptor.Intercept(s, pkt, ctx)
		if ctx.Cancelled() {
			continue
		}

		if err = dst.WritePacket(pkt); err != nil {
			if closedConnectionErr(err) || errors.Is(err, context.Canceled) {
				return
			}
			gb.log.Error(fmt.Sprintf("error writing to %s", writeName), "err", err)
			return
		}
	}
}

// closedConnectionErr ...
func closedConnectionErr(err error) bool {
	return strings.Contains(err.Error(), "use of closed network connection")
}

// noinspection ALL
//
//go:linkname world_finaliseBlockRegistry github.com/df-mc/dragonfly/server/world.finaliseBlockRegistry
func world_finaliseBlockRegistry()
