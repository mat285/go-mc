package level

import (
	"bytes"
	"testing"

	pk "github.com/Tnze/go-mc/net/packet"
)

// buildLight encodes a light section the way a server does, with or without
// the Trust Edges boolean a downgrading proxy leaves out.
func buildLight(t *testing.T, trustEdges bool, skyMask, blockMask int64) []byte {
	t.Helper()
	count := func(m int64) int {
		n := 0
		for i := 0; i < 64; i++ {
			if m&(1<<i) != 0 {
				n++
			}
		}
		return n
	}
	arrays := func(n int) []pk.ByteArray {
		out := make([]pk.ByteArray, n)
		for i := range out {
			out[i] = make(pk.ByteArray, 2048)
			out[i][0] = byte(i + 1)
		}
		return out
	}
	sky, block := arrays(count(skyMask)), arrays(count(blockMask))

	var fields pk.Tuple
	if trustEdges {
		fields = append(fields, pk.Boolean(true))
	}
	fields = append(fields,
		pk.BitSet{skyMask}, pk.BitSet{blockMask},
		pk.BitSet{^skyMask}, pk.BitSet{^blockMask},
		pk.Array(sky), pk.Array(block),
	)
	var buf bytes.Buffer
	if _, err := fields.WriteTo(&buf); err != nil {
		t.Fatalf("encode: %v", err)
	}
	return buf.Bytes()
}

func TestLightDataReadsAVanillaSection(t *testing.T) {
	data := buildLight(t, true, 0x3c040, 0x40)
	var l lightData
	n, err := l.ReadFrom(bytes.NewReader(data))
	if err != nil {
		t.Fatalf("vanilla 1.20.2 light section rejected: %v", err)
	}
	if n != int64(len(data)) {
		t.Errorf("consumed %d bytes, want %d", n, len(data))
	}
	if len(l.SkyLight) != 5 || len(l.BlockLight) != 1 {
		t.Errorf("sky=%d block=%d, want 5 and 1", len(l.SkyLight), len(l.BlockLight))
	}
}

// TestLightDataReadsAProxySection is the bug: ViaProxy, downgrading 26.2 to
// 764, omits Trust Edges. Reading it anyway ate the sky mask's length byte and
// shifted every field after, which surfaced as "VarInt is too big" and dropped
// the connection on every join.
func TestLightDataReadsAProxySection(t *testing.T) {
	data := buildLight(t, false, 0x3c040, 0x40)
	var l lightData
	n, err := l.ReadFrom(bytes.NewReader(data))
	if err != nil {
		t.Fatalf("proxy light section rejected: %v", err)
	}
	if n != int64(len(data)) {
		t.Errorf("consumed %d bytes, want %d", n, len(data))
	}
	if len(l.SkyLight) != 5 || len(l.BlockLight) != 1 {
		t.Errorf("sky=%d block=%d, want 5 and 1", len(l.SkyLight), len(l.BlockLight))
	}
}

// Both layouts must give the same answer, or the shim is choosing badly.
func TestLightDataAgreesAcrossLayouts(t *testing.T) {
	var withEdges, without lightData
	if _, err := withEdges.ReadFrom(bytes.NewReader(buildLight(t, true, 0x3c040, 0x40))); err != nil {
		t.Fatal(err)
	}
	if _, err := without.ReadFrom(bytes.NewReader(buildLight(t, false, 0x3c040, 0x40))); err != nil {
		t.Fatal(err)
	}
	if len(withEdges.SkyLight) != len(without.SkyLight) ||
		len(withEdges.BlockLight) != len(without.BlockLight) ||
		withEdges.SkyLightMask[0] != without.SkyLightMask[0] ||
		withEdges.BlockLightMask[0] != without.BlockLightMask[0] {
		t.Error("the two layouts decoded to different light data")
	}
}

// The shim must not have become a way to accept anything: a section that fits
// neither layout is still an error.
func TestLightDataStillRejectsNonsense(t *testing.T) {
	for _, c := range []struct {
		name string
		data []byte
	}{
		{"truncated", buildLight(t, true, 0x3c040, 0x40)[:200]},
		{"trailing junk", append(buildLight(t, false, 0x40, 0x40), 0xff, 0xff, 0xff)},
		{"empty", nil},
		{"garbage", bytes.Repeat([]byte{0xff}, 64)},
	} {
		t.Run(c.name, func(t *testing.T) {
			var l lightData
			if _, err := l.ReadFrom(bytes.NewReader(c.data)); err == nil {
				t.Error("accepted a light section that fits neither layout")
			}
		})
	}
}

// A mask promising more arrays than follow must not pass: exact consumption
// alone would let a short section through if the counts happened to line up.
func TestLightDataChecksMasksAgainstArrays(t *testing.T) {
	data := buildLight(t, false, 0x3c040, 0x40)
	// Claim one more sky section than the arrays that follow.
	var l lightData
	bad := append([]byte{}, data...)
	bad[1] |= 0x80 // set another bit in the sky mask's first byte
	if _, err := l.ReadFrom(bytes.NewReader(bad)); err == nil {
		t.Error("accepted a mask that disagrees with the arrays it announces")
	}
}
