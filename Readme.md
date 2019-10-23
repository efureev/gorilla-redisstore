[![Codacy Badge](https://api.codacy.com/project/badge/Grade/0cdced379f3e41d39732a720263c8393)](https://app.codacy.com/app/efureev/hubMessage?utm_source=github.com&utm_medium=referral&utm_content=efureev/hubMessage&utm_campaign=Badge_Grade_Dashboard)
[![Build Status](https://travis-ci.org/efureev/hubMessage.svg?branch=master)](https://travis-ci.org/efureev/hubMessage)
[![Maintainability](https://api.codeclimate.com/v1/badges/82d6074b251f785f8c23/maintainability)](https://codeclimate.com/github/efureev/hubMessage/maintainability)
[![Test Coverage](https://api.codeclimate.com/v1/badges/82d6074b251f785f8c23/test_coverage)](https://codeclimate.com/github/efureev/hubMessage/test_coverage)
[![codecov](https://codecov.io/gh/efureev/hubMessage/branch/master/graph/badge.svg)](https://codecov.io/gh/efureev/hubMessage)
[![Go Report Card](https://goreportcard.com/badge/github.com/efureev/hubMessage)](https://goreportcard.com/report/github.com/efureev/hubMessage)

# RedisStore

## Install
```bash
go get -u github.com/efureev/redisstore
```

A [`Gorilla Sessions Store`](https://www.gorillatoolkit.org/pkg/sessions#Store) implementation backed by Redis.

It uses [`go-redis`](https://github.com/go-redis/redis) as client to connect to Redis.

## Example
```go

package main

import (
    "github.com/go-redis/redis/v7"
    "github.com/gorilla/sessions"
    "github.com/efureev/redisstore"
    "log"
    "net/http"
    "net/http/httptest"
)

func main() {

    client := redis.NewClient(&redis.Options{
        Addr:     "localhost:6379",
    })

    // New default RedisStore
    store, err := redisstore.NewRedisStore(client)
    if err != nil {
        log.Fatal("failed to create redis store: ", err)
    }

    // Example changing configuration for sessions
    store.KeyPrefix("session_")
    store.Options(sessions.Options{
        Path:   "/path",
        Domain: "example.com",
        MaxAge: 86400 * 60,
    })

    // Request y writer for testing
    req, _ := http.NewRequest("GET", "http://www.example.com", nil)
    w := httptest.NewRecorder()

    // Get session
    session, err := store.Get(req, "session-key")
    if err != nil {
        log.Fatal("failed getting session: ", err)
    }

    // Add a value
    session.Values["foo"] = "bar"

    // Save session
    if err = sessions.Save(req, w); err != nil {
        log.Fatal("failed saving session: ", err)
    }

    // Delete session (MaxAge <= 0)
    session.Options.MaxAge = -1
    if err = sessions.Save(req, w); err != nil {
        log.Fatal("failed deleting session: ", err)
    }
}

```
