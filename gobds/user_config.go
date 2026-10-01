package gobds

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/df-mc/go-nethernet"
	"github.com/df-mc/go-nethernet/endpoint"
	"github.com/google/uuid"
	"github.com/restartfu/gophig"
	"github.com/sandertv/gophertunnel/minecraft"
	"github.com/sandertv/gophertunnel/minecraft/protocol/login"
	"github.com/sandertv/gophertunnel/minecraft/resource"
	"github.com/smell-of-curry/gobds/gobds/claim"
	"github.com/smell-of-curry/gobds/gobds/cmd"
	"github.com/smell-of-curry/gobds/gobds/infra"
	"github.com/smell-of-curry/gobds/gobds/session"
	"github.com/smell-of-curry/gobds/gobds/util/area"
	"github.com/smell-of-curry/gobds/gobds/util/translator"
	"github.com/smell-of-curry/gobds/gobds/whitelist"
)

// UserConfig ...
type UserConfig struct {
	Network struct {
		ServerRegion string

		Servers []ServerConfig

		Whitelisted   bool
		WhitelistPath string

		SecuredSlots      int
		MaxRenderDistance int
		FlushRate         int

		SentryDSN string

		// TurnAddress is host:port of the coturn server used for NetherNet ICE
		// relay candidates (empty = no TURN). TurnSecret is its static-auth-secret.
		TurnAddress string
		TurnSecret  string
	}
	Border struct {
		Enabled    bool
		MinX, MinZ int32
		MaxX, MaxZ int32
	}
	Claims struct {
		PrefilterEnabled     bool
		DenyRenderingEnabled bool
		PollInterval         string
		MaxSnapshotAge       string
	}
	AFKTimer struct {
		Enabled         bool
		TimeoutDuration string
		// WarnApproaching is how long a player must be idle before receiving
		// an "AFK in 1 minute" soft warning. Always sent regardless of fullness.
		WarnApproaching string
		// MarkAFK is how long a player must be idle before being told they
		// are now AFK. Always sent regardless of fullness.
		MarkAFK string
		// FinalWarning is how long a player must be idle before receiving the
		// near-capacity hard warning. Only sent when fullness >= FullnessThreshold.
		FinalWarning string
		// FullnessThreshold is the fraction (0..1) of MaxPlayers at or above
		// which the proxy will start kicking AFK players, longest-AFK first.
		FullnessThreshold float64
	}
	Resources struct {
		PacksRequired bool

		CommandPath string
		// URLResources and PathResources are legacy and unfiltered. They are
		// loaded after Packs, so they sit below Packs in priority.
		URLResources  []string
		PathResources []string
		Packs         []PackConfig `toml:"Packs"`
	}
	AuthenticationService struct {
		Enabled bool
		URL     string
		Key     string
	}
	VPNService struct {
		Enabled bool
		URL     string
		Key     string
		// WhitelistedCIDRs are IP ranges (e.g. "45.230.64.0/22") never
		// treated as VPN/proxy connections. Used for residential ISP
		// blocks the detection API misclassifies.
		WhitelistedCIDRs []string
	}
	TrafficProtection session.TrafficConfig
	DuplicateXUID     struct {
		Enabled bool
	}
	Encryption struct {
		Key string
	}
}

// PackConfig is one Resources.Packs entry. Exactly one of URL or Path is set.
// MinMemoryTier 0 and an empty Language send the pack to every player.
type PackConfig struct {
	URL           string `toml:"URL"`
	Path          string `toml:"Path"`
	MinMemoryTier int    `toml:"MinMemoryTier"`
	Language      string `toml:"Language"`
}

// packFilter is the per-player predicate for one loaded pack. Zero value
// matches every client (legacy URLResources / PathResources).
type packFilter struct {
	MinMemoryTier int
	Language      string
}

// packOffer is a loaded pack plus the filter applied before it is sent.
// offers from UserConfig.packs are highest-priority first: Resources.Packs
// (index 0 highest), then unfiltered legacy URLResources and PathResources.
type packOffer struct {
	id     string
	pack   *resource.Pack
	filter packFilter
	legacy bool
}

