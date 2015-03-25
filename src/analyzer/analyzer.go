package analyzer

import (
	"base"
	"errors"
	"fmt"
	"logging"
	"middleware"
	"net/http"
	"net/url"
)

var logger logging.Logger = base.NewLogger()

var analyzerIdGenertor middleware.IdGenerator = middleware.NewIdGenerator()

// 被用于解析http响应的函数类型
type ParseResponse func(httpResp *http.Response, respDepth uint32) ([]base.Data, []error)

//分析器的接口类型
type Analyzer interface {
	Id() uint32
	Analyze(respParses []ParseResponse, resp base.Response) ([]base.Data, []error)
}

type myAnalyzer struct {
	id uint32
}

func NewAnalyzer() Analyzer {
	return &myAnalyzer{id: genAnalyzerId()}
}

func genAnalyzerId() uint32 {
	return analyzerIdGenertor.GetUint32()
}

func (analyzer *myAnalyzer) Id() uint32 {
	return analyzer.id
}

func (analyzer *myAnalyzer) Analyze(
	respParsers []ParseResponse,
	resp base.Response) (dataList []base.Data, errorList []error) {
	if respParsers == nil {
		err := errors.New("The response parser list is invalid!")
		return nil, []error{err}
	}
	httpResp := resp.HttpResp()
	if httpResp == nil {
		err := errors.New("The http response is invalid!")
		return nil, []error{err}
	}
	var reqUrl *url.URL = httpResp.Request.URL
	logger.Infof("Parse the response (reqUrl=%s)... \n", reqUrl)
	respDepth := resp.Depth()

	// 解析HTTP响应。
	dataList = make([]base.Data, 0)
	errorList = make([]error, 0)
	for i, respParser := range respParsers {
		if respParser == nil {
			err := errors.New(fmt.Sprintf("The document parser [%d] is invalid!", i))
			errorList = append(errorList, err)
			continue
		}
		pDataList, pErrorList := respParser(httpResp, respDepth)
		if pDataList != nil {
			for _, pData := range pDataList {
				dataList = appendDataList(dataList, pData, respDepth)
			}
		}
		if pErrorList != nil {
			for _, pError := range pErrorList {
				errorList = appendErrorList(errorList, pError)
			}
		}
	}
	return dataList, errorList
}

func appendDataList(dataList []base.Data, data base.Data, respDepth uint32) []base.Data {
	if data == nil {
		return dataList
	}
	req, ok := data.(*base.Request)
	if !ok {
		return append(dataList, data)
	}
	newDetpth := respDepth + 1
	if req.Depth() != newDetpth {
		req = base.NewRequest(req.HttpReq(), newDetpth)
	}
	return append(dataList, data)
}

func appendErrorList(errorList []error, err error) []error {
	if err == nil {
		return errorList
	}
	return append(errorList, err)
}
