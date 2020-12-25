package models

import "net/url"

type Attachment struct {
	ID         int
	Expanded   bool
	Url        url.URL
	AttachedTo Essay
}
