package scheduler

import (
	"analyzer"
	"base"
	"downloader"
	"errors"
	"fmt"
	"itempipeline"
	"logging"
	"middleware"
	"net/http"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

var logger logging.Logger = base.NewLogger()

const (
	DOWNLOADER_CODE   = "downloader"
	ANALYZER_CODE     = "analyzer"
	ITEMPIPELINE_CODE = "item_pipeline"
	SCHEDULER_CODE    = "scheduler"
)

type Scheduler interface {
	// 开启调度器。
	// 调用该方法会使调度器创建和初始化各个组件。在此之后，调度器会激活爬取流程的执行。
	// 参数channelLen用来指定数据传输通道的长度
	// 参数poolSize用来设定网页下载器池和分析器池的容量
	// 参数crawlDepth代表了需要被爬取的网页的最大深度值。深度大于此值的网页会被忽略。
	// 参数httpClientGenerator代表的是被用来生成HTTP客户端的函数。
	// 参数respParsers的值应为分析器所需的被用来解析HTTP响应的函数的序列。
	// 参数itemProcessors的值应为需要被置入条目处理管道中的条目处理器的序列。
	// 参数firstHttpReq即代表首次请求。调度器会以此为起始点开始执行爬取流程。
	Start(channelArgs base.ChannelArgs,
		poolBaseArgs base.PoolBaseArgs,
		crawDepth uint32,
		httpClientGenerator GenHttpClient,
		respParsers []analyzer.ParseResponse,
		itemProcessors []itempipeline.ProcessItem,
		firstHttpReq *http.Request,
	) (err error)
	// 调用该方法会停止调度器的运行。所有处理模块执行的流程都会被中止。
	Stop() bool
	// 判断调度器是否正在运行。
	Running() bool
	// 获得错误通道。调度器以及各个处理模块运行过程中出现的所有错误都会被发送到该通道。
	// 若该方法的结果值为nil，则说明错误通道不可用或调度器已被停止。
	ErrorChan() <-chan error
	// 判断所有处理模块是否都处于空闲状态。
	Idle() bool
	// 获取摘要信息。
	Summary(prefix string) SchedSummary
}

type GenHttpClient func() *http.Client

type myScheduler struct {
	channelArgs   base.ChannelArgs              //池的尺寸
	poolBaseArgs  base.PoolBaseArgs             //通道容量
	crawlDepth    uint32                        //爬取的最大深度，首次请求的深度为0
	primaryDomain string                        //主域名
	chanman       middleware.ChannelManager     //通道管理器
	stopSign      middleware.StopSign           //停止信号
	dlpool        downloader.PageDownloaderPool //网页下载器池
	analyzerPool  analyzer.AnalyzerPool         //分析器池
	itemPipeline  itempipeline.Itempipeline     //条目处理管道
	running       uint32                        //0表示未运行，1表示已运行，2表示已停止
	reqCache      requestCache                  //请求缓存
	urlMap        map[string]bool               //已请求的url字典
	wg            sync.WaitGroup
}

func NewScheduler() Scheduler {
	return &myScheduler{}
}

func (sched *myScheduler) Start(
	channelArgs base.ChannelArgs,
	poolBaseArgs base.PoolBaseArgs,
	crawDepth uint32,
	httpClientGenerator GenHttpClient,
	respParsers []analyzer.ParseResponse,
	itemProcessors []itempipeline.ProcessItem,
	firstHttpReq *http.Request) (err error) {
	defer func() {
		if p := recover(); p != nil {
			errMsg := fmt.Sprintf("Fatal Scheduler Error: %s\n", p)
			logger.Fatal(errMsg)
			err = errors.New(errMsg)
		}
	}()
	if atomic.LoadUint32(&sched.running) == 1 {
		return errors.New("The scheduler has been started\n")
	}
	atomic.StoreUint32(&sched.running, 1)
	if err := channelArgs.Check(); err != nil {
		return err
	}
	sched.channelArgs = channelArgs
	if err := poolBaseArgs.Check(); err != nil {
		return err
	}
	sched.poolBaseArgs = poolBaseArgs
	sched.crawlDepth = crawDepth
	sched.chanman = generateChannelManager(sched.channelArgs)
	if httpClientGenerator == nil {
		return errors.New("The http client generator list is invalid")
	}
	dlPool, err := generatePageDownloaderPool(sched.poolBaseArgs.PageDownloaderPoolSize(), httpClientGenerator)
	if err != nil {
		errMsg := fmt.Sprintf("Occur error shen get pagedownloader pool: %s\n", err)
		return errors.New(errMsg)
	}
	sched.dlpool = dlPool
	analyzerPool, err := generateAnalyzerPool(sched.poolBaseArgs.AnalyzerPoolSize())
	if err != nil {
		errMsg := fmt.Sprintf("Occur error shen get analyzer pool: %s\n", err)
		return errors.New(errMsg)
	}
	sched.analyzerPool = analyzerPool
	if itemProcessors == nil {
		return errors.New("THe item processor list is invalid")
	}
	for i, ip := range itemProcessors {
		if ip == nil {
			return errors.New(fmt.Sprintf("The %dth item processor is invalid", i))
		}
	}
	sched.itemPipeline = generateItemPipeline(itemProcessors)
	if sched.stopSign == nil {
		sched.stopSign = middleware.NewStopSign()
	} else {
		sched.stopSign.Reset()
	}
	sched.urlMap = make(map[string]bool)
	sched.reqCache = newRequestCache()
	sched.wg.Add(4)
	sched.startDownloading()
	sched.activateAnalyzers(respParsers)
	sched.openItemPipeline()
	sched.schedule(10 * time.Millisecond)
	//
	if firstHttpReq == nil {
		return errors.New("The first http request is invalid")
	}
	pd, err := getPrimaryDomain(firstHttpReq.Host)
	if err != nil {
		return err
	}
	sched.primaryDomain = pd
	firstReq := base.NewRequest(firstHttpReq, 0)
	sched.reqCache.put(firstReq)
	sched.wg.Wait()
	return nil
}

//激活下载器
func (sched *myScheduler) startDownloading() {
	go func() {
		defer sched.wg.Done()
		for {
			req, ok := <-sched.getReqchan()
			if !ok {
				break
			}
			go sched.download(req)
		}
	}()
}

func (sched *myScheduler) getReqchan() chan base.Request {
	reqchan, err := sched.chanman.ReqChan()
	if err != nil {
		panic(err)
	}
	return reqchan
}

func (sched *myScheduler) getRespchan() chan base.Response {
	respchan, err := sched.chanman.RespChan()
	if err != nil {
		panic(err)
	}
	return respchan
}

func (sched *myScheduler) getErrorChan() chan error {
	errchan, err := sched.chanman.ErrorChan()
	if err != nil {
		panic(err)
	}
	return errchan
}

func (sched *myScheduler) getItemChan() chan base.Item {
	itemchan, err := sched.chanman.ItemChan()
	if err != nil {
		panic(err)
	}
	return itemchan
}

func (sched *myScheduler) download(req base.Request) {
	defer func() {
		if p := recover(); p != nil {
			errMsg := fmt.Sprintf("Fatal Download Error: %s\n", p)
			logger.Fatal(errMsg)
		}
	}()
	downloader, err := sched.dlpool.Take()
	if err != nil {
		errMsg := fmt.Sprintf("Downloader pool error: %s", err)
		sched.sendError(errors.New(errMsg), SCHEDULER_CODE)
		return
	}
	defer func() {
		err := sched.dlpool.Return(downloader)
		if err != nil {
			errMsg := fmt.Sprintf("Downloader pool error: %s", err)
			sched.sendError(errors.New(errMsg), SCHEDULER_CODE)
		}
	}()
	code := generateCode(DOWNLOADER_CODE, downloader.Id())
	respp, err := downloader.Download(req)
	if respp != nil {
		sched.sendResp(*respp, code)
	}
	if err != nil {
		sched.sendError(err, code)
	}
}

func (sched *myScheduler) sendResp(resp base.Response, code string) bool {
	if sched.stopSign.Signed() {
		sched.stopSign.Deal(code)
		return false
	}
	sched.getRespchan() <- resp
	return true
}

func (sched *myScheduler) sendError(err error, code string) bool {
	if err == nil {
		return false
	}
	codePrefix := parseCode(code)[0]
	var errorType base.ErrorType
	switch codePrefix {
	case DOWNLOADER_CODE:
		errorType = base.DOWNLOADER_ERROR
	case ANALYZER_CODE:
		errorType = base.ANALYZER_ERROR
	case ITEMPIPELINE_CODE:
		errorType = base.ITEM_PROCESSOR_ERROR
	}
	cError := base.NewCrawlerError(errorType, err.Error())
	if sched.stopSign.Signed() {
		sched.stopSign.Deal(code)
		return false
	}
	go func() {
		sched.getErrorChan() <- cError
	}()
	return true
}

//激活分析器
func (sched *myScheduler) activateAnalyzers(respParsers []analyzer.ParseResponse) {
	go func() {
		defer sched.wg.Done()
		for {
			resp, ok := <-sched.getRespchan()
			if !ok {
				break
			}
			go sched.analyze(respParsers, resp)
		}
	}()
}

func (sched *myScheduler) analyze(respParsers []analyzer.ParseResponse, resp base.Response) {
	defer func() {
		if p := recover(); p != nil {
			errMsg := fmt.Sprintf("Fatal Analysis Error:%s\n", p)
			logger.Fatal(errMsg)
		}
	}()

	analyzer, err := sched.analyzerPool.Take()
	if err != nil {
		errMsg := fmt.Sprintf("Analyzer pool error:%s", err)
		sched.sendError(errors.New(errMsg), SCHEDULER_CODE)
		return
	}
	defer func() {
		err := sched.analyzerPool.Return(analyzer)
		if err != nil {
			errMsg := fmt.Sprintf("Analyzer pool error: %s", err)
			sched.sendError(errors.New(errMsg), SCHEDULER_CODE)
		}
	}()
	code := generateCode(ANALYZER_CODE, analyzer.Id())
	dataList, errs := analyzer.Analyze(respParsers, resp)
	if errs != nil {
		for _, err := range errs {
			sched.sendError(err, code)
		}
	}
	if dataList != nil {
		for _, data := range dataList {
			if data == nil {
				continue
			}
			switch d := data.(type) {
			case *base.Item:
				sched.sendItem(*d, code)
			case *base.Request:
				sched.saveReqToCache(*d, code)
			default:
				errMsg := fmt.Sprintf("Unsupported data type '%T'! value=(%v)\n", d, d)
				sched.sendError(errors.New(errMsg), code)
			}
		}
	}
	if err != nil {
		for _, err := range errs {
			sched.sendError(err, code)
		}
	}
}

func (sched *myScheduler) sendItem(item base.Item, code string) bool {
	if sched.stopSign.Signed() {
		sched.stopSign.Deal(code)
		return false
	}
	sched.getItemChan() <- item
	return true
}

func (sched *myScheduler) saveReqToCache(req base.Request, code string) bool {
	httpReq := req.HttpReq()
	if httpReq == nil {
		logger.Warnln("Ignore the request! It's http request is invalid")
		return false
	}
	reqUrl := httpReq.URL
	if reqUrl == nil {
		logger.Warnln("Ignore the request! It's url is invalid")
		return false
	}
	if strings.ToLower(reqUrl.Scheme) != "http" {
		logger.Warnf("Ignore the request! It's url scheme '%s',but not a http", reqUrl.Scheme)
		return false
	}

	if _, ok := sched.urlMap[reqUrl.String()]; ok {
		logger.Warnf("Ignore the request! it's url is repeated.(requestUrl=%s)\n", reqUrl)
		return false
	}
	if pd, _ := getPrimaryDomain(httpReq.Host); pd != sched.primaryDomain {
		logger.Warnf("Ignore the request!it's host '%s' not in primary domain '%s' requestUrl=%s\n", httpReq.Host, sched.primaryDomain, reqUrl)
		return false
	}
	if req.Depth() > sched.crawlDepth {
		logger.Warnf("Ignore the request! it's depth %d greater than %d\n request=%s", req.Depth(), sched.crawlDepth, reqUrl)
		return false
	}
	if sched.stopSign.Signed() {
		sched.stopSign.Deal(code)
		return false
	}
	sched.reqCache.put(&req)
	sched.urlMap[reqUrl.String()] = true
	return true
}

func (sched *myScheduler) openItemPipeline() {
	go func() {
		defer sched.wg.Done()
		sched.itemPipeline.SetFailFsat(true)
		code := ITEMPIPELINE_CODE
		for item := range sched.getItemChan() {
			go func(item base.Item) {
				defer func() {
					if p := recover(); p != nil {
						errMsg := fmt.Sprintf("Fatal Item processing error:%s\n", p)
						logger.Fatal(errMsg)
					}
				}()
				errs := sched.itemPipeline.Send(item)
				if errs != nil {
					for _, err := range errs {
						sched.sendError(err, code)
					}
				}
			}(item)
		}
	}()
}

func (sched *myScheduler) schedule(interval time.Duration) {
	go func() {
		defer sched.wg.Done()
		for {
			if sched.stopSign.Signed() {
				sched.stopSign.Deal(SCHEDULER_CODE)
				return
			}
			remainder := cap(sched.getReqchan()) - len(sched.getReqchan())
			var temp *base.Request
			for remainder > 0 {
				temp = sched.reqCache.get()
				if temp == nil {
					break
				}
				sched.getReqchan() <- *temp
				remainder--
			}
			time.Sleep(interval)
		}
	}()
}

func (sched *myScheduler) Stop() bool {
	if atomic.LoadUint32(&sched.running) != 1 {
		return false
	}
	sched.stopSign.Sign()
	sched.chanman.Close()
	sched.reqCache.close()
	atomic.StoreUint32(&sched.running, 2)
	return true
}

func (sched *myScheduler) Running() bool {
	return atomic.LoadUint32(&sched.running) == 1
}

func (sched *myScheduler) ErrorChan() <-chan error {
	if sched.chanman.Status() != middleware.CHANNEL_MANAGER_STATUS_INITIALIZED {
		return nil
	}
	return sched.getErrorChan()
}

func (sched *myScheduler) Idle() bool {
	idleDlPool := sched.dlpool.Used() == 0
	idleAnalyzerPool := sched.analyzerPool.Used() == 0
	idleItemPipeline := sched.itemPipeline.ProcessingNumber() == 0
	if idleDlPool && idleAnalyzerPool && idleItemPipeline {
		return true
	}
	return false
}

func (sched *myScheduler) Summary(prefix string) SchedSummary {
	return NewSchedSummary(sched, prefix)
}
