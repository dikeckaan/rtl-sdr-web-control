package db

import "time"

type User struct {
	ID           int    `json:"id"`
	Username     string `json:"username"`
	PasswordHash string `json:"-"`
	Role         string `json:"role"`
}

type PagerMessage struct {
	ID        int       `json:"id"`
	Timestamp time.Time `json:"timestamp"`
	MsgType   string    `json:"msg_type"`
	Address   string    `json:"address"`
	FuncCode  int       `json:"func_code"`
	Message   string    `json:"message"`
	Frequency string    `json:"frequency"`
	RawLine   string    `json:"raw_line,omitempty"`
}

type AISShip struct {
	MMSI        string    `json:"mmsi"`
	Name        string    `json:"name"`
	ShipType    int       `json:"ship_type"`
	Destination string    `json:"destination"`
	Lat         float64   `json:"lat"`
	Lon         float64   `json:"lon"`
	Speed       float64   `json:"speed"`
	Course      float64   `json:"course"`
	Heading     int       `json:"heading"`
	LastSeen    time.Time `json:"last_seen"`
	FirstSeen   time.Time `json:"first_seen"`
}

type GSMScan struct {
	ID        int       `json:"id"`
	Band      string    `json:"band"`
	Timestamp time.Time `json:"timestamp"`
	Result    string    `json:"result"`
}

type ISSCapture struct {
	ID        int       `json:"id"`
	Filename  string    `json:"filename"`
	Filepath  string    `json:"filepath"`
	Duration  int       `json:"duration"`
	Size      int64     `json:"size"`
	PassRise  time.Time `json:"pass_rise,omitempty"`
	PassSet   time.Time `json:"pass_set,omitempty"`
	MaxAlt    float64   `json:"max_alt,omitempty"`
	CreatedAt time.Time `json:"created_at"`
}

type ServiceState struct {
	ServiceID   string    `json:"service_id"`
	LastStarted time.Time `json:"last_started"`
	LastStopped time.Time `json:"last_stopped"`
	AutoStart   bool      `json:"auto_start"`
}
