package server

type Session struct {
	Active      bool
	User        string
	Group       string
	UID         int
	GID         int
	IsRoot      bool
	PartitionID string
}

var CurrentSession = Session{}
