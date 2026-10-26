package session

import (
	"testing"

	"github.com/sandertv/gophertunnel/minecraft/protocol"
	"github.com/sandertv/gophertunnel/minecraft/protocol/packet"
	"github.com/smell-of-curry/gobds/gobds/cmd"
)

func TestPackCommandShadowsVanilla(t *testing.T) {
	cmd.Register(cmd.New("zzshadowtest", "pack command", []string{"zzShadowAlias"}, nil, false))
	pkt := &packet.AvailableCommands{Commands: []protocol.Command{
		{Name: "zzshadowtest", Description: "vanilla by name"},
		{Name: "zzshadowalias", Description: "vanilla by alias"},
		{Name: "give", Description: "vanilla give"},
	}}

	h := &AvailableCommandsHandler{}
	if err := h.Handle(nil, pkt, nil); err != nil {
		t.Fatal(err)
	}

	for _, c := range pkt.Commands {
		if c.Description == "vanilla by name" || c.Description == "vanilla by alias" {
			t.Fatalf("vanilla %q still listed next to the pack command", c.Name)
		}
	}
	for _, name := range []string{"zzshadowtest", "zzshadowalias"} {
		if _, ok := h.cache.Load(name); ok {
			t.Fatalf("vanilla %s still cached, /%s would go to BDS", name, name)
		}
	}
	if _, ok := h.cache.Load("give"); !ok {
		t.Fatal("vanilla give missing from cache")
	}
}
