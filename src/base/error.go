package base

import (
	"bytes"
	"fmt"
)

type ErrorType string

type CrawlerError interface {
	Type() ErrorType //获得错误类型
	Error() string   //获得错误提示信息
}

const (
	DOWNLOADER_ERROR     ErrorType = "Downloader Error"
	ANALYZER_ERROR       ErrorType = "Analyzer Error"
	ITEM_PROCESSOR_ERROR ErrorType = "Item Processsor Error"
)

type myCrawlerError struct {
	errType    ErrorType //错误类型
	errMsg     string    //错误提示信息
	fullErrMsg string    //完整的错误提示信息
}

//初始化
func NewCrawlerError(errType ErrorType, errMsg string) CrawlerError {
	return &myCrawlerError{errType: errType, errMsg: errMsg}
}

//获得错误类型
func (ce *myCrawlerError) Type() ErrorType {
	return ce.errType
}

//获得错误提示信息
func (ce *myCrawlerError) Error() string {
	if ce.fullErrMsg == "" {
		ce.genFullErrMsg()
	}
	return ce.fullErrMsg
}

//生成错误提示信息
func (ce *myCrawlerError) genFullErrMsg() {
	var buffer bytes.Buffer
	buffer.WriteString("爬虫错误(Crawler Error)：")
	if ce.errType != "" {
		buffer.WriteString(string(ce.errType))
		buffer.WriteString(": ")
	}
	buffer.WriteString(ce.errMsg)
	ce.fullErrMsg = fmt.Sprintf("%s\n", buffer.String())
	return
}
