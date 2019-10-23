package redisstore

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-redis/redis/v7"
	"github.com/gorilla/sessions"
)

const (
	redisAddr = "localhost:6379"
	secretKet = "secret-key"
)

func TestNew(t *testing.T) {
	store, err := NewRedisStoreWithRedisConfig(&redis.Options{
		Addr: redisAddr,
	}, []byte(secretKet))
	if err != nil {
		t.Fatal("failed to create redis store", err)
	}
	defer store.Close()

	req, err := http.NewRequest("GET", "http://www.example.com", nil)
	if err != nil {
		t.Fatal("failed to create request", err)
	}

	session, err := store.New(req, "hello")
	if err != nil {
		t.Fatal("failed to create session", err)
	}
	if session.IsNew == false {
		t.Fatal("session is not new")
	}
}

func TestOptions(t *testing.T) {

	store, err := NewRedisStoreWithRedisConfig(&redis.Options{
		Addr: redisAddr,
	}, []byte(secretKet))
	if err != nil {
		t.Fatal("failed to create redis store", err)
	}
	defer store.Close()

	opts := sessions.Options{
		Path:   "/path",
		MaxAge: 99999,
	}
	store.Options(opts)

	req, err := http.NewRequest("GET", "http://www.example.com", nil)
	if err != nil {
		t.Fatal("failed to create request", err)
	}

	session, err := store.New(req, "hello")
	if session != nil &&
		session.Options != nil &&
		(session.Options.Path != opts.Path || session.Options.MaxAge != opts.MaxAge) {
		t.Fatal("failed to set options")
	}
}

func TestSave(t *testing.T) {
	store, err := NewRedisStoreWithRedisConfig(&redis.Options{
		Addr: redisAddr,
	}, []byte(secretKet))
	if err != nil {
		t.Fatal("failed to create redis store", err)
	}
	defer store.Close()

	req, err := http.NewRequest("GET", "http://www.example.com", nil)
	if err != nil {
		t.Fatal("failed to create request", err)
	}
	w := httptest.NewRecorder()

	session, err := store.New(req, "hello")
	if err != nil {
		t.Fatal("failed to create session", err)
	}

	session.Values["key"] = "value2"
	err = session.Save(req, w)
	if err != nil {
		t.Fatal("failed to save: ", err)
	}
}

func TestSaveSimple(t *testing.T) {
	store, err := NewRedisStoreSimple(redisAddr, ``, 10, []byte(``))
	if err != nil {
		t.Fatal("failed to create redis store", err)
	}
	defer store.Close()

	req, err := http.NewRequest("GET", "http://www.example.com", nil)
	if err != nil {
		t.Fatal("failed to create request", err)
	}
	w := httptest.NewRecorder()

	session, err := store.New(req, "hello2")
	if err != nil {
		t.Fatal("failed to create session", err)
	}

	session.Values["key"] = "value3"
	err = session.Save(req, w)
	if err != nil {
		t.Fatal("failed to save: ", err)
	}
}

func TestDelete(t *testing.T) {

	store, err := NewRedisStoreWithRedisConfig(&redis.Options{
		Addr: redisAddr,
	}, []byte(secretKet))
	if err != nil {
		t.Fatal("failed to create redis store", err)
	}
	defer store.Close()

	req, err := http.NewRequest("GET", "http://www.example.com", nil)
	if err != nil {
		t.Fatal("failed to create request", err)
	}
	w := httptest.NewRecorder()
	session, err := store.New(req, "hello")
	if err != nil {
		t.Fatal("failed to create session", err)
	}

	session.Values["key"] = "value2"

	err = session.Save(req, w)
	if err != nil {
		t.Fatal("failed to save session: ", err)
	}

	session.Options.MaxAge = -1
	err = session.Save(req, w)
	if err != nil {
		t.Fatal("failed to delete session: ", err)
	}
}

func TestDeleteNative(t *testing.T) {

	store, err := NewRedisStoreWithRedisConfig(&redis.Options{
		Addr: redisAddr,
	}, []byte(secretKet))
	if err != nil {
		t.Fatal("failed to create redis store", err)
	}
	defer store.Close()

	req, err := http.NewRequest("GET", "http://www.example.com", nil)
	if err != nil {
		t.Fatal("failed to create request", err)
	}
	w := httptest.NewRecorder()
	session, err := store.New(req, "hello")
	if err != nil {
		t.Fatal("failed to create session", err)
	}

	session.Values["key"] = "value2"

	err = session.Save(req, w)
	if err != nil {
		t.Fatal("failed to save session: ", err)
	}

	err = store.Delete(req, w, session)
	if err != nil {
		t.Fatal("failed to delete session: ", err)
	}
}

func TestFlashes(t *testing.T) {
	store, err := NewRedisStoreWithRedisConfig(&redis.Options{
		Addr: redisAddr,
	}, []byte(secretKet))
	if err != nil {
		t.Fatal("failed to create redis store", err)
	}
	defer store.Close()

	req, err := http.NewRequest("GET", "http://www.example.com", nil)
	if err != nil {
		t.Fatal("failed to create request", err)
	}
	w := httptest.NewRecorder()

	session, err := store.Get(req, "session-key")
	if err != nil {
		t.Fatalf("Error getting session: %v", err)
	}

	// Get a flash.
	flashes := session.Flashes()
	if len(flashes) != 0 {
		t.Errorf("Expected empty flashes; Got %v", flashes)
	}
	// Add some flashes.
	session.AddFlash("foo")
	session.AddFlash("bar")
	// Custom key.
	session.AddFlash("baz", "custom_key")

	// Save.
	if err = sessions.Save(req, w); err != nil {
		t.Fatalf("Error saving session: %v", err)
	}
	hdr := w.Header()
	cookies, ok := hdr["Set-Cookie"]
	if !ok || len(cookies) != 1 {
		t.Fatalf("No cookies. Header: %s", hdr)
	}
}

func TestJSONSerializer(t *testing.T) {
	store, err := NewRedisStoreWithRedisConfig(&redis.Options{
		Addr: redisAddr,
	}, []byte(secretKet))
	if err != nil {
		t.Fatal("failed to create redis store", err)
	}
	defer store.Close()

	store.Serializer(JSONSerializer{})

	req, err := http.NewRequest("GET", "http://www.example.com", nil)
	if err != nil {
		t.Fatal("failed to create request", err)
	}
	w := httptest.NewRecorder()

	session, err := store.Get(req, "session-key")
	if err != nil {
		t.Fatalf("Error getting session: %v", err)
	}

	// Get a flash.
	flashes := session.Flashes()
	if len(flashes) != 0 {
		t.Errorf("Expected empty flashes; Got %v", flashes)
	}
	// Add some flashes.
	session.AddFlash("foo")

	// Save
	if err = sessions.Save(req, w); err != nil {
		t.Fatalf("Error saving session: %v", err)
	}
	hdr := w.Header()
	cookies, ok := hdr["Set-Cookie"]
	if !ok || len(cookies) != 1 {
		t.Fatalf("No cookies. Header: %s", hdr)
	}

	// Get a session.
	req.Header.Add("Cookie", cookies[0])
	if session, err = store.Get(req, "session-key"); err != nil {
		t.Fatalf("Error getting session: %v", err)
	}

	// Check all saved values.
	flashes = session.Flashes()
	if len(flashes) != 1 {
		t.Fatalf("Expected flashes; Got %v", flashes)
	}

	if flashes[0] != "foo" {
		t.Errorf("Expected foo,bar; Got %v", flashes)
	}
}