// packs loads Packs, then legacy URLResources, then PathResources.
// A pack that fails to download or open is logged and skipped.
func (c UserConfig) packs(log *slog.Logger) []packOffer {
	offers := make([]packOffer, 0, len(c.Resources.Packs)+len(c.Resources.URLResources)+len(c.Resources.PathResources))

	for i, pc := range c.Resources.Packs {
		pack, err := readConfiguredPack(pc)
		if err != nil {
			log.Error("failed to load pack", "err", err)
			continue
		}
		offers = append(offers, packOffer{
			id:   fmt.Sprintf("pack-%d", i),
			pack: pack,
			filter: packFilter{
				MinMemoryTier: pc.MinMemoryTier,
				Language:      pc.Language,
			},
		})
	}
	for _, url := range c.Resources.URLResources {
		pack, err := readURLPack(url)
		if err != nil {
			log.Error("failed to load url pack", "err", err)
			continue
		}
		offers = append(offers, packOffer{pack: pack, legacy: true})
	}
	for _, path := range c.Resources.PathResources {
		pack, err := resource.ReadPath(path)
		if err != nil {
			log.Error("failed to load path pack", "err", err)
			continue
		}
		offers = append(offers, packOffer{pack: pack, legacy: true})
	}

	// Translator merge is later-wins. Walk stack order (lowest priority
	// first) so Packs[0], the highest priority pack, overwrites keys last.
	for _, offer := range stackOrder(offers) {
		if err := translator.Setup(offer.pack); err != nil {
			log.Error("failed to setup translator", "err", err)
		}
	}
	return offers
}

// readConfiguredPack loads one Resources.Packs entry. URL goes through
// readURLPack so the pack is served over RakNet chunks, not as a CDN
// DownloadURL.
func readConfiguredPack(pc PackConfig) (*resource.Pack, error) {
	switch {
	case pc.URL != "" && pc.Path != "":
		return nil, errors.New("pack has both URL and Path")
	case pc.URL != "":
		return readURLPack(pc.URL)
	case pc.Path != "":
		return resource.ReadPath(pc.Path)
	default:
		return nil, errors.New("pack has neither URL nor Path")
	}
}

// readURLPack downloads and compiles the resource pack at the given URL WITHOUT
// recording the URL as the pack's CDN DownloadURL. resource.ReadURL sets
// pack.DownloadURL to the source URL, which the listener then advertises in
// TexturePackInfo.DownloadURL, making clients fetch the pack over HTTPS
// themselves. That client-side CDN path is unreliable (it requires a direct,
// redirect-free URL serving a zip whose manifest.json is NOT at the archive
// root — GitHub release assets violate both), and only triggers on a cache
// miss, i.e. exactly when the hub and this proxy serve different pack
// versions: players got stuck on "Loading resource packs" or joined with a
// half-applied pack. Serving over RakNet chunks (empty DownloadURL) matches
// how the hub delivers packs and works for every client.
//
// Note the subfolder layout above is a requirement of the client-side CDN
// download only; server-side compilation (resource.Read here, resource.ReadPath
// on the hub) accepts manifest.json at the archive root, which is exactly the
// layout our release assets use.
func readURLPack(url string) (*resource.Pack, error) {
	// Bound the whole fetch (connect + body read inside resource.Read): this
	// runs during listener setup, so an unbounded http.Get on a stalled URL
	// would hang proxy startup forever.
	client := &http.Client{Timeout: 5 * time.Minute}
	resp, err := client.Get(url)
	if err != nil {
		return nil, fmt.Errorf("download resource pack: %w", err)
	}
	defer func() {
		_ = resp.Body.Close()
	}()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("download resource pack: %v (%d)", resp.Status, resp.StatusCode)
	}
	return resource.Read(resp.Body)
}

// loadCommands loads commands from the file.
func (c UserConfig) loadCommands(log *slog.Logger) error {
	rawBytes, err := os.ReadFile(c.Resources.CommandPath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			dir := filepath.Dir(c.Resources.CommandPath)
			if err = os.MkdirAll(dir, os.ModePerm); err != nil {
				panic(err)
			}
			if createErr := os.WriteFile(c.Resources.CommandPath, []byte("{}"), os.ModePerm); createErr != nil {
				return createErr
			}
			rawBytes, err = os.ReadFile(c.Resources.CommandPath)
			if err != nil {
				return err
			}
		} else {
			return err
		}
	}

	var commands map[string]cmd.EngineResponseCommand
	if err = json.Unmarshal(rawBytes, &commands); err != nil {
		log.Error("failed to unmarshal commands", "err", err)
		return err
	}
	cmd.LoadFrom(commands)
	log.Info("loaded commands", "count", len(commands))

	return nil
}

// makeBorder returns new border instance.
func (c UserConfig) makeBorder() *area.Area2D {
	if !c.Border.Enabled {
		return nil
	}
	return area.NewArea2D(c.Border.MinX, c.Border.MinZ, c.Border.MaxX, c.Border.MaxZ)
}

