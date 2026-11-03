package history

import (
	"path/filepath"
	"slices"
	"testing"
)

func TestSavedNamesAreLoadedAgain(t *testing.T) {
	store := NewStore(filepath.Join(t.TempDir(), "nested", "received.json"))
	want := []string{"photo.png", "notes.txt"}

	if err := store.Save(want); err != nil {
		t.Fatalf("Save: %v", err)
	}
	got, err := store.Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if !slices.Equal(got, want) {
		t.Fatalf("Load = %v, want %v", got, want)
	}
}

func TestLoadWithoutAFileReturnsNothing(t *testing.T) {
	got, err := NewStore(filepath.Join(t.TempDir(), "received.json")).Load()

	if err != nil || len(got) != 0 {
		t.Fatalf("Load = %v, %v; want no names and no error", got, err)
	}
}
