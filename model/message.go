package model

var (
	Success      = "0000" //成功
	ServiceError = "0001" //服務暫停使用
	DataTypeErr  = "0002" //格式錯誤
	AuthErr      = "0003" //驗證失敗
	TokenDBErr   = "0004" //驗證DB錯誤
	TokenLockErr = "0005" //Token被鎖
	PointError   = "0006" //點數不足
)

type MessageInfo struct {
	Type int    //0斷線 1已建立連線 2對話 3送禮 4主播切換 5主播離線 6送禮結果
	CID  string `json:"-"`

	UID   string `json:"-"`
	Point int    `json:"-"`

	Data interface{}
}

type StatusInfo struct {
	CID   string
	IsVIP string
	Point int
}

type ErrorMessageInfo struct {
	Code string
}
