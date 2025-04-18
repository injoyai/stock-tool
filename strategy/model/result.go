package model

type Result struct {
	Name string `json:"name"`
	Data *Data  `json:"data"`
}

type Data struct {
	Data   [][5]string `json:"data"`
	Points []*Point    `json:"markPoints"`
}

type Point struct {
	Index int    `json:"index"`
	Type  string `json:"type"`
}
