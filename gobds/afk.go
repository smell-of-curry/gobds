package gobds

import (
	"context"
	"sort"
	"time"

	"github.com/sandertv/gophertunnel/minecraft/text"
	"github.com/smell-of-curry/gobds/gobds/session"
)

// afkEvaluatorInterval is how often the AFK evaluator ticks. Movement is
// tracked per-packet on the session itself so 2s is plenty of granularity.
const afkEvaluatorInterval = 2 * time.Second

// afkCandidate pairs a session with its current idle duration so the evaluator
// can sort and filter without re-reading the session state more than once.
type afkCandidate struct {
	s   *session.Session
	dur time.Duration
}

// afkEvaluator runs per-Server for the lifetime of the listen loop. It sends
// soft warnings regardless of fullness, and only escalates to the final
// warning and actual kicks when the server is at or above the configured
// fullness threshold. When kicking, the longest-AFK sessions go first.
func (gb *GoBDS) afkEvaluator(srv *Server, ctx context.Context) {
	timer := gb.conf.AFKTimer
	if timer == nil {
		return
	}

	t := time.NewTicker(afkEvaluatorInterval)
	defer t.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-gb.ctx.Done():
			return
		case <-t.C:
			gb.evaluateAFK(srv)
		}
	}
}

// evaluateAFK performs a single pass over the server's sessions.
func (gb *GoBDS) evaluateAFK(srv *Server) {
	timer := gb.conf.AFKTimer
	sessions := srv.Sessions()
	if len(sessions) == 0 {
		return
	}

	cands := make([]afkCandidate, 0, len(sessions))
	for _, s := range sessions {
		cands = append(cands, afkCandidate{s: s, dur: s.AFKDuration()})
	}

	// Soft warnings always fire regardless of fullness. They only fire once
	// per idle streak because the flags are reset on movement.
	for _, c := range cands {
		if c.dur >= timer.WarnApproaching && !c.s.WarnedApproaching() {
			c.s.Message(text.Colourf("<yellow>You will be marked AFK in 1 minute. Move to reset your timer.</yellow>"))
			c.s.SetWarnedApproaching(true)
		}
		if c.dur >= timer.MarkAFK && !c.s.MarkedAFK() {
			c.s.Message(text.Colourf("<gold>You are now AFK. Move to reset your timer.</gold>"))
			c.s.SetMarkedAFK(true)
		}
	}

	if srv.StatusProvider == nil {
		return
	}
	status := srv.StatusProvider.ServerStatus(-1, -1)
	if status.MaxPlayers <= 0 {
		return
	}
	fullness := float64(status.PlayerCount) / float64(status.MaxPlayers)
	if fullness < timer.FullnessThreshold {
		return
	}

	for _, c := range cands {
		if c.dur >= timer.FinalWarning && !c.s.WarnedFinal() {
			c.s.Message(text.Colourf("<red>Server is near capacity. Move now or you will be kicked for being AFK.</red>"))
			c.s.SetWarnedFinal(true)
		}
	}

	// Only sessions that are past the kick threshold are eligible, sorted
	// longest-AFK first so the most-idle players go before the borderline ones.
	eligible := cands[:0:0]
	for _, c := range cands {
		if c.dur >= timer.TimeoutDuration {
			eligible = append(eligible, c)
		}
	}
	if len(eligible) == 0 {
		return
	}
	sort.Slice(eligible, func(i, j int) bool { return eligible[i].dur > eligible[j].dur })

	// Use a local counter because the upstream StatusProvider is foreign and
	// only refreshes periodically; without this we'd keep kicking until every
	// eligible AFK player is gone.
	threshold := int(float64(status.MaxPlayers) * timer.FullnessThreshold)
	remaining := status.PlayerCount
	for _, c := range eligible {
		if remaining < threshold {
			return
		}
		c.s.Disconnect(text.Colourf("<red>You've been kicked for being AFK.</red>"))
		remaining--
	}
}
