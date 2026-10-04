package level

import (
	"bytes"
	"testing"

	pk "github.com/Tnze/go-mc/net/packet"
)

// A 26.2 server downgraded by ViaProxy sends MOTION_BLOCKING_NO_LEAVES
// alongside the two heightmaps this version knows. Decoding strictly made
// every chunk packet fail, which disconnects the client the moment it joins.
func TestChunkHeightmaps_UnknownEntryIsSkipped(t *testing.T) {
	var buf bytes.Buffer
	sent := struct {
		MotionBlocking         []uint64 `nbt:"MOTION_BLOCKING"`
		MotionBlockingNoLeaves []uint64 `nbt:"MOTION_BLOCKING_NO_LEAVES"`
		WorldSurface           []uint64 `nbt:"WORLD_SURFACE"`
	}{
		MotionBlocking:         []uint64{1, 2, 3},
		MotionBlockingNoLeaves: []uint64{9, 9, 9},
		WorldSurface:           []uint64{4, 5, 6},
	}
	if _, err := (pk.NBTField{V: sent}).WriteTo(&buf); err != nil {
		t.Fatalf("encode: %v", err)
	}

	var got struct {
		MotionBlocking []uint64 `nbt:"MOTION_BLOCKING"`
		WorldSurface   []uint64 `nbt:"WORLD_SURFACE"`
	}
	if _, err := (pk.NBTField{V: &got, AllowUnknownFields: true}).ReadFrom(&buf); err != nil {
		t.Fatalf("decode: %v", err)
	}

	for _, tc := range []struct {
		name      string
		got, want []uint64
	}{
		{"MOTION_BLOCKING", got.MotionBlocking, sent.MotionBlocking},
		{"WORLD_SURFACE", got.WorldSurface, sent.WorldSurface},
	} {
		if len(tc.got) != len(tc.want) {
			t.Errorf("%s: got %v, want %v", tc.name, tc.got, tc.want)
			continue
		}
		for i := range tc.want {
			if tc.got[i] != tc.want[i] {
				t.Errorf("%s: got %v, want %v", tc.name, tc.got, tc.want)
				break
			}
		}
	}
}

// The same payload read strictly is what used to happen, and it still fails --
// so the test above is proving the flag, not a decoder that became lenient
// everywhere.
func TestChunkHeightmaps_StrictDecodeStillRejects(t *testing.T) {
	var buf bytes.Buffer
	sent := struct {
		MotionBlockingNoLeaves []uint64 `nbt:"MOTION_BLOCKING_NO_LEAVES"`
	}{MotionBlockingNoLeaves: []uint64{1}}
	if _, err := (pk.NBTField{V: sent}).WriteTo(&buf); err != nil {
		t.Fatalf("encode: %v", err)
	}

	var got struct {
		MotionBlocking []uint64 `nbt:"MOTION_BLOCKING"`
	}
	if _, err := (pk.NBTField{V: &got}).ReadFrom(&buf); err == nil {
		t.Fatal("strict decode should still reject an unknown heightmap")
	}
}
