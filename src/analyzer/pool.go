package analyzer

type AnalyzerPool interface {
	Take() (PageDownloader, error)  //从池中取出一个分析器
	Return(dl PageDownloader) error //归还一个下载器
	Total() uint32                  //分析器总数
	Used() uint32                   //正在使用的数量
}
