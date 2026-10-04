package registry

import "testing"

func dimRegistry(names ...string) Registry[Dimension] {
	var r Registry[Dimension]
	r.Value = make([]struct {
		Name    string    `nbt:"name"`
		ID      int32     `nbt:"id"`
		Element Dimension `nbt:"element"`
	}, len(names))
	for i, n := range names {
		r.Value[i].Name = n
		r.Value[i].ID = int32(i)
		r.Value[i].Element.MinY = int32(-64 - i)
	}
	return r
}

// A proxy may strip the namespace from the registry, from the packet that
// refers to it, or from neither. All four combinations have to resolve.
func TestRegistryFind_NamespaceInsensitive(t *testing.T) {
	for _, tc := range []struct{ stored, looked string }{
		{"minecraft:overworld", "minecraft:overworld"},
		{"minecraft:overworld", "overworld"},
		{"overworld", "minecraft:overworld"},
		{"overworld", "overworld"},
	} {
		r := dimRegistry(tc.stored)
		i, dim := r.Find(tc.looked)
		if dim == nil {
			t.Errorf("stored %q, looked up %q: not found", tc.stored, tc.looked)
			continue
		}
		if i != 0 {
			t.Errorf("stored %q, looked up %q: got index %d, want 0", tc.stored, tc.looked, i)
		}
	}
}

// The right entry, not merely some entry.
func TestRegistryFind_PicksTheRightEntry(t *testing.T) {
	r := dimRegistry("minecraft:overworld", "minecraft:the_nether", "minecraft:the_end")
	i, dim := r.Find("the_nether")
	if dim == nil {
		t.Fatal("the_nether not found")
	}
	if i != 1 || dim.MinY != -65 {
		t.Errorf("got index %d min_y %d, want 1 and -65", i, dim.MinY)
	}
}

// Stripping the prefix must not make unrelated namespaces collide.
func TestRegistryFind_OtherNamespacesStayDistinct(t *testing.T) {
	r := dimRegistry("mymod:overworld")
	if _, dim := r.Find("minecraft:overworld"); dim != nil {
		t.Error("mymod:overworld should not answer to minecraft:overworld")
	}
	if _, dim := r.Find("mymod:overworld"); dim == nil {
		t.Error("mymod:overworld should answer to its own full name")
	}
}

func TestRegistryFind_Missing(t *testing.T) {
	r := dimRegistry("minecraft:overworld")
	i, dim := r.Find("minecraft:the_end")
	if dim != nil || i != -1 {
		t.Errorf("got index %d dim %v, want -1 and nil", i, dim)
	}
}
