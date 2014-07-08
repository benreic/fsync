package main

type SetMetadata struct {
	SetId  string
	Photos []MediaMetadata
}

// Media metadata struct
type MediaMetadata struct {
	PhotoId  string
	Title    string
	Filename string
}
