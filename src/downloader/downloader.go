package downloader

import (
	"base"
)

type PageDownloader interface {
	Id() uint32 //获取id
	Download(req base.Request) (*base.Response, error)
}
