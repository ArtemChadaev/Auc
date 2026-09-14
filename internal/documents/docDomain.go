package documents

import (
	"errors"
	"net/netip"
	"time"
)

type Consents struct {
	UserID *int64
	AnonID *int64
	//ip net.IP
	Documents []Document
}

// version 0-9.0-99
type Document struct {
	ID      int64
	Name    string
	Version string
	Text    []Description
}

type Description struct {
	Title        *string
	Subtitle     *string
	Text         *string
	BulletList   []string
	NumberedList []NumberedList
}

type NumberedList struct {
	Text *string
	Line []NumberedList
}

type UserConsents struct {
	id        int64
	userID    int64
	action    string
	grantedAt time.Time
	ip        *netip.Addr
	source    string
}

// TODO: Если понадобится и когда будет точно известно какие типы, типизировать source в user_consents

var ErrDocumentNotFound = errors.New("document not found")
var ErrDocumentNotUserAnonID = errors.New("not have to save ID in user_consents")
