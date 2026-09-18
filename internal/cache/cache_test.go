package cache

import "testing"

func TestFileStoreRoundTripAndIdempotentPut(t *testing.T) {
	store := &FileStore{Dir: t.TempDir()}
	key := Key("state", "rule")
	if _, ok, err := store.Get(key); err != nil || ok {
		t.Fatalf("unexpected miss: ok=%v err=%v", ok, err)
	}
	if err := store.Put(key, 0.91); err != nil {
		t.Fatal(err)
	}
	if err := store.Put(key, 0.91); err != nil {
		t.Fatal(err)
	}
	value, ok, err := store.Get(key)
	if err != nil || !ok || value != 0.91 {
		t.Fatalf("unexpected hit: value=%v ok=%v err=%v", value, ok, err)
	}
}
