package domain

import "errors"

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

var ErrDocumentNotFound = errors.New("document not found")
