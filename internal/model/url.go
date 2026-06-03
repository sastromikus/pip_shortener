package model

// URLItem describes a short URL record prepared for storage.
type URLItem struct {
	ID       string
	Original string
}

// UserURL describes a URL owned by a specific user.
type UserURL struct {
	ShortID  string
	Original string
}
