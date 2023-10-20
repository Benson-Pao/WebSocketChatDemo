package service

import (
	"IOCopyByWS/db"
	"IOCopyByWS/model"
	"encoding/json"
	"errors"
	"log"
	"strconv"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

var (
	poolCount   = 15
	queueLen    = 1000
	Config      *model.ConfigInfo
	ResourceObj *model.ResourceObject
	VerifyData  sync.Map

	BlockData sync.Map
	BlockLock sync.RWMutex
)

type ConnInfo struct {
	Service *ServiceInfo
	Chat    *UserListInfo
	User    *UserListInfo
	//ChatCh  chan interface{}
	IsClose bool
	sync.RWMutex
}

type UserListInfo struct {
	sync.RWMutex `json:"-"`
	Count        int
	List         sync.Map
	CreateCh     chan interface{} `json:"-"`
	ChangeCh     chan interface{} `json:"-"`
	CloseCh      chan interface{} `json:"-"`
	MsgCh        chan interface{} `json:"-"`
	IsClose      bool             `json:"-"`
}

type ChatInfo struct {
	CID          string
	Name         string
	Conn         *websocket.Conn `json:"-"`
	sync.RWMutex `json:"-"`
	Status       int //0 離線 1連線中 2已連線
	Users        sync.Map
	Time         time.Time
	Count        int
}

func (c *ChatInfo) Close() {
	c.RWMutex.Lock()
	defer c.RWMutex.Unlock()
	if c.Status > 0 {
		c.Conn.Close()
		c.Status = 0
	}

}

func (c *ChatInfo) GetChatStatus() int {
	c.RWMutex.RLock()
	defer c.RWMutex.RUnlock()
	return c.Status
}

func (c *ChatInfo) SetChatStatus(status int) {
	c.RWMutex.Lock()
	defer c.RWMutex.Unlock()

	c.Status = status

}

type UserInfo struct {
	sync.RWMutex `json:"-"`
	UID          string `json:"MemberID"`
	Name         string `json:"string"`
	Token        string `json:"Token"`
	ClientIP     string

	VIPTime   model.TimeInfo `json:"-"`
	IsVIP     string         `json:"IsVIP"`
	CheckTime model.TimeInfo `json:"-"`
	CID       string
	Status    int `json:"-"` //0己離線 1連線請求中 2已連線 3己閒置 4驗證中
	Point     int `json:"-"`

	Conn *websocket.Conn  `json:"-"`
	Ch   chan interface{} `json:"-"`
}

func (u *UserInfo) GetStatus() int {
	u.RLock()
	defer u.RUnlock()
	return u.Status
}

func (u *UserInfo) SetStatus(status int) int {
	u.Lock()
	defer u.Unlock()
	u.Status = status
	if status == 0 {
		u.Conn.Close()
	}
	return u.Status
}

func NewChat(s *ServiceInfo, blockCloseCh <-chan bool) *ConnInfo {
	ret := &ConnInfo{
		Service: s,
		Chat: &UserListInfo{
			Count:    0,
			CreateCh: make(chan interface{}, queueLen), //建房間用
			ChangeCh: make(chan interface{}, 0),
			CloseCh:  make(chan interface{}, 0),
			MsgCh:    make(chan interface{}, queueLen), //廣播用
			IsClose:  false,
		},
		User: &UserListInfo{
			Count:    0,
			CreateCh: make(chan interface{}, queueLen), //建使用者
			ChangeCh: make(chan interface{}, 0),
			CloseCh:  make(chan interface{}, queueLen), //關使用者
			MsgCh:    make(chan interface{}, queueLen), //處理訊息和送禮
			IsClose:  false,
		},
		//ChatCh:  make(chan interface{}, 100),
		IsClose: false,
	}
	Config = s.Config

	for i := 0; i < poolCount; i++ {
		go ret.CreateUserWorker()
	}

	for i := 0; i < poolCount; i++ {
		go ret.CloseUserWorker()
	}

	for i := 0; i < poolCount; i++ {
		go ret.ChatBroadcastMsg()
	}

	for i := 0; i < poolCount; i++ {
		go ret.UserBroadcastMsg()
	}

	go s.CreateChatWorkerSource(ret.Chat.CreateCh)

	go func() {
		for v := range ret.Chat.CreateCh {
			if !ret.Service.GetIsClose() {
				temp := v.(*model.ResourceObject)
				if temp.AuthAPIErr == nil {
					if temp.ChatItemsErr == nil {
						ResourceObj = temp
						for _, item := range ResourceObj.ChatItems {
							_, ok := ret.Chat.List.Load(item.Id)
							if !ok {
								go ret.CreatChatConn(item.Id, item.UserName)
							}

						}

					} else {
						log.Println("[Error]:", temp.ChatItemsErr)
					}
				} else {
					log.Println("[Error]:", temp.AuthAPIErr)
				}

			}

		}
	}()

	go func() {
		//每小時檢查一次清除一週完全沒用到的記錄

		ticker := time.NewTicker(1 * time.Hour)
		defer ticker.Stop()
		for {
			select {
			case _, ok := <-blockCloseCh:
				if !ok {
					log.Println("Block Timeout check Stop")
					return
				}
			case <-ticker.C:

				log.Println("Block Timeout check")

				BlockData.Range(func(key, value any) bool {

					block := value.(*model.BlockInfo)
					if time.Now().Sub(block.Time).Hours() > 24*7 {
						if tokenInfo, ok := VerifyData.Load(key); ok {
							token := tokenInfo.(*model.TokenInfo)
							if time.Now().Sub(token.Time).Hours() > 24*7 {
								VerifyData.Delete(key)
								BlockData.Delete(key)
							}
						}
					} else {
						block.IPList.Range(func(key, value any) bool {
							iplock := value.(*model.LockInfo)
							if time.Now().Sub(iplock.Time).Hours() > 24*7 {
								if !iplock.IsOK {
									block.IPList.Delete(key)
								}
							}
							return true
						})
					}

					return true
				})
			}
		}
	}()
	return ret

}

func (s *ConnInfo) CreatChatConn(id int, userName string) {
	defer func() {
		if r := recover(); r != nil {
			log.Println("CreatChatConn panic: ", r)
		}
	}()

	//log.Println(s.Chat.Count)
	_, ok := s.Chat.List.Load(strconv.Itoa(id))
	if ok {
		//log.Println(strconv.Itoa(id))
		return
	}

	c, _, err := websocket.DefaultDialer.Dial(ResourceObj.AuthAPI, nil)
	if err != nil {
		log.Println("[Error] dial:", err)
		return
	}
	defer c.Close()

	chatItem := &ChatInfo{
		CID:    "0",
		Status: 1,
		Time:   time.Now(),
		Count:  0,
	}
	st := time.Now().Unix()
	for {
		c.UnderlyingConn().SetDeadline(time.Now().Add(30 * time.Second))
		_, msg, err := c.ReadMessage()

		if err != nil {
			log.Println("read:", err)

			if chatItem.CID != "0" && chatItem.GetChatStatus() > 0 {

				chatItem.SetChatStatus(0)
				resdata := map[string]interface{}{}
				resdata["CID"] = chatItem.CID
				msgSend := &model.MessageInfo{Type: 5, CID: chatItem.CID, Data: resdata}
				s.Chat.MsgCh <- msgSend

			}

			return
		}
		//log.Printf("receive: %s\n", msg)
		msgData := make(map[string]interface{}, 0)
		err = json.Unmarshal(msg, &msgData)
		if err != nil {
			log.Println("[Error]: ", err)
			continue
		}

		if v, ok := msgData["subscriptionKey"].(string); ok {
			switch v {
			case "connected":
				if params, ok := msgData["params"].(map[string]interface{}); ok {
					if clientId, ok := params["clientId"].(string); ok {
						sendObj := make(map[string]interface{})
						sendObj["id"] = strconv.FormatInt(st, 10) + "-sub-newChatMessage:" + strconv.Itoa(id)
						sendObj["method"] = "PUT"
						sendObj["url"] = "/front/clients/" + clientId + "/subscriptions/newChatMessage:" + strconv.Itoa(id)
						bytes, err := json.Marshal(sendObj)
						if err != nil {
							log.Println("[Error]: ", err)
							c.Close()
							return
						}

						if chatItem.CID == "0" {

							s.Chat.Lock()
							chatItem.CID = strconv.Itoa(id)
							chatItem.Name = userName
							old, ok := s.Chat.List.LoadAndDelete(chatItem.CID)
							if ok {
								v, _ := old.(*ChatInfo)
								v.Close()
								s.Chat.Count--
							}
							chatItem.SetChatStatus(2)
							s.Chat.List.Store(chatItem.CID, chatItem)
							s.Chat.Count++
							s.Chat.Unlock()

							//log.Println(s.Chat.Count)
						}

						c.WriteMessage(websocket.TextMessage, bytes)

					}

				}

			case "newChatMessage:" + strconv.Itoa(id):
				obj := make(map[string]interface{})
				//log.Println(msgData)
				//obj.params.message.type

				data := &model.MessageInfo{
					Type: 0,
					CID:  strconv.Itoa(id),
				}
				if params, ok := msgData["params"].(map[string]interface{}); ok {
					if message, ok := params["message"].(map[string]interface{}); ok {
						if details, ok := message["details"].(map[string]interface{}); ok {

							//test, _ := json.Marshal(details)
							//log.Println(string(test))

							if body, ok := details["body"]; ok {
								obj["Text"] = body
							}

							if amount, ok := details["amount"]; ok {
								obj["Point"] = amount
								//log.Println(reflect.TypeOf(details["amount"]).String())
							}

						}

						if userData, ok := message["userData"].(map[string]interface{}); ok {
							if username, ok := userData["username"].(string); ok {
								obj["UserName"] = username
							}
						}

						if t, ok := message["type"].(string); ok {
							switch t {
							case "text":
								data.Type = 2

								obj["CID"] = chatItem.CID
								if userData, ok := message["userData"].(map[string]interface{}); ok {
									if username, ok := userData["username"]; ok {
										obj["UserName"] = username
									}
									if isModel, ok := userData["isModel"]; ok {
										obj["IsModel"] = isModel
									}

									//data.Data = obj
									//bytes, _ := json.Marshal(data)
									//log.Println("[text]", string(bytes))
									//c.WriteMessage(websocket.TextMessage, bytes)
								}

							case "tip":
								data.Type = 3

								obj["CID"] = chatItem.CID
								if userData, ok := message["userData"].(map[string]interface{}); ok {
									if username, ok := userData["username"]; ok {
										obj["UserName"] = username
									}
									if isModel, ok := userData["isModel"]; ok {
										obj["IsModel"] = isModel
									}

									//data.Data = obj
									//bytes, _ := json.Marshal(data)
									//log.Println("[tip]", string(bytes))
									//log.Println(string(msg))
									//c.WriteMessage(websocket.TextMessage, bytes)
								}

							}
						}

					}

				}

				if data.Type == 2 || data.Type == 3 {
					data.Data = obj
					/*
						bytes, _ := json.Marshal(data)
						typeMsg := ""
						if data.Type == 2 {
							typeMsg = "[text]"
						} else {
							typeMsg = "[tip]"
						}
						log.Println(typeMsg, string(bytes))
					*/
					if !s.Chat.IsClose {
						s.Chat.MsgCh <- data
					}

				}

			}
		}

	}
}

func (s *ConnInfo) ChatBroadcastMsg() {
	for v := range s.Chat.MsgCh {
		info := v.(*model.MessageInfo)
		switch info.Type {
		case 2:
			fallthrough
		case 3:
			fallthrough
		case 5:
			if data, ok := s.Chat.List.Load(info.CID); ok {
				c := data.(*ChatInfo)
				c.Users.Range(func(key, value any) bool {
					if uid, ok := value.(string); ok {
						if user, ok := s.User.List.Load(uid); ok {
							u, _ := user.(*UserInfo)

							if info.Type == 5 {

								resData := info.Data.(map[string]interface{})
								resData["IsVIP"] = u.IsVIP
								resData["Point"] = u.Point

								u.Lock()
								WriteMessageInfo(u.Conn, info)
								u.Unlock()

								u.SetStatus(3)
								s.ChatUserDelete(u)
							} else {
								u.Lock()
								WriteMessageInfo(u.Conn, info)
								u.Unlock()
							}

						}

					}
					return true
				})

				if info.Type == 5 {
					s.Chat.Lock()
					s.Chat.List.Delete(c.CID)
					if s.Chat.Count > 0 {
						s.Chat.Count--
					}
					s.Chat.Unlock()

					log.Println("[Chat close]: ", c.CID)
					log.Println("[Chat count]:", s.Chat.Count)
				}

			}
		}

	}
}

func (s *ConnInfo) UserBroadcastMsg() {
	for v := range s.User.MsgCh {
		info := v.(*model.MessageInfo)

		switch info.Type {
		case 2: //發話
			if data, ok := s.Chat.List.Load(info.CID); ok {
				c := data.(*ChatInfo)
				c.Users.Range(func(key, value any) bool {
					if !s.Chat.IsClose {
						s.Chat.MsgCh <- info
					}
					return true
				})
			}
		case 3: //送禮
			/*Demo用 送禮註解
			if data, ok := s.Chat.List.Load(info.CID); ok {

				c := data.(*ChatInfo)
				uid, _ := strconv.Atoi(info.UID)
				cid, _ := strconv.Atoi(info.CID)

				result := &model.MessageInfo{
					Type: 6,
				}
				obj := map[string]interface{}{}
				obj["CID"] = info.CID


				tokeninfo, _ := VerifyData.Load(info.UID)
				token := tokeninfo.(*model.TokenInfo)

				//與暫存比較點數(暫存一分內的更新記錄)
				if p, err := token.PointInfo.GetPoint(); err == nil {
					if p < info.Point {
						obj["Point"] = p
						obj["IsVIP"] = token.VIPInfo.IsVIP
						obj["Code"] = model.PointError
						result.Data = obj

						if u, ok := s.User.List.Load(info.UID); ok {
							user, _ := u.(*UserInfo)

							user.Lock()
							WriteMessageInfo(user.Conn, result)
							user.Unlock()
						}

						continue
					}
				}

				res, err := db.SetPoint(uid, info.Point, cid, c.Name)
				if err != nil {
					log.Println(err)
				}
				Cinfo := &model.ConsumeResultInfo{}
				db.ReflectField(res, Cinfo)

				switch Cinfo.Result {
				case -1: //沒使用者資料
					obj["Point"] = 0
					obj["IsVIP"] = "0"
					obj["Code"] = model.PointError

					token.PointInfo.SetPoint(0)
					token.VIPInfo.IsVIP = "0"

				case 1: //扣點成功
					obj["Point"] = Cinfo.LivePoint
					obj["IsVIP"] = strconv.FormatInt(Cinfo.IsLiveVIP, 10)
					obj["Code"] = model.Success

					token.PointInfo.SetPoint(int(Cinfo.LivePoint))
					token.VIPInfo.IsVIP = strconv.FormatInt(Cinfo.IsLiveVIP, 10)

					if !s.Chat.IsClose {
						s.Chat.MsgCh <- info
					}

				case 0: //點數不足
					obj["Point"] = Cinfo.LivePoint
					obj["IsVIP"] = strconv.FormatInt(Cinfo.IsLiveVIP, 10)
					obj["Code"] = model.PointError

					token.PointInfo.SetPoint(int(Cinfo.LivePoint))
					token.VIPInfo.IsVIP = strconv.FormatInt(Cinfo.IsLiveVIP, 10)
				}



				result.Data = obj



				if u, ok := s.User.List.Load(info.UID); ok {
					user, _ := u.(*UserInfo)

					user.Lock()

					if itm, ok := obj["Point"].(int64); ok {
						user.Point = int(itm)
					}

					if itm, ok := obj["IsVIP"].(string); ok {
						user.IsVIP = itm
					}

					user.CheckTime.CheckTime = time.Now()

					user.CheckTime.EndTime = Cinfo.LiveEndTime

					WriteMessageInfo(user.Conn, result)
					user.Unlock()

				}


			}
			*/
		}
	}
}

func (s *UserListInfo) Close() {
	s.Lock()
	defer s.Unlock()
	s.IsClose = true
	close(s.CreateCh)
	close(s.ChangeCh)
	close(s.CloseCh)
	close(s.MsgCh)
}

func (s *ConnInfo) Close() {
	s.Lock()
	defer s.Unlock()
	s.IsClose = false
	//close(s.ChatCh)
	s.Chat.Close()
	s.User.Close()
}

func (s *ConnInfo) CloseUserWorker() {
	for ch := range s.User.CloseCh {

		uid := ch.(string)
		v, ok := s.User.List.Load(uid)
		if ok {
			userInfo := v.(*UserInfo)

			s.UserDelete(userInfo)

			log.Println("Delete User:", uid)
		}

	}
}

// 檢核全域Token
func VerifyToken(UID string, Token string) (bool, error) {
	if v, ok := VerifyData.Load(UID); ok {
		token := v.(*model.TokenInfo)
		if token.Token == Token {
			token.Time = time.Now()
			return true, nil
		} else {
			if time.Now().Sub(token.Time).Minutes() < 3 {
				return false, nil
			} else {
				return false, errors.New("Token Time Out")
			}
		}
	} else {
		return false, errors.New("Token Not Found")
	}
}

// 取得db Token刷新全域Token
func (u *UserInfo) SetVerify(b *model.BlockInfo) error {
	b.Lock()
	defer b.Unlock()

	uid, _ := strconv.Atoi(u.UID)
	status := u.GetStatus()

	switch status {
	case 0:
		return errors.New("Connection Disconnect")
	case 4:
		return errors.New("Work Processing")
	default:
		u.SetStatus(4)
		data, err := db.GetUserInfo(uid)
		if err != nil {
			//沒資料即產生新的(沒資料也會有err 依錯誤字串判斷)
			if err.Error() == "Data Not Found" {
				VerifyData.Store(u.UID, &model.TokenInfo{
					UID:   u.UID,
					Token: err.Error(), //沒有就隨便塞
					Time:  time.Now(),
					PointInfo: &model.PointInfo{
						Point: 0,
						Time:  time.Now(),
						State: 2,
					},
					VIPInfo: &model.VIPInfo{
						EndTime: time.Now().Add(-1),
						Time:    time.Now(),
						State:   2,
					},
				})
				u.Point = 0
				u.SetStatus(status)
				//沒登入過則當被拒
				return errors.New("Denied")

			} else {
				log.Println(err)
				u.SetStatus(status)
				return errors.New("Data Work Error")
			}
		} else {
			if v, ok := data.(map[string]interface{}); ok {
				info := &model.TokenInfo{
					Time: time.Now(),
					PointInfo: &model.PointInfo{
						Point: 0,
						Time:  time.Now(),
						State: 1,
					},
					VIPInfo: &model.VIPInfo{
						EndTime: time.Now().Add(-1),
						Time:    time.Now(),
						State:   1,
					},
				}
				//Token	LivePoint	LiveEndTime	IsLiveVIP

				if token, ok := v["Token"].(string); ok {
					info.Token = token
					info.Time = time.Now()
					//VerifyData.Store(u.UID, info)
				}

				if p, ok := v["LivePoint"].(int64); ok {
					info.PointInfo.Point = int(p)
					u.Point = info.PointInfo.Point
				}

				if isvip, ok := v["IsLiveVIP"].(int64); ok {
					info.VIPInfo.IsVIP = strconv.FormatInt(isvip, 10)
					u.IsVIP = info.VIPInfo.IsVIP
				}

				if endtime, ok := v["LiveEndTime"].(time.Time); ok {
					u.VIPTime.EndTime = endtime
					u.VIPTime.CheckTime = time.Now()
					info.VIPInfo.EndTime = endtime
				}

				info.PointInfo.SetState(2)

				info.VIPInfo.SetState(2)

				VerifyData.Store(u.UID, info)

			}

			u.SetStatus(status)
			return nil
		}

	}
}

func (u *UserInfo) Verify(b *model.BlockInfo) error {

	if ok, err := VerifyToken(u.UID, u.Token); err != nil {

		return u.SetVerify(b)
	} else {
		if !ok {
			return errors.New("Denied")
		} else {

			return u.SetVerify(b)
		}
	}
}

func (s *ConnInfo) CreateUserWorker() {

	for ch := range s.User.CreateCh {

		userInfo := ch.(*UserInfo)
		/*  Demo用把驗證註解
		info, _ := BlockData.LoadOrStore(userInfo.UID, model.CreateBlockInfo(userInfo.UID, userInfo.ClientIP))
		blockInfo := info.(*model.BlockInfo)
		lockInfo, _ := blockInfo.IPList.Load(userInfo.ClientIP)
		if !blockInfo.Check(false) {

			if !lockInfo.(*model.LockInfo).IsOK {
				userInfo.Lock()
				WriteError(userInfo.Conn, model.TokenLockErr)
				userInfo.Unlock()
				continue
			}
		}

		err := userInfo.Verify(blockInfo)
		if err != nil {
			switch err.Error() {
			case "Connection Disconnect":
				continue
			case "Denied":
				blockInfo.Check(true)
				lockInfo.(*model.LockInfo).Set(false)
				userInfo.Ch <- false
				continue
			case "Work Processing":
				continue
			case "Data Work Error":
				userInfo.Lock()
				WriteError(userInfo.Conn, model.TokenDBErr)
				userInfo.Unlock()
				continue

			}
		}
		*/

		if old, ok := s.User.List.Load(userInfo.UID); ok {
			olduser := old.(*UserInfo)
			s.UserDelete(olduser)
		}
		userInfo.SetStatus(2)

		s.UserAdd(userInfo)
		s.ChatUserAdd(userInfo)

		//lockInfo.(*model.LockInfo).Set(true)

		userInfo.Ch <- true

	}
}

func (c *ConnInfo) ChatUserAdd(u *UserInfo) {
	if v, ok := c.Chat.List.Load(u.CID); ok {
		chat := v.(*ChatInfo)
		if chat.GetChatStatus() > 0 {
			if _, ok := chat.Users.LoadOrStore(u.UID, u.UID); !ok {
				chat.Lock()
				chat.Count++
				chat.Unlock()

			}
			u.SetStatus(2)

			u.Lock()
			WriteMsg(u.Conn, 1, model.StatusInfo{CID: u.CID, IsVIP: u.IsVIP, Point: u.Point})
			u.Unlock()
		} else {
			u.SetStatus(3)

			u.Lock()
			WriteMsg(u.Conn, 5, model.StatusInfo{CID: u.CID, IsVIP: u.IsVIP, Point: u.Point})
			u.Unlock()
		}
	} else {
		u.SetStatus(3)

		u.Lock()
		WriteMsg(u.Conn, 5, model.StatusInfo{CID: u.CID, IsVIP: u.IsVIP, Point: u.Point})
		u.Unlock()
	}
}

func (info *ConnInfo) UserAdd(user *UserInfo) {
	info.User.Lock()
	defer info.User.Unlock()
	info.User.List.Store(user.UID, user)
	info.User.Count++
}

func (info *ConnInfo) UserDelete(user *UserInfo) {
	info.User.Lock()
	defer info.User.Unlock()
	info.ChatUserDelete(user)

	v, ok := info.User.List.LoadAndDelete(user.UID)
	if ok {

		v.(*UserInfo).SetStatus(0)
		if info.User.Count > 0 {
			info.User.Count--
		}
	}

}

func (info *ConnInfo) ChatUserDelete(user *UserInfo) {
	info.Chat.Lock()
	defer info.Chat.Unlock()
	if v, ok := info.Chat.List.Load(user.CID); ok {
		chat := v.(*ChatInfo)
		chat.Lock()
		chat.Users.Delete(user.UID)
		if chat.Count > 0 {
			chat.Count--
		}
		chat.Unlock()
	}
}

func WriteError(c *websocket.Conn, errCode string) {

	bytes, _ := json.Marshal(model.MessageInfo{
		Type: 0,
		Data: model.ErrorMessageInfo{
			Code: errCode,
		},
	})
	c.WriteMessage(websocket.TextMessage, bytes)
}

func WriteMsg(c *websocket.Conn, Type int, data interface{}) {

	bytes, _ := json.Marshal(model.MessageInfo{
		Type: Type,
		Data: data,
	})
	c.WriteMessage(websocket.TextMessage, bytes)
}

func WriteMessageInfo(c *websocket.Conn, data *model.MessageInfo) {

	bytes, _ := json.Marshal(data)
	c.WriteMessage(websocket.TextMessage, bytes)
}
