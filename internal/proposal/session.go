package proposal

import "encoding/json"

// SessionUserInfo 当前用户会话信息
type SessionUserInfo struct {
	Id       int32  `json:"id"`       // ID
	UserName string `json:"username"` // 用户名
	NickName string `json:"nickname"` // 昵称
}

// Marshal 序列化到JSON
func (user *SessionUserInfo) Marshal() (jsonRaw []byte) {
	jsonRaw, _ = json.Marshal(user)
	return
}
