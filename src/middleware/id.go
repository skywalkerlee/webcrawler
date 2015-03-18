package middleware

type IdGenertor interface {
	GetUint32() uint32 //获得一个uint32类型的id
}