// afkTimer returns new AFKTimer instance.
func (c UserConfig) afkTimer() *infra.AFKTimer {
	if !c.AFKTimer.Enabled {
		return nil
	}
	parse := func(s string, fallback time.Duration) time.Duration {
		if s == "" {
			return fallback
		}
		d, err := time.ParseDuration(s)
		if err != nil {
			return fallback
		}
		return d
	}

	timeout := parse(c.AFKTimer.TimeoutDuration, 10*time.Minute)
	warn := parse(c.AFKTimer.WarnApproaching, 4*time.Minute)
	mark := parse(c.AFKTimer.MarkAFK, 5*time.Minute)
	final := parse(c.AFKTimer.FinalWarning, 9*time.Minute)

	threshold := c.AFKTimer.FullnessThreshold
	if threshold <= 0 || threshold > 1 {
		threshold = 0.9
	}

	return &infra.AFKTimer{
		TimeoutDuration:   timeout,
		WarnApproaching:   warn,
		MarkAFK:           mark,
		FinalWarning:      final,
		FullnessThreshold: threshold,
	}
}

// whiteList returns new Whitelist instance.
func (c UserConfig) whiteList(log *slog.Logger) *whitelist.Whitelist {
	if !c.Network.Whitelisted {
		return nil
	}
	conf, err := whitelist.ReadConfig(c.Network.WhitelistPath)
	if err != nil {
		log.Error("failed to read whitelist config", "err", err)
		return nil
	}
	return whitelist.NewWhitelist(conf.Entries)
}

// dialerFunc returns a dialer func for a specific server.
func (c UserConfig) dialerFunc(remoteAddress, transport string, log *slog.Logger) DialerFunc {
	return func(identityData login.IdentityData, clientData login.ClientData, ctx context.Context) (session.Conn, error) {
		d := minecraft.Dialer{
			ClientData:   clientData,
			IdentityData: identityData,

			DownloadResourcePack: func(id uuid.UUID, version string, current, total int) bool { return false },

			FlushRate:           time.Millisecond * time.Duration(c.Network.FlushRate),
			ErrorLog:            log,
			KeepXBLIdentityData: true,
		}
		if transport == "nethernet" {
			url := netherNetDialAddress(remoteAddress)
			return d.DialContextNetwork(ctx, minecraft.NetherNet{
				Signaling: endpoint.NewClient(),
				Dialer: nethernet.Dialer{
					API:                     newNetherNetDialAPI(url),
					DisableTrickleICE:       true,
					AllowIdentitylessServer: true,
					Log:                     log,
				},
				Log: log,
			}, url)
		}
		return d.DialContext(ctx, "raknet", remoteAddress)
	}
}

// listenConfig returns the shared Minecraft listen config for a server.
// Both the RakNet listener and the NetherNet listener use this config, so
// FetchResourcePacks filters packs for each transport.
func (c UserConfig) listenConfig(srv *Server, offers []packOffer) minecraft.ListenConfig {
	cfg := minecraft.ListenConfig{
		ErrorLog:       srv.Log,
		StatusProvider: srv.StatusProvider,

		FlushRate:            time.Millisecond * time.Duration(c.Network.FlushRate),
		ResourcePacks:        packsFrom(stackOrder(offers)),
		TexturePacksRequired: c.Resources.PacksRequired,
		// Called before ResourcePacksInfo (gophertunnel minecraft/conn.go:889).
		// The returned slice replaces the listener's packs for this client.
		FetchResourcePacks: func(_ login.IdentityData, client login.ClientData, _ []*resource.Pack) []*resource.Pack {
			matched := filterPackOffers(offers, client.MemoryTier, client.LanguageCode)
			return packsFrom(stackOrder(matched))
		},
	}

	if srv.Log.Enabled(context.Background(), slog.LevelDebug) {
		cfg.ErrorLog = srv.Log.With("net origin", "gophertunnel")
	}
	return cfg
}

// normalizeLanguage lowercases a client or config language code and treats
// '-' and '_' as the same separator.
func normalizeLanguage(code string) string {
	return strings.ToLower(strings.ReplaceAll(code, "-", "_"))
}

// packMatches reports whether a client with memoryTier and languageCode
// should receive a pack with filter f. MinMemoryTier 0 and an empty Language
// match every client. Both conditions are AND-ed.
func packMatches(f packFilter, memoryTier int, languageCode string) bool {
	if f.MinMemoryTier != 0 && memoryTier < f.MinMemoryTier {
		return false
	}
	if f.Language != "" && normalizeLanguage(languageCode) != normalizeLanguage(f.Language) {
		return false
	}
	return true
}

// filterPackOffers keeps offers the client should receive, in offers order
// (highest priority first).
func filterPackOffers(offers []packOffer, memoryTier int, languageCode string) []packOffer {
	matched := make([]packOffer, 0, len(offers))
	for _, offer := range offers {
		if packMatches(offer.filter, memoryTier, languageCode) {
			matched = append(matched, offer)
		}
	}
	return matched
}

