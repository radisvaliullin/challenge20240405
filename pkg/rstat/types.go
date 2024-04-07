package rstat

type Resp struct {
	Header  Header
	Payload Payload
}

type Header struct {
	Used      int
	Remaining int
	Reset     int
}

type Payload struct {
	Data PayloadData `json:"data"`
}

type PayloadData struct {
	After    string     `json:"after"`
	Dist     int        `json:"dist"`
	Children []Children `json:"children"`
	Before   string     `json:"before"`
}

type Children struct {
	Data ChildData `json:"data"`
}

type ChildData struct {
	Title   string  `json:"title"`
	Name    string  `json:"name"`
	Ups     int     `json:"ups"`
	Author  string  `json:"author"`
	Created float64 `json:"created"`
}
