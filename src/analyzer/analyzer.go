package analyzer

import (
	"base"
	"net/http"
)

// 被用于解析http响应的函数类型
type ParseResponse func(httpResp *http.Response, respDepth uint32) ([]base.Data, []error)

//分析器的接口类型
type Analyzer interface {
	Id() uint32
	Analyzer(respParses []ParseResponse, resp base.Response) ([]base.Data, []error)
}
