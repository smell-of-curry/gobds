package gobds

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/sandertv/go-raknet"
	"github.com/sandertv/gophertunnel/minecraft"
	"github.com/sandertv/gophertunnel/minecraft/protocol/packet"
	"github.com/sandertv/gophertunnel/minecraft/resource"
	"github.com/smell-of-curry/gobds/gobds/infra"
	"github.com/smell-of-curry/gobds/gobds/interceptor"
	"github.com/smell-of-curry/gobds/gobds/service/claim"
	"github.com/smell-of-curry/gobds/gobds/session"
	"github.com/smell-of-curry/gobds/gobds/session/handlers"
	"github.com/smell-of-curry/gobds/gobds/util/area"
	"github.com/smell-of-curry/gobds/gobds/util/translator"
)

// GoBDS ...
type GoBDS struct {
	conf *Config

	listener *minecraft.Listener

	resources []*resource.Pack

	interceptor *interceptor.Interceptor

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
	gb.setupResources()
	gb.setupServices()
	gb.setupInterceptor()
	gb.setupBorder()

	gb.log.Info("completed initial setup")

	go gb.tick()
}

// setupResources ...
func (gb *GoBDS) setupResources() {
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

// setupServices ...
func (gb *GoBDS) setupServices() {
	infra.ClaimService = claim.NewService(gb.log, gb.conf.Services.ClaimService)
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
		packet.IDLevelChunk:           handlers.LevelChunk{},
		packet.IDPlayerAuthInput:      handlers.PlayerAuthInput{},
		packet.IDRemoveActor:          handlers.RemoveActor{},
		packet.IDSetActorData:         handlers.SetActorData{},
		packet.IDSubChunk:             handlers.SubChunk{},
	} {
		intercept.AddHandler(id, h)
	}
	gb.interceptor = intercept
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
			claims, err := infra.ClaimService.FetchClaims()
			if err != nil {
				gb.log.Error("failed to fetch claims", "err", err)
				continue
			}
			infra.SetClaims(claims)
		}
	}
}

// Start ...
func (gb *GoBDS) Start() error {
	gb.log.Info("starting gobds")

	remoteAddr := gb.conf.Network.RemoteAddress
	_, err := raknet.Ping(remoteAddr)
	if err != nil {
		return err
	}

	prov, err := minecraft.NewForeignStatusProvider(remoteAddr)
	if err != nil {
		return err
	}

	cfg := minecraft.ListenConfig{
		AuthenticationDisabled: false,

		StatusProvider: prov,

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
	defer func(l *minecraft.Listener) {
		gb.log.Info("shutting down gobds")
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
	// TODO: Handle whitelist logic, secured slots logic
	// I would just add an admin config containing the names of exempt players.

	d := minecraft.Dialer{
		ClientData:   conn.ClientData(),
		IdentityData: conn.IdentityData(),

		DownloadResourcePack: func(id uuid.UUID, version string, current, total int) bool { return false },

		FlushRate: time.Millisecond * time.Duration(gb.conf.Network.FlushRate),
		ErrorLog:  gb.log,
	}
	serverConn, err := d.DialTimeout("raknet", gb.conf.Network.RemoteAddress, time.Second*60)
	if err != nil {
		gb.log.Error("error dialing connection", "err", err)
		_ = gb.listener.Disconnect(conn, "error dialing connection")
		return
	}

	gb.startGame(conn, serverConn)
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
	defer gb.listener.Disconnect(client, "connection closed")
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
