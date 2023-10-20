package service

import (
	"IOCopyByWS/model"
	client "IOCopyByWS/net"
	"context"
	"encoding/json"
	"log"
	"sync"
	"time"
)

var (
	PoolSize = 15
	//configFile = "./config.json"
)

type ServiceInfo struct {
	sync.RWMutex
	WG     sync.WaitGroup
	Conns  sync.Map
	Config *model.ConfigInfo

	ResourceDataCh chan *model.ResourceObject
	Context        context.Context
	//Chat    *ConnInfo
	IsPause bool
	IsClose bool
}

func NewService(cxt context.Context, config *model.ConfigInfo) *ServiceInfo {
	ret := &ServiceInfo{
		Config:  config,
		Context: cxt,
		//Chat:    NewChat(),
		IsPause:        false,
		IsClose:        false,
		ResourceDataCh: make(chan *model.ResourceObject, 10),
	}

	//read resource
	go func() {
		for {
			select {
			case <-time.Tick(30 * time.Second):
				if !ret.IsPause {
					ret.Run()
				}
			case <-ret.Context.Done():
				log.Println("Read Resource Stop")
				ret.Close()
				return
			}

		}
	}()

	ret.Run()

	return ret
}

func (s *ServiceInfo) Close() {

	s.Lock()
	defer s.Unlock()
	s.IsClose = true
	close(s.ResourceDataCh)

}

func (s *ServiceInfo) GetIsClose() bool {

	s.RWMutex.RLock()
	defer s.RWMutex.RUnlock()
	return s.IsClose
}

func (s *ServiceInfo) Run() {
	v := &model.ResourceObject{AuthAPIErr: nil, ChatItemsErr: nil}
	err, authData := s.GetWSAuth()
	if err != nil {
		log.Println("[Error]:", err)
		v.AuthAPIErr = err
	} else {
		v.AuthAPI = authData.Data.WSURL
	}

	err, data := s.GetOPList()
	if err != nil {
		log.Println("[Error]:", err)
		v.ChatItemsErr = err
	} else {
		v.ChatItems = data.Models
	}

	if !s.GetIsClose() {
		s.ResourceDataCh <- v
	}

}

func (s *ServiceInfo) GetOPList() (error, *model.ResourceInfo) {
	body, err := client.Get(s.Config.ResourceURL)
	if err != nil {
		return err, nil
	}
	data := &model.ResourceInfo{}
	err = json.Unmarshal(body, data)
	if err != nil {
		return err, nil
	}
	return nil, data
}

func (s *ServiceInfo) GetWSAuth() (error, *model.WSAuthInfo) {
	body, err := client.Get(s.Config.ResourceAuth)
	if err != nil {
		return err, nil
	}
	data := &model.WSAuthInfo{}
	err = json.Unmarshal(body, data)
	if err != nil {
		return err, nil
	}
	return nil, data

}

func (s *ServiceInfo) GetConfig() model.ConfigInfo {
	s.RLock()
	defer s.RUnlock()
	return *s.Config
}

func (s *ServiceInfo) CreateChatWorkerSource(jobs chan<- interface{}) {
	for item := range s.ResourceDataCh {
		if !s.GetIsClose() {

			jobs <- item
		}
	}
}
