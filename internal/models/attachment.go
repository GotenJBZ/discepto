package models

import "net/url"

type Attachment struct {
	ID         int
	Expanded   bool
	URL        url.URL
	AttachedTo Essay
}
