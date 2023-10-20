package server

import (
	"IOCopyByWS/db"
	"IOCopyByWS/model"
	"IOCopyByWS/service"
	"encoding/json"
	"log"
	"net"
	"net/http"
	"regexp"
	"strings"
	"sync"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

var (
	Config     *model.ConfigInfo
	ServerAuth = "0DHaAO1qCk6W"
	regNumber  = regexp.MustCompile("^[0-9]+$")
)

type Server struct {
	Route    *gin.Engine
	Port     string
	Listener net.Listener
	Service  *service.ServiceInfo
	Chat     *service.ConnInfo

	sync.RWMutex
}

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

func NewServer(s *service.ServiceInfo, c *service.ConnInfo) (*Server, error) {

	/*
		cert, err := tls.LoadX509KeyPair(s.Config.Csr, s.Config.Key)
		if err != nil {
			return nil, err
		}

		l, err := tls.Listen("tcp", ":"+s.Config.WSPort, &tls.Config{Certificates: []tls.Certificate{cert}})
		if err != nil {
			return nil, err
		}
	*/

	//l, err := net.Listen("tcp", ":"+s.Config.WSPort)
	//if err != nil {
	//	return nil, err
	//}
	Config = s.Config
	route := gin.Default()
	ret := &Server{
		Route: route,
		Port:  ":" + s.Config.WSPort,
		//Listener: l,
		Service: s,
		Chat:    c,
	}

	route.GET("/ws", ret.WSHandler)
	route.GET("/count", ret.GetCountHandler)
	route.GET("/users/:UID", ret.GetUsersHandler)
	route.GET("/chats", ret.GetChatsHandler)
	route.GET("/getEnable", ret.GetEnableHandler)
	route.GET("/setEnable/:IsRun", ret.SetEnableHandler)
	route.GET("/ReLoadDBConnString", ret.ReLoadDBConnStringHandler)

	go func() {

		//log.Println("WS listen On ", s.Config.WSPort)
		//ret.Serve()
		route.Run(ret.Port)
	}()

	return ret, nil
}

func GetClientIP(r *http.Request) string {
	forwardedFor := r.Header.Get("X-Forwarded-For")
	if forwardedFor != "" {

		ips := strings.Split(forwardedFor, ",")
		clientIP := strings.TrimSpace(ips[0])
		return clientIP
	}

	realIP := r.Header.Get("X-Real-IP")
	if realIP != "" {
		return realIP
	}

	clientIP := strings.Split(r.RemoteAddr, ":")[0]
	return clientIP
}

func (s *Server) WSHandler(ctx *gin.Context) {
	userInfo := &service.UserInfo{
		Status:   1,
		UID:      "0",
		ClientIP: GetClientIP(ctx.Request),
	}

	c, err := upgrader.Upgrade(ctx.Writer, ctx.Request, nil)
	if err != nil {
		log.Print("upgrade:", err)
		return
	}
	defer c.Close()

	if Config.IsRun == "0" {
		WriteError(c, model.ServiceError)
		return
	}

	for {
		_, message, err := c.ReadMessage()
		if err != nil {
			log.Println("[client close err]: ", err)

			if userInfo.UID != "0" {
				if userInfo.GetStatus() > 1 {
					s.Chat.User.CloseCh <- userInfo.UID
					log.Println("[client close]: ", userInfo.UID)
				}
			}

			break
		}

		log.Printf("recv: %s", message)

		msg := &model.MessageInfo{}
		err = json.Unmarshal(message, msg)
		if err != nil {
			log.Println("[Warning]:", err)
			continue
		}

		switch msg.Type {
		case 1: //建立連線

			if userInfo.GetStatus() > 1 {
				continue
			}

			userInfo.Conn = c

			item, ok := msg.Data.(map[string]interface{})
			if !ok {
				WriteError(c, model.DataTypeErr)
				return
			}
			userInfo.UID, ok = item["MemberID"].(string)
			if !ok {
				WriteError(c, model.DataTypeErr)
				return
			} else {
				if !regNumber.MatchString(userInfo.UID) {
					WriteError(c, model.DataTypeErr)
					return
				}
			}
			userInfo.CID, ok = item["CID"].(string)
			if !ok {
				WriteError(c, model.DataTypeErr)
				return
			} else {
				if !regNumber.MatchString(userInfo.CID) {
					WriteError(c, model.DataTypeErr)
					return
				}
			}
			userInfo.Token, ok = item["Token"].(string)
			if !ok {
				WriteError(c, model.DataTypeErr)
				return
			}

			userInfo.Name, ok = item["Name"].(string)
			if !ok {
				userInfo.Name = ""
			}

			userInfo.Ch = make(chan interface{})
			if !s.Chat.User.IsClose {
				s.Chat.User.CreateCh <- userInfo
			}

			v, ok := <-userInfo.Ch
			if ok {
				if !v.(bool) {
					WriteError(c, model.AuthErr)
					userInfo.SetStatus(0)
					close(userInfo.Ch)
					return
				}

			}
			close(userInfo.Ch)
		case 2: //發話
			//{"Type":2,"Data":{"CID":"111252860","IsModel":true,"Text":":bye:","UserName":"Jennie_Spa"}}
			if item, ok := msg.Data.(map[string]interface{}); ok {
				if userInfo.GetStatus() > 1 {
					if text, ok := item["Text"].(string); ok {
						sendmsg := &model.MessageInfo{
							Type: 2,
							CID:  userInfo.CID,
							UID:  userInfo.UID,
						}
						obj := map[string]interface{}{}
						obj["CID"] = userInfo.CID
						obj["IsModel"] = false
						obj["Text"] = text
						obj["UserName"] = userInfo.Name

						sendmsg.Data = obj
						if !s.Chat.User.IsClose {
							s.Chat.User.MsgCh <- sendmsg
						}
					}
				}
			}
		case 3: //送禮
			//{"Type":3,"Data":{"CID":"111252860","IsModel":false,"Point":200,"Text":"","UserName":"tieudao88"}}
			if item, ok := msg.Data.(map[string]interface{}); ok {
				if userInfo.GetStatus() > 1 {

					if point, ok := item["Point"].(float64); ok {

						sendmsg := &model.MessageInfo{
							Type:  3,
							CID:   userInfo.CID,
							UID:   userInfo.UID,
							Point: int(point),
						}

						obj := map[string]interface{}{}
						obj["CID"] = userInfo.CID
						obj["IsModel"] = false
						obj["Point"] = sendmsg.Point
						obj["Text"] = ""
						obj["UserName"] = userInfo.Name

						sendmsg.Data = obj

						if !s.Chat.User.IsClose {
							s.Chat.User.MsgCh <- sendmsg
						}
					}
				}
			}
		case 4: //主播切換
			if item, ok := msg.Data.(map[string]interface{}); ok {
				if userInfo.GetStatus() > 1 {
					if cid, ok := item["CID"].(string); ok {
						s.Chat.ChatUserDelete(userInfo)
						userInfo.CID = cid
						s.Chat.ChatUserAdd(userInfo)
					}

				}
			}

		}

		//err = c.WriteMessage(mt, message)
		if string(message) == "quit" {
			//break
			err = c.WriteMessage(websocket.CloseMessage, []byte("close"))
			if err != nil {
				log.Println("Error:", err)
				break
			}
		}

	}
}

func (s *Server) GetCountHandler(ctx *gin.Context) {
	if ctx.Request.Header.Get("Auth") != ServerAuth {
		return
	}
	res := map[string]int{}
	res["Chats"] = s.Chat.Chat.Count
	res["Users"] = s.Chat.User.Count
	bytes, _ := json.Marshal(res)
	ctx.Writer.Write(bytes)
}

func (s *Server) GetUsersHandler(ctx *gin.Context) {
	if ctx.Request.Header.Get("Auth") != ServerAuth {
		return
	}
	if ctx.Param("UID") == "-1" {
		res := make([]map[string]interface{}, 0)

		s.Chat.User.List.Range(func(key, value any) bool {
			u := value.(*service.UserInfo)
			item := map[string]interface{}{}
			item["UID"] = u.UID
			item["CID"] = u.CID
			item["Status"] = u.Status
			res = append(res, item)
			return true
		})
		bytes, _ := json.Marshal(res)
		ctx.Writer.Write(bytes)
	} else {
		item, ok := s.Chat.User.List.Load(ctx.Param("UID"))
		if ok {
			u := item.(*service.UserInfo)
			item := map[string]interface{}{}
			item["UID"] = u.UID
			item["CID"] = u.CID
			item["Status"] = u.Status
			bytes, _ := json.Marshal(item)
			ctx.Writer.Write(bytes)
		}

	}
}

func (s *Server) GetChatsHandler(ctx *gin.Context) {
	if ctx.Request.Header.Get("Auth") != ServerAuth {
		return
	}
	res := make([]map[string]interface{}, 0)

	s.Chat.Chat.List.Range(func(key, value any) bool {
		c := value.(*service.ChatInfo)
		item := map[string]interface{}{}
		item["CID"] = c.CID
		item["Users"] = c.Count
		item["Status"] = c.Status
		res = append(res, item)
		return true
	})
	bytes, _ := json.Marshal(res)
	ctx.Writer.Write(bytes)
}

func (s *Server) GetEnableHandler(ctx *gin.Context) {
	if ctx.Request.Header.Get("Auth") != ServerAuth {
		return
	}
	ctx.Writer.Write([]byte(Config.IsRun))
}

func (s *Server) SetEnableHandler(ctx *gin.Context) {
	if ctx.Request.Header.Get("Auth") != ServerAuth {
		return
	}
	Config.IsRun = ctx.Param("IsRun")
	ctx.Writer.Write([]byte(Config.IsRun))
}

func (s *Server) ReLoadDBConnStringHandler(ctx *gin.Context) {
	if ctx.Request.Header.Get("Auth") != ServerAuth {
		return
	}
	db.LoadConnectFile()
	ctx.Writer.WriteHeader(http.StatusOK)
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
