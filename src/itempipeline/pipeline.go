package itempipeline

import (
	"base"
)

type Itempipeline interface {
	//发送条目
	send(item base.Item) []error
	//条目是否快速失败 ，快速失败指处理流程出错，如果出错则忽略后续操作
	FailFast() bool
	//设置是否快速失败
	SetFailFsat(failRast bool)
	//获得已发送、已接受、已处理的条目计数值
	Count() []uint64
	//获取正在被处理的条目总数
	ProcessingNumber() uint64
	//获取摘要信息
	Summary() string
}
