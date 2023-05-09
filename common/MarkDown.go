package common
type MsgMarkDown struct {

	Msgtype  string   `json:"msgtype"`
	MarkDown MarkDown `json:"markdown"`
	At       At       `json:"at"`
	TaskID   uint     //我们次数使用的uint，而是没有使用foreignkey重写外键，所以此处指向的Task表中的Model中的ID
}
type MarkDown struct {

	Title         string `json:"title"`
	Text          string `json:"text"`
	MsgMarkDownID uint
}
