package gobds

import (
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/gofrs/flock"
	"github.com/sandertv/gophertunnel/minecraft/protocol/login"
	"github.com/smell-of-curry/gobds/gobds/session"
)

// Player ...
type Player struct {
	PlayerName         string `json:"playerName"`
	Identity           string `json:"identity"`
	ClientSelfSignedID string `json:"clientSelfSignedID"`
}

// PlayerManager ...
type PlayerManager struct {
	mu      sync.RWMutex
	players map[string]Player

	filePath string
	fileLock *flock.Flock

	// dirty flag indicates if the in-memory state has changed since the last save.
	dirty bool

	log      *slog.Logger
	shutdown chan struct{}
	wg       sync.WaitGroup
}

// NewPlayerManager ...
func NewPlayerManager(filePath string, log *slog.Logger) (*PlayerManager, error) {
	m := &PlayerManager{
		players: make(map[string]Player),

		filePath: filePath,
		fileLock: flock.New(filePath + ".lock"),

		log:      log,
		shutdown: make(chan struct{}),
	}
	if err := m.load(); err != nil {
		log.Error("failed to load player manager", "error", err)
	}
	return m, nil
}

// Start ...
func (m *PlayerManager) Start(interval time.Duration) {
	m.wg.Add(1)
	go func() {
		defer m.wg.Done()
		t := time.NewTicker(interval)
		defer t.Stop()
		for {
			select {
			case <-t.C:
				if err := m.save(); err != nil {
					m.log.Error("error saving player manager", "error", err)
				}
			case <-m.shutdown:
				m.log.Error("shutting down player manager")
				return
			}
		}
	}()
}

// Close ...
func (m *PlayerManager) Close() {
	close(m.shutdown)
	m.wg.Wait()
	if err := m.save(); err != nil {
		m.log.Error("error saving player manager", "error", err)
	}
}

// load ...
func (m *PlayerManager) load() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if err := os.MkdirAll(filepath.Dir(m.filePath), os.ModePerm); err != nil {
		panic(fmt.Errorf("failed to create directory: %w", err))
	}

	if err := m.fileLock.Lock(); err != nil {
		panic(err)
	}
	defer func() {
		if err := m.fileLock.Unlock(); err != nil {
			m.log.Error("failed to unlock file during load", "err", err)
		}
	}()

	data, err := os.ReadFile(m.filePath)
	if err != nil {
		if os.IsNotExist(err) {
			if createErr := os.WriteFile(m.filePath, []byte("{}"), os.ModePerm); createErr != nil {
				panic(err)
			}
			return nil
		}
		return err
	}

	if len(data) == 0 {
		return nil
	}
	return json.Unmarshal(data, &m.players)
}

// save ...
func (m *PlayerManager) save() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if !m.dirty {
		return nil
	}

	data, err := json.MarshalIndent(m.players, "", "  ")
	if err != nil {
		return err
	}

	if err = m.fileLock.Lock(); err != nil {
		return err
	}
	defer func() {
		if err := m.fileLock.Unlock(); err != nil {
			m.log.Error("failed to unlock file during save", "err", err)
		}
	}()

	tempFile := m.filePath + ".tmp"
	if err = os.WriteFile(tempFile, data, os.ModePerm); err != nil {
		return err
	}
	if err = os.Rename(tempFile, m.filePath); err != nil {
		return err
	}

	m.dirty = false
	m.log.Debug("saved player manager to disk", "path", m.filePath)
	return nil
}

// IdentityDataOf ...
func (m *PlayerManager) IdentityDataOf(conn session.Conn) login.IdentityData {
	m.mu.RLock()
	xuid := conn.IdentityData().XUID
	player, exists := m.players[xuid]
	m.mu.RUnlock()

	if exists {
		return login.IdentityData{
			XUID:        xuid,
			DisplayName: player.PlayerName,
			Identity:    player.Identity,
			TitleID:     conn.IdentityData().TitleID,
		}
	}

	m.mu.Lock()
	defer m.mu.Unlock()
	if p, ok := m.players[xuid]; ok {
		player = p
	} else {
		newPlayer := Player{
			PlayerName:         conn.IdentityData().DisplayName,
			Identity:           conn.IdentityData().Identity,
			ClientSelfSignedID: conn.ClientData().SelfSignedID,
		}
		m.players[xuid] = newPlayer
		m.dirty = true
		player = newPlayer
		m.log.Debug("added player to list", "name", newPlayer.PlayerName, "xuid", xuid)
	}
	return login.IdentityData{
		XUID:        xuid,
		DisplayName: player.PlayerName,
		Identity:    player.Identity,
		TitleID:     conn.IdentityData().TitleID,
	}
}

// ClientDataOf ...
func (m *PlayerManager) ClientDataOf(conn session.Conn) login.ClientData {
	m.mu.RLock()
	xuid := conn.IdentityData().XUID
	player, exists := m.players[xuid]
	clientData := conn.ClientData()
	m.mu.RUnlock()

	if exists {
		clientData.SelfSignedID = player.ClientSelfSignedID
		return clientData
	}
	m.log.Warn("client data not found", "xuid", xuid)
	return clientData
}

// XUIDFromPlayerName ...
func (m *PlayerManager) XUIDFromPlayerName(name string) (string, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	for xuid, player := range m.players {
		if strings.EqualFold(player.PlayerName, name) {
			return xuid, nil
		}
	}

	return "", errors.New("player not found")
}

// PlayerFromXUID ...
func (m *PlayerManager) PlayerFromXUID(xuid string) (Player, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	player, ok := m.players[xuid]
	if !ok {
		return Player{}, errors.New("player not found")
	}

	return player, nil
}
