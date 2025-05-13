package gobds

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
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
	"github.com/smell-of-curry/gobds/gobds/service/identity"
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
	infra.IdentityService = identity.NewService(gb.log, gb.conf.Services.IdentityService)
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
	infra.WorldBorder = area.NewArea2D(b.MaxX, b.MaxZ, b.MinX, b.MinZ)
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
		AuthenticationDisabled: true,

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
	gb.log.Info("listener established")

	gb.listener = l
	defer gb.listener.Close()

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
				gb.log.Info("accepted new connection")
				gb.accept(conn)
			}(conn.(*minecraft.Conn))
		}
	}
}

// accept ...
func (gb *GoBDS) accept(conn *minecraft.Conn) {
	go gb.closeOnceDone(conn)

	// TODO: Handle whitelist logic, secured slots logic
	// I would just add an admin config containing the names of exempt players.

	clientData, identityData := conn.ClientData(), conn.IdentityData()

	response, err := infra.IdentityService.IdentityOf(clientData.ThirdPartyName)
	if err != nil {
		_ = gb.listener.Disconnect(conn, fmt.Sprintf("identity service: %v", err))
		return
	}

	identityData.XUID = response.XUID
	uid := uuid.NewSHA1(uuid.NameSpaceDNS, []byte(identityData.XUID))
	identityData.Identity = uid.String()
	err = identityData.Validate()
	if err != nil {
		_ = gb.listener.Disconnect(conn, fmt.Sprintf("failed to validate identity data: %v", err))
		return
	}

	d := minecraft.Dialer{
		KeepXBLIdentityData: true,

		ClientData:   clientData,
		IdentityData: identityData,

		DownloadResourcePack: func(id uuid.UUID, version string, current, total int) bool {
			return false
		},

		FlushRate: time.Millisecond * time.Duration(gb.conf.Network.FlushRate),

		ErrorLog: gb.log,
	}
	serverConn, err := d.DialTimeout("raknet", gb.conf.Network.RemoteAddress, time.Second*60)
	if err != nil {
		gb.log.Error("error dialing connection", "err", err)
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
	go func() {
		for {
			select {
			case <-gb.ctx.Done():
				return
			default:
				pk, err := client.ReadPacket()
				if err != nil {
					gb.log.Error("error reading client packet", "err", err)
					return
				}

				ctx := session.NewContext()
				gb.interceptor.Intercept(s, pk, ctx)
				if ctx.Cancelled() {
					continue
				}

				if err = server.WritePacket(pk); err != nil {
					gb.log.Error("error writing to server", "err", err)
					return
				}
			}
		}
	}()
	go func() {
		for {
			select {
			case <-gb.ctx.Done():
				return
			default:
				pk, err := server.ReadPacket()
				if err != nil {
					gb.log.Error("error reading server packet", "err", err)
					return
				}

				ctx := session.NewContext()
				gb.interceptor.Intercept(s, pk, ctx)
				if ctx.Cancelled() {
					continue
				}

				if err = client.WritePacket(pk); err != nil {
					gb.log.Error("error writing to client", "err", err)
					return
				}
			}
		}
	}()
}

// closeOnceDone ...
func (gb *GoBDS) closeOnceDone(conn *minecraft.Conn) {
	<-gb.ctx.Done()
	if conn != nil {
		_ = conn.Close()
	}
}
