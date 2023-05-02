package entities

import "time"

type Message struct {
	Time     time.Time `json:"-"`
	Messages []string  `json:"messages"`
	Total    int       `json:"-"`
}
