package registry

import "strings"

type Registry[E any] struct {
	Type  string `nbt:"type"`
	Value []struct {
		Name    string `nbt:"name"`
		ID      int32  `nbt:"id"`
		Element E      `nbt:"element"`
	} `nbt:"value"`
}

// Find looks an entry up by name, treating "minecraft:overworld" and
// "overworld" as the same key. The namespace is implied where it is missing:
// a vanilla server sends it on every name, but a protocol-downgrading proxy
// (ViaVersion/ViaProxy) strips it from the registry while the packet that
// then refers to the entry may keep it, or the other way round. Matching
// exactly leaves the client unable to find a dimension that is right there.
func (r *Registry[E]) Find(name string) (int32, *E) {
	for i := range r.Value {
		if sameKey(r.Value[i].Name, name) {
			return int32(i), &r.Value[i].Element
		}
	}
	return -1, nil
}

// sameKey compares two registry names ignoring an implied minecraft: prefix
// on either side. Keys in another namespace are compared whole, so
// "mymod:stone" never matches "minecraft:stone".
func sameKey(a, b string) bool {
	return a == b || stripNamespace(a) == stripNamespace(b)
}

func stripNamespace(name string) string {
	return strings.TrimPrefix(name, "minecraft:")
}

func (r *Registry[E]) FindByID(id int32) *E {
	if id >= 0 && id < int32(len(r.Value)) && r.Value[id].ID == id {
		return &r.Value[id].Element
	}
	for i := range r.Value {
		if r.Value[i].ID == id {
			return &r.Value[i].Element
		}
	}
	return nil
}
