package session

import "testing"

func TestCommandChunksInOrder(t *testing.T) {
	var c commandChunks
	if full, done, err := c.add(1, 3, "aa"); err != nil || done || full != "" {
		t.Fatalf("chunk 1: full=%q done=%v err=%v", full, done, err)
	}
	if full, done, err := c.add(2, 3, "bb"); err != nil || done || full != "" {
		t.Fatalf("chunk 2: full=%q done=%v err=%v", full, done, err)
	}
	full, done, err := c.add(3, 3, "cc")
	if err != nil || !done || full != "aabbcc" {
		t.Fatalf("chunk 3: full=%q done=%v err=%v", full, done, err)
	}
}

func TestCommandChunksOutOfOrder(t *testing.T) {
	var c commandChunks
	if _, _, err := c.add(1, 3, "aa"); err != nil {
		t.Fatalf("chunk 1: %v", err)
	}
	if _, _, err := c.add(3, 3, "cc"); err == nil {
		t.Fatal("out-of-order chunk 3 must error")
	}
}

func TestCommandChunksNMismatch(t *testing.T) {
	var c commandChunks
	if _, _, err := c.add(1, 3, "aa"); err != nil {
		t.Fatalf("chunk 1: %v", err)
	}
	if _, _, err := c.add(2, 4, "bb"); err == nil {
		t.Fatal("n mismatch must error")
	}
}

func TestCommandChunksRestartAtOne(t *testing.T) {
	var c commandChunks
	if _, _, err := c.add(1, 3, "aa"); err != nil {
		t.Fatalf("chunk 1: %v", err)
	}
	if _, _, err := c.add(2, 3, "bb"); err != nil {
		t.Fatalf("chunk 2: %v", err)
	}
	if full, done, err := c.add(1, 2, "xx"); err != nil || done || full != "" {
		t.Fatalf("restart chunk 1: full=%q done=%v err=%v", full, done, err)
	}
	full, done, err := c.add(2, 2, "yy")
	if err != nil || !done || full != "xxyy" {
		t.Fatalf("restart chunk 2: full=%q done=%v err=%v", full, done, err)
	}
}

func TestCommandChunksRejectsNOver64(t *testing.T) {
	var c commandChunks
	if _, _, err := c.add(1, 65, "aa"); err == nil {
		t.Fatal("n>64 must error")
	}
}
