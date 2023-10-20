package model

type ResourceInfo struct {
	Models []ResourceItemInfo `json:"models"`
}
type ResourceItemInfo struct {
	Id       int    `json:"id"`
	UserName string `json:"username"`
	IsModel  bool   `json:"isModel"`

	/*
		IsNonNude        bool `json:"isNonNude"`
		IsHD             bool `json:"isHd"`
		IsVR             bool `json:"isVr"`
		IsMobile         bool `json:"isMobile"`
		IsNew            bool `json:"isNew"`
		IsLive           bool `json:"isLive"`
		IsOnline         bool `json:"isOnline"`
		IsRecommended    bool `json:"isRecommended"`
		IsTagVerified    bool `json:"isTagVerified"`
		IsLovense        bool `json:"isLovense"`
		IsKiiroo         bool `json:"isKiiroo"`
		IsAvatarApproved bool `json:"isAvatarApproved"`
	*/
}

type ResourceObject struct {
	ChatItems    []ResourceItemInfo
	AuthAPI      string
	ChatItemsErr error
	AuthAPIErr   error
}
