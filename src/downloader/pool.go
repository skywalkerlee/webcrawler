package downloader

type PageDownloaderPool interface {
	Take() (PageDownloader, error)  //从票池中取出一个下载器
	Return(dl PageDownloader) error //归还一个下载器
	Total() uint32                  //下载器总数
	Used() uint32                   //正在使用的数量
}
