package middleware

import ()

type Pool interface {
	Take() (Entity, error)      //取出实体
	Return(entity Entity) error //归还实体
	Total() uint32              //实体容量
	used() uint32               //已被使用数量
}

type Entity interface {
	Id() uint32
}
