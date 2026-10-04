package registry

import (
	"strings"
	"testing"

	"github.com/Tnze/go-mc/nbt"
)

// The wire shape of a Registry[E], spelled out independently of the decoder's
// own types so the test exercises the bytes rather than a round trip.
type wireEntry[E any] struct {
	Name    string `nbt:"name"`
	ID      int32  `nbt:"id"`
	Element E      `nbt:"element"`
}

type wireRegistry[E any] struct {
	Type  string         `nbt:"type"`
	Value []wireEntry[E] `nbt:"value"`
}

// raw encodes v as the payload of a single tag, the form RawMessage holds.
func raw(t *testing.T, v any) nbt.RawMessage {
	t.Helper()
	var sb strings.Builder
	enc := nbt.NewEncoder(&sb)
	enc.NetworkFormat(true) // root tag type byte, no name
	if err := enc.Encode(v, ""); err != nil {
		t.Fatalf("encode: %v", err)
	}
	data := []byte(sb.String())
	return nbt.RawMessage{Type: data[0], Data: data[1:]}
}

const (
	testMinY   = -64
	testHeight = 384
)

func overworld(t *testing.T) wireRegistry[Dimension] {
	t.Helper()
	return wireRegistry[Dimension]{
		Type: "minecraft:dimension_type",
		Value: []wireEntry[Dimension]{{
			Name: "minecraft:overworld",
			ID:   0,
			Element: Dimension{
				HasSkylight: true,
				Natural:     true,
				MinY:        testMinY,
				Height:      testHeight,
				// RawMessage must hold a real tag to be re-encodable.
				MonsterSpawnLightLevel: raw(t, int32(0)),
			},
		}},
	}
}

func chatTypes() wireRegistry[ChatType] {
	return wireRegistry[ChatType]{
		Type:  "minecraft:chat_type",
		Value: []wireEntry[ChatType]{{Name: "minecraft:chat", ID: 0}},
	}
}

// decode marshals fields as a compound and feeds it through NetworkCodec.
func decode(t *testing.T, fields map[string]nbt.RawMessage) (NetworkCodec, error) {
	t.Helper()
	data, err := nbt.Marshal(fields)
	if err != nil {
		t.Fatalf("marshal codec: %v", err)
	}
	var codec NetworkCodec
	err = nbt.Unmarshal(data, &codec)
	return codec, err
}

func assertDimension(t *testing.T, codec NetworkCodec) {
	t.Helper()
	_, dim := codec.DimensionType.Find("minecraft:overworld")
	if dim == nil {
		t.Fatal("dimension_type registry is empty; min_y and height would be misread")
	}
	if dim.MinY != testMinY || dim.Height != testHeight {
		t.Errorf("dimension: got min_y=%d height=%d, want %d and %d",
			dim.MinY, dim.Height, testMinY, testHeight)
	}
}

// A vanilla server namespaces every registry key.
func TestNetworkCodec_NamespacedKeys(t *testing.T) {
	codec, err := decode(t, map[string]nbt.RawMessage{
		"minecraft:dimension_type": raw(t, overworld(t)),
		"minecraft:chat_type":      raw(t, chatTypes()),
	})
	if err != nil {
		t.Fatalf("decode: %v", err)
	}
	assertDimension(t, codec)
	if len(codec.ChatType.Value) != 1 {
		t.Errorf("chat_type: got %d entries, want 1", len(codec.ChatType.Value))
	}
}

// ViaProxy's downgrade drops the namespace. This is the bug: go-mc used to
// fail the whole packet with `unknown field "worldgen/biome"`.
func TestNetworkCodec_BareKeys(t *testing.T) {
	codec, err := decode(t, map[string]nbt.RawMessage{
		"dimension_type": raw(t, overworld(t)),
		"chat_type":      raw(t, chatTypes()),
		"worldgen/biome": raw(t, wireRegistry[nbt.RawMessage]{Type: "worldgen/biome"}),
	})
	if err != nil {
		t.Fatalf("decode: %v", err)
	}
	assertDimension(t, codec)
}

// Mixed spellings in one payload, with the namespaced key winning.
func TestNetworkCodec_MixedKeys(t *testing.T) {
	codec, err := decode(t, map[string]nbt.RawMessage{
		"minecraft:dimension_type": raw(t, overworld(t)),
		"chat_type":                raw(t, chatTypes()),
	})
	if err != nil {
		t.Fatalf("decode: %v", err)
	}
	assertDimension(t, codec)
	if len(codec.ChatType.Value) != 1 {
		t.Errorf("chat_type: got %d entries, want 1", len(codec.ChatType.Value))
	}
}

// Registries from a newer protocol are skipped, not fatal.
func TestNetworkCodec_UnknownRegistryIgnored(t *testing.T) {
	codec, err := decode(t, map[string]nbt.RawMessage{
		"minecraft:dimension_type": raw(t, overworld(t)),
		"minecraft:wolf_variant":   raw(t, wireRegistry[nbt.RawMessage]{Type: "minecraft:wolf_variant"}),
		"banner_pattern":           raw(t, wireRegistry[nbt.RawMessage]{Type: "banner_pattern"}),
	})
	if err != nil {
		t.Fatalf("decode: %v", err)
	}
	assertDimension(t, codec)
}

// The leniency must not extend to registries we do know: silently accepting a
// broken dimension_type would have the bot join and misread every coordinate.
func TestNetworkCodec_MalformedKnownRegistryErrors(t *testing.T) {
	_, err := decode(t, map[string]nbt.RawMessage{
		"dimension_type": raw(t, "not a registry"),
	})
	if err == nil {
		t.Fatal("expected an error for a malformed dimension_type, got nil")
	}
	if !strings.Contains(err.Error(), "dimension_type") {
		t.Errorf("error should name the registry, got: %v", err)
	}
}

// A codec missing a registry entirely leaves that field zeroed, not an error.
func TestNetworkCodec_MissingRegistry(t *testing.T) {
	codec, err := decode(t, map[string]nbt.RawMessage{
		"dimension_type": raw(t, overworld(t)),
	})
	if err != nil {
		t.Fatalf("decode: %v", err)
	}
	assertDimension(t, codec)
	if len(codec.ChatType.Value) != 0 {
		t.Errorf("chat_type should be empty, got %d entries", len(codec.ChatType.Value))
	}
}

func TestNetworkCodec_NonCompoundErrors(t *testing.T) {
	var codec NetworkCodec
	data, err := nbt.Marshal("a string, not a compound")
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if err := nbt.Unmarshal(data, &codec); err == nil {
		t.Fatal("expected an error decoding a non-compound codec, got nil")
	}
}
