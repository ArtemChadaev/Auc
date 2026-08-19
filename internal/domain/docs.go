package domain

type docs struct {
	id   int
	name string
	ver  int
	text textDocs
}
type textDocs struct {
	Title       string
	Description string
	Paragraph   []paragraph
}

type paragraph struct {
	Title *string
	Text  *string
}
