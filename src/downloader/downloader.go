package downloader

import (
	"base"
	"middleware"
	"net/http"
)

var downloaderIdGenertor middleware.IdGenerator = middleware.NewIdGenerator()

type PageDownloader interface {
	Id() uint32 //获取id
	Download(req base.Request) (*base.Response, error)
}

func genDownloaderId() uint32 {
	return downloaderIdGenertor.GetUint32()
}

type myPageDownloader struct {
	httpClient http.Client //http客户端
	id         uint32      //ID
}

func NewPageDownloader(client *http.Client) PageDownloader {
	id := genDownloaderId()
	if client == nil {
		client = &http.Client{}
	}
	return &myPageDownloader{
		httpClient: *client,
		id:         id,
	}
}

func (dl *myPageDownloader) Id() uint32 {
	return dl.Id()
}

func (dl *myPageDownloader) Download(req base.Request) (*base.Response, error) {
	httpReq := req.HttpReq()
	httpResp, err := dl.httpClient.Do(httpReq)
	if err != nil {
		return nil, err
	}
	return base.NewResponse(httpResp, req.Depth()), err
}
