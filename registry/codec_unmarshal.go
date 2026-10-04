package registry

import (
	"fmt"

	"github.com/Tnze/go-mc/nbt"
)

// namespace is the implied namespace of every vanilla registry key.
const namespace = "minecraft:"

// registryFields maps the un-namespaced registry name to the NetworkCodec
// field it fills. Keep in sync with the struct tags on NetworkCodec.
func (c *NetworkCodec) registryFields() map[string]any {
	return map[string]any{
		"chat_type":      &c.ChatType,
		"damage_type":    &c.DamageType,
		"dimension_type": &c.DimensionType,
		"trim_material":  &c.TrimMaterial,
		"trim_pattern":   &c.TrimPattern,
		"worldgen/biome": &c.WorldGenBiome,
	}
}

// UnmarshalNBT decodes the registry codec compound, accepting registry keys
// both namespaced ("minecraft:dimension_type", what a vanilla server sends)
// and bare ("dimension_type", what protocol-downgrading proxies such as
// ViaVersion/ViaProxy emit). Keys that match no known registry are skipped
// instead of failing the decode.
//
// The permissiveness is deliberately one-directional: unknown keys are
// ignored, but a registry we do know about is still decoded strictly, so a
// malformed dimension_type is an error rather than a silently empty registry
// (the bot reads min_y and height out of it).
func (c *NetworkCodec) UnmarshalNBT(tagType byte, r nbt.DecoderReader) error {
	if tagType != nbt.TagCompound {
		return fmt.Errorf("registry: cannot decode network codec from tag %#02x", tagType)
	}

	// Buffer the compound, then re-read it as a map so the key lookup can be
	// done by us rather than by the struct-tag matcher.
	var raw nbt.RawMessage
	if err := raw.UnmarshalNBT(tagType, r); err != nil {
		return err
	}
	var fields map[string]nbt.RawMessage
	if err := raw.Unmarshal(&fields); err != nil {
		return err
	}

	for name, target := range c.registryFields() {
		value, ok := fields[namespace+name]
		if !ok {
			value, ok = fields[name]
		}
		if !ok {
			continue
		}
		if err := value.Unmarshal(target); err != nil {
			return fmt.Errorf("registry: fail to decode %q: %w", name, err)
		}
	}
	return nil
}
