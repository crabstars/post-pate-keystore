package dbrepo

type UserEntry struct {
	UserId string `json:"userId"`
	Key    string `json:"key"`
	Iv     string `json:"iv"`
}
