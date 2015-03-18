package base

import (
	"net/http"
)

//请求
type Request struct {
	httpReq *http.Request //http请求
	depth   uint32        //请求的深度
}

//初始化Request结构
func NewRequest(httpReq *http.Request, depth uint32) *Request {
	return &Request{httpReq: httpReq, depth: depth}
}

//获取http请求
func (req *Request) HttpReq() *http.Request {
	return req.httpReq
}

//获取请求深度
func (req *Request) Depth() uint32 {
	return req.depth
}

//响应
type Response struct {
	httpResp *http.Response
	depth    uint32
}

//初始化响应
func NewResponse(httpResp *http.Response, depth uint32) *Response {
	return &Response{httpResp: httpResp, depth: depth}
}

//获取http响应
func (resp *Response) HttpResp() *http.Response {
	return resp.httpResp
}

//获取响应深度
func (resp *Response) Depth() uint32 {
	return resp.depth
}

//条目
type Item map[string]interface{}

//数据接口
type Data interface {
	Valid() bool //数据是否有效
}

func (req *Request) Valid() bool {
	return req.httpReq != nil && req.httpReq.URL != nil
}

func (resp *Response) Valid() bool {
	return resp.httpResp != nil && resp.httpResp.Body != nil
}

func (item *Item) Valid() bool {
	return item != nil
}
