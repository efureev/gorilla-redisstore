package redisstore

import (
	"encoding/base32"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/go-redis/redis/v7"
	"github.com/gorilla/securecookie"
	"github.com/gorilla/sessions"
)

// Amount of time for cookies/redis keys to expire.
const sessionExpire = 86400 * 30

// RedisStore stores gorilla sessions in Redis
type RedisStore struct {
	// client to connect to redis
	client redis.UniversalClient
	Codecs []securecookie.Codec
	// default options to use when a new session is created
	options sessions.Options
	// key prefix with which the session will be stored
	keyPrefix string
	// key generator
	keyGen KeyGenFunc
	// session serializer
	serializer    SessionSerializer
	maxLength     int
	DefaultMaxAge int // default Redis TTL for a MaxAge == 0 session
}

// KeyGenFunc defines a function used by store to generate a key
type KeyGenFunc func() string

// NewRedisStore returns a new RedisStore with default configuration
func NewRedisStore(client redis.UniversalClient, keyPairs ...[]byte) (*RedisStore, error) {
	rs := &RedisStore{
		options: sessions.Options{
			Path:   "/",
			MaxAge: sessionExpire,
		},
		Codecs:        securecookie.CodecsFromPairs(keyPairs...),
		client:        client,
		keyPrefix:     "session:",
		keyGen:        generateRandomKey,
		serializer:    GobSerializer{},
		maxLength:     0,
		DefaultMaxAge: 60 * 20, // 20 minutes seems like a reasonable default
	}

	return rs, rs.client.Ping().Err()
}

// NewRedisStoreWithRedisConfig returns a new RedisStore with default Redis configuration
func NewRedisStoreWithRedisConfig(options *redis.Options, keyPairs ...[]byte) (*RedisStore, error) {
	client := redis.NewClient(options)

	return NewRedisStore(client, keyPairs...)
}

// NewRedisStoreSimple returns a new RedisStore with easy config
func NewRedisStoreSimple(address, password string, db int, keyPairs ...[]byte) (*RedisStore, error) {
	opts := &redis.Options{
		Addr:     address,
		Password: password,
		DB:       db,
	}

	return NewRedisStoreWithRedisConfig(opts, keyPairs...)
}

// Get returns a session for the given name after adding it to the registry.
func (s *RedisStore) Get(r *http.Request, name string) (*sessions.Session, error) {
	return sessions.GetRegistry(r).Get(s, name)
}

// Close closes the underlying *redis client
func (s *RedisStore) Close() error {
	return s.client.Close()
}

// New returns a session for the given name without adding it to the registry.
func (s *RedisStore) New(r *http.Request, name string) (*sessions.Session, error) {
	session := sessions.NewSession(s, name)
	opts := s.options
	session.Options = &opts
	session.IsNew = true

	c, err := r.Cookie(name)
	if err != nil {
		return session, nil
	}
	session.ID = c.Value

	err = securecookie.DecodeMulti(name, c.Value, &session.ID, s.Codecs...)
	if err == nil {
		err = s.load(session)
		if err == nil {
			session.IsNew = false
		} else if err == redis.Nil {
			err = nil // no data stored
		}
	}

	return session, err
}

// Save adds a single session to the response.
//
// If the Options.MaxAge of the session is <= 0 then the session file will be
// deleted from the store. With this process it enforces the properly
// session cookie handling so no need to trust in the cookie management in the
// web browser.
func (s *RedisStore) Save(r *http.Request, w http.ResponseWriter, session *sessions.Session) error {
	// Delete if max-age is <= 0
	if session.Options.MaxAge <= 0 {
		if err := s.delete(session); err != nil {
			return err
		}
		http.SetCookie(w, sessions.NewCookie(session.Name(), "", session.Options))
		return nil
	}

	if session.ID == "" {
		id := s.keyGen()
		if id == `` {
			return errors.New(`redisstore: failed to generate session id`)
		}
		session.ID = id
	}
	if err := s.save(session); err != nil {
		return err
	}

	encoded, err := securecookie.EncodeMulti(session.Name(), session.ID, s.Codecs...)
	if err != nil {
		return err
	}

	http.SetCookie(w, sessions.NewCookie(session.Name(), encoded, session.Options))
	return nil
}

// Options set options to use when a new session is created
func (s *RedisStore) Options(opts sessions.Options) {
	s.options = opts
}

// KeyPrefix sets the key prefix to store session in Redis
func (s *RedisStore) KeyPrefix(keyPrefix string) {
	s.keyPrefix = keyPrefix
}

// SetMaxLength sets RediStore.maxLength if the `l` argument is greater or equal 0
// maxLength restricts the maximum length of new sessions to l.
// If l is 0 there is no limit to the size of a session, use with caution.
// The default for a new RediStore is 4096. Redis allows for max.
// value sizes of up to 512MB (http://redis.io/topics/data-types)
// Default: 4096,
func (s *RedisStore) SetMaxLength(l int) {
	if l >= 0 {
		s.maxLength = l
	}
}

// KeyGen sets the key generator function
func (s *RedisStore) KeyGen(f KeyGenFunc) {
	s.keyGen = f
}

// MaxAge restricts the maximum age, in seconds, of the session record
// both in database and a browser. This is to change session storage configuration.
// If you want just to remove session use your session `s` object and change it's
// `Options.MaxAge` to -1, as specified in
//    http://godoc.org/github.com/gorilla/sessions#Options
//
// Default is the one provided by this package value - `sessionExpire`.
// Set it to 0 for no restriction.
// Because we use `MaxAge` also in SecureCookie crypting algorithm you should
// use this function to change `MaxAge` value.
func (s *RedisStore) MaxAge(v int) {
	var c *securecookie.SecureCookie
	var ok bool
	s.options.MaxAge = v
	for i := range s.Codecs {
		if c, ok = s.Codecs[i].(*securecookie.SecureCookie); ok {
			c.MaxAge(v)
		} else {
			fmt.Printf("Can't change MaxAge on codec %v\n", s.Codecs[i])
		}
	}
}

// Serializer sets the session serializer to store session
func (s *RedisStore) Serializer(ss SessionSerializer) {
	s.serializer = ss
}

// save writes session in Redis
func (s *RedisStore) save(session *sessions.Session) error {
	b, err := s.serializer.Serialize(session)
	if err != nil {
		return err
	}

	if s.maxLength != 0 && len(b) > s.maxLength {
		return errors.New("sessionStore: the value to store is too big")
	}

	age := session.Options.MaxAge
	if age == 0 {
		age = s.DefaultMaxAge
	}

	return s.client.Set(s.keyPrefix+session.ID, b, time.Duration(age)*time.Second).Err()
}

// load reads session from Redis
func (s *RedisStore) load(session *sessions.Session) error {
	cmd := s.client.Get(s.keyPrefix + session.ID)
	if cmd.Err() != nil {
		return cmd.Err()
	}

	b, err := cmd.Bytes()
	if err != nil {
		return err
	}

	return s.serializer.Deserialize(b, session)
}

// delete deletes session in Redis
func (s *RedisStore) delete(session *sessions.Session) error {
	return s.client.Del(s.keyPrefix + session.ID).Err()
}

// SessionSerializer provides an interface for serialize/deserialize a session
type SessionSerializer interface {
	Serialize(s *sessions.Session) ([]byte, error)
	Deserialize(b []byte, s *sessions.Session) error
}

// generateRandomKey returns a new random key
func generateRandomKey() string {
	k := securecookie.GenerateRandomKey(64)
	if k == nil {
		return ``
	}

	return strings.TrimRight(base32.StdEncoding.EncodeToString(k), "=")
}