// stackOrder converts highest-priority-first offers into gophertunnel's
// ResourcePacks order (lowest priority first).
//
// gophertunnel v1.62.0 copies conn.resourcePacks in slice order into
// ResourcePackStack.TexturePacks (minecraft/conn.go:1117) and does not
// reverse it. packet.ResourcePackStack says the first entry "will be applied
// first" (minecraft/protocol/packet/resource_pack_stack.go:16), so index 0
// on the wire is the lowest priority. Resources.Packs is the opposite
// (index 0 highest, same as packs.json), so that prefix is reversed.
//
// Legacy URLResources/PathResources are not part of that prefix. They keep
// today's load order and are placed before Packs, which leaves a Packs-empty
// config unchanged and keeps a monolith lower priority than Packs.
// highestFirst must be Packs (legacy == false) followed by legacy entries.
func stackOrder(highestFirst []packOffer) []packOffer {
	split := 0
	for split < len(highestFirst) && !highestFirst[split].legacy {
		split++
	}
	out := make([]packOffer, 0, len(highestFirst))
	out = append(out, highestFirst[split:]...)
	for i := split - 1; i >= 0; i-- {
		out = append(out, highestFirst[i])
	}
	return out
}

func packsFrom(offers []packOffer) []*resource.Pack {
	out := make([]*resource.Pack, len(offers))
	for i, offer := range offers {
		out[i] = offer.pack
	}
	return out
}

// listenerFunc returns a RakNet listener for a specific server.
func (c UserConfig) listenerFunc(srv *Server, cfg minecraft.ListenConfig) (Listener, error) {
	l, err := cfg.ListenNetwork(loggedRakNet{log: cfg.ErrorLog}, srv.LocalAddress)
	if err != nil {
		return nil, fmt.Errorf("create listener: %w", err)
	}
	srv.Log.Info("listener running.", "addr", l.Addr())
	return listener{l}, nil
}

// DefaultConfig ...
func DefaultConfig() UserConfig {
	c := UserConfig{}

	c.Network.ServerRegion = "Some region"

	const defaultKey = "secret-key"
	c.Network.Servers = []ServerConfig{
		{
			Name:          "Some server",
			LocalAddress:  "127.0.0.1:19132",
			RemoteAddress: "127.0.0.1:19133",
			MOTD:          "Some server",
			MaxPlayers:    85,
			ClaimService: struct {
				Enabled bool
				URL     string
				Key     string
			}{
				Enabled: false,
				URL:     "http://127.0.0.1:8080/fetch/claims",
				Key:     defaultKey,
			},
		},
	}

	c.Network.Whitelisted = false
	c.Network.WhitelistPath = "whitelist.json"

	c.Network.SecuredSlots = 0
	c.Network.MaxRenderDistance = 16
	c.Network.FlushRate = 20

	c.Border.Enabled = false
	c.Claims.PrefilterEnabled = false
	c.Claims.DenyRenderingEnabled = false
	c.Claims.PollInterval = claim.DefaultPollInterval.String()
	c.Claims.MaxSnapshotAge = claim.DefaultMaxSnapshotAge.String()

	c.AFKTimer.Enabled = true
	c.AFKTimer.TimeoutDuration = "10m"
	c.AFKTimer.WarnApproaching = "4m"
	c.AFKTimer.MarkAFK = "5m"
	c.AFKTimer.FinalWarning = "9m"
	c.AFKTimer.FullnessThreshold = 0.9

	c.Resources.PacksRequired = false
	c.Resources.CommandPath = "resources/commands.json"

	c.AuthenticationService.Enabled = false
	c.AuthenticationService.URL = "http://127.0.0.1:8080/authentication"
	c.AuthenticationService.Key = defaultKey

	c.VPNService.Enabled = false
	c.VPNService.URL = "http://ip-api.com/json"
	// Megalink S.R.L. (Argentina) — residential ISP flagged as proxy by ip-api.
	c.VPNService.WhitelistedCIDRs = []string{"45.230.64.0/22"}

	c.TrafficProtection = session.DefaultTrafficConfig()
	c.DuplicateXUID.Enabled = false

	c.Encryption.Key = defaultKey
	return c
}

// ReadConfig ...
func ReadConfig() (UserConfig, error) {
	g := gophig.NewGophig[UserConfig]("./config.toml", gophig.TOMLMarshaler{}, os.ModePerm)
	_, err := g.LoadConf()
	if os.IsNotExist(err) {
		err = g.SaveConf(DefaultConfig())
		if err != nil {
			return UserConfig{}, err
		}
	}
	c, err := g.LoadConf()
	return c, err
}
