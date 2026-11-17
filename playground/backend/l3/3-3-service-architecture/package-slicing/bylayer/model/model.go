// Package model holds every entity of the URL shortener, layered style.
package model

import "time"

type Link struct {
	Code string
	URL  string
}

type Click struct {
	Code string
	At   time.Time
}

type Count struct {
	Code  string
	Total int
}
