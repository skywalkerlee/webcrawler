package middleware

import (
	"base"
	"errors"
	"fmt"
	"sync"
)

// 被用来表示通道管理器的状态的类型。
type ChannelManagerStatus uint8

const (
	CHANNEL_MANAGER_STATUS_UNINITIALIZED ChannelManagerStatus = 0 // 未初始化状态。
	CHANNEL_MANAGER_STATUS_INITIALIZED   ChannelManagerStatus = 1 // 已初始化状态。
	CHANNEL_MANAGER_STATUS_CLOSED        ChannelManagerStatus = 2 // 已关闭状态。
)

// 表示状态代码与状态名称之间的映射关系的字典。
var statusNameMap = map[ChannelManagerStatus]string{
	CHANNEL_MANAGER_STATUS_UNINITIALIZED: "uninitialized",
	CHANNEL_MANAGER_STATUS_INITIALIZED:   "initialized",
	CHANNEL_MANAGER_STATUS_CLOSED:        "closed",
}

//摘要信息模板

type ChannelManager interface {
	//初始化管道管理器
	//参数channelLen代表管道管理器中的各类管道初始长度
	//参数reset指明是否重新初始化管道管理器
	Init(channelLen uint, reset bool) bool
	//关闭管道管理器
	Close() bool
	//获取请求传输通道
	ReqChan() (chan base.Request, error)
	//获取响应传输通道
	RespChan() (chan base.Response, error)
	//获取条目传输通道
	ItemChan() (chan base.Item, error)
	//获取错误传输通道
	ErrorChan() (chan error, error)
	//获取通道创渎职
	ChannelLen() uint
	//获取通道管理器状态
	Status() ChannelManagerStatus
	//获取摘要信息
	Summary() string
}

type myChannelManager struct {
	channelLen uint                 //通道长度
	reqCh      chan base.Request    //请求通道
	respCh     chan base.Response   //响应通道
	itemCh     chan base.Item       //条目通道
	errorCh    chan error           //错误通道
	status     ChannelManagerStatus //通道管理器状态
	rwmutex    sync.RWMutex         //读写锁
}

func NewChannelManager(channelLen uint) ChannelManager {
	if channelLen == 0 {
		channelLen = 5
	}
	chanman := &myChannelManager{}
	chanman.Init(channelLen, true)
	return chanman
}

func (chanman *myChannelManager) Init(channelLen uint, reset bool) bool {
	if channelLen == 0 {
		panic(errors.New("The channel length is invalid"))
	}
	chanman.rwmutex.Lock()
	defer chanman.rwmutex.Unlock()
	if chanman.status == CHANNEL_MANAGER_STATUS_INITIALIZED && !reset {
		return false
	}
	chanman.channelLen = channelLen
	chanman.reqCh = make(chan base.Request, channelLen)
	chanman.respCh = make(chan base.Response, channelLen)
	chanman.itemCh = make(chan base.Item, channelLen)
	chanman.errorCh = make(chan error, channelLen)
	chanman.status = CHANNEL_MANAGER_STATUS_INITIALIZED
	return true
}

func (chanman *myChannelManager) Close() bool {
	chanman.rwmutex.Lock()
	defer chanman.rwmutex.Unlock()
	if chanman.status != CHANNEL_MANAGER_STATUS_INITIALIZED {
		return false
	}
	close(chanman.reqCh)
	close(chanman.respCh)
	close(chanman.itemCh)
	close(chanman.errorCh)
	chanman.status = CHANNEL_MANAGER_STATUS_CLOSED
	return true
}

func (chanman *myChannelManager) checkStatus() error {
	if chanman.status == CHANNEL_MANAGER_STATUS_INITIALIZED {
		return nil
	}
	statusName, ok := statusNameMap[chanman.status]
	if !ok {
		statusName = fmt.Sprintf("%d", chanman.status)
	}
	errMsg := fmt.Sprintf("The undesirable status of channel manager: %S!\n", statusName)
	return errors.New(errMsg)
}

func (chanman *myChannelManager) ReqChan() (chan base.Request, error) {
	chanman.rwmutex.RLock()
	defer chanman.rwmutex.RUnlock()
	if err := chanman.checkStatus(); err != nil {
		return nil, err
	}
}
func (chanman *myChannelManager) RespChan() (chan base.Response, error) {
	chanman.rwmutex.RLock()
	defer chanman.rwmutex.RUnlock()
	if err := chanman.checkStatus(); err != nil {
		return nil, err
	}
	return chanman.respCh, nil
}

func (chanman *myChannelManager) ItemChan() (chan base.Item, error) {
	chanman.rwmutex.RLock()
	defer chanman.rwmutex.RUnlock()
	if err := chanman.checkStatus(); err != nil {
		return nil, err
	}
	return chanman.itemCh, nil
}

func (chanman *myChannelManager) ErrorChan() (chan error, error) {
	chanman.rwmutex.RLock()
	defer chanman.rwmutex.RUnlock()
	if err := chanman.checkStatus(); err != nil {
		return nil, err
	}
	return chanman.errorCh, nil
}
