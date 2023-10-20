package model

import (
	"errors"
	"sync"
	"time"
)

type TokenInfo struct {
	UID   string    `json:"UID"`
	Token string    `json:"Token"`
	Time  time.Time `json:"-"`

	State        int `json:"-"` //0初始 1查詢中 2己確認 3己失效
	sync.RWMutex `json:"-"`

	PointInfo *PointInfo `json:"-"`
	VIPInfo   *VIPInfo   `json:"-"`
}

type BlockInfo struct {
	UID          string
	sync.RWMutex `json:"-"`
	IPList       sync.Map `json:"-"`
	Count        int
	Time         time.Time
}

func CreateBlockInfo(UID string, IP string) *BlockInfo {
	bl := &BlockInfo{
		UID:   UID,
		Time:  time.Now(),
		Count: 0,
	}
	bl.IPList.LoadOrStore(IP, CreateLockInfo(IP))
	return bl
}

func (info *BlockInfo) Reset() {
	info.Time = time.Now()
	info.Count = 0
}

func (info *BlockInfo) CheckTime() bool {
	return time.Now().Sub(info.Time).Seconds() < 300
}

func (info *BlockInfo) Check(isset bool) bool {
	info.Lock()
	defer info.Unlock()
	if isset {
		if info.CheckTime() {
			info.Count++
			return info.Count <= 10
		} else {
			info.Reset()
			info.Count++
			return true
		}
	} else {
		if info.CheckTime() {
			return info.Count <= 10
		} else {
			info.Reset()
			return true
		}
	}

}

type LockInfo struct {
	sync.RWMutex
	IPAddr string
	IsLock bool
	Time   time.Time

	Count int
	IsOK  bool
}

func (info *LockInfo) Reset() {
	info.Count = 0
	info.IsLock = false
	info.IsOK = false
	info.Time = time.Now()
}

func CreateLockInfo(IP string) *LockInfo {
	info := &LockInfo{
		IPAddr: IP,
	}
	info.Reset()
	return info
}

func (info *LockInfo) Set(isOK bool) bool {
	info.Lock()
	defer info.Unlock()
	if isOK {
		info.Reset()
		info.IsOK = true
		return true
	} else {
		if info.CheckTime() {
			info.Count++
			return info.Count <= 3
		} else {
			info.Reset()
			info.Count++
			return true
		}

	}
}

func (info *LockInfo) CheckTime() bool {
	return time.Now().Sub(info.Time).Seconds() < 180
}

type PointInfo struct {
	Point        int `json:"UID"`
	sync.RWMutex `json:"-"`
	Time         time.Time `json:"-"`
	State        int       `json:"-"` //0初始 1查詢中 2己確認 3己失效
}

func (p *PointInfo) SetState(state int) {
	p.RWMutex.Lock()
	defer p.RWMutex.Unlock()
	p.State = state
}

func (p *PointInfo) GetState() int {
	p.RWMutex.RLock()
	defer p.RWMutex.RUnlock()
	return p.State
}

func (p *PointInfo) SetPoint(point int) {
	p.RWMutex.Lock()
	defer p.RWMutex.Unlock()
	p.Point = point
	p.Time = time.Now()
}

func (p *PointInfo) GetPoint() (int, error) {
	p.RWMutex.RLock()
	defer p.RWMutex.RUnlock()
	if time.Now().Before(p.Time.Add(60 * time.Second)) {
		return p.Point, nil
	} else {
		return p.Point, errors.New("Time Out")
	}
}

type VIPInfo struct {
	EndTime      time.Time `json:"-"`
	sync.RWMutex `json:"-"`
	Time         time.Time `json:"-"`
	State        int       `json:"-"` //0初始 1查詢中 2己確認 3己失效
	IsVIP        string    `json:"-"`
}

func (v *VIPInfo) SetState(state int) {
	v.RWMutex.Lock()
	defer v.RWMutex.Unlock()
	v.State = state
}

func (v *VIPInfo) GetState() int {
	v.RWMutex.RLock()
	defer v.RWMutex.RUnlock()
	return v.State
}

func (v *VIPInfo) SetEndTime(etime time.Time) {
	v.RWMutex.Lock()
	defer v.RWMutex.Unlock()
	v.Time = time.Now()
	v.EndTime = etime
}

func (v *VIPInfo) GetEndTime() (time.Time, error) {
	v.RWMutex.RLock()
	defer v.RWMutex.RUnlock()

	if time.Now().Before(v.Time.Add(60 * time.Second)) {
		return v.EndTime, nil
	} else {
		return v.EndTime, errors.New("Time Out")
	}
}

// 因db回傳的數字型能都是int64
type ConsumeResultInfo struct {
	Result      int64
	LivePoint   int64
	LiveEndTime time.Time
	IsLiveVIP   int64
}

//type APIResultInfo struct {
//	Result  int
//	Message interface{}
//	Code    string
//}

//type APIRequestInfo struct {
//	Action  string
//	Message interface{}
//}

type WSAuthInfo struct {
	Data WSAuthDataInfo `json:"data"`
}

type WSAuthDataInfo struct {
	WSURL string `json:"websocketUrl"`
}

type TimeInfo struct {
	CheckTime time.Time
	EndTime   time.Time
}
