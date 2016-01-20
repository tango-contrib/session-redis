session-redis [![Build Status](https://drone.io/github.com/tango-contrib/session-redis/status.png)](https://drone.io/github.com/tango-contrib/session-redis/latest) [![](http://gocover.io/_badge/github.com/tango-contrib/session-redis)](http://gocover.io/github.com/tango-contrib/session-redis)
======

Session-redis is a store of [session](https://github.com/tango-contrib/session) middleware for [Tango](https://github.com/lunny/tango) stored session data via [redis](http://redis.io). 

## Installation

    go get github.com/tango-contrib/session-redis

## Simple Example

```Go
package main

import (
    "github.com/lunny/tango"
    "github.com/tango-contrib/session"
    "github.com/tango-contrib/session-redis"
)

type SessionAction struct {
    session.Session
}

func (a *SessionAction) Get() string {
    a.Session.Set("test", "1")
    return a.Session.Get("test").(string)
}

func main() {
    o := tango.Classic()
    o.Use(session.New(session.Options{
        Store: redistore.New(redistore.Options{
                Host:    "127.0.0.1",
                DbIndex: 0,
                MaxAge:  30 * time.Minute,
            }),
        }))
    o.Get("/", new(SessionAction))
}
```

## Getting Help

- [API Reference](https://gowalker.org/github.com/tango-contrib/session-redis)

## License

This project is under BSD License. See the [LICENSE](LICENSE) file for the full license text.