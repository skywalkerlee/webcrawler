package downloader

import (
	"errors"
	"fmt"
	"middleware"
	"reflect"
)

type PageDownloaderPool interface {
	Take() (PageDownloader, error)  //从票池中取出一个下载器
	Return(dl PageDownloader) error //归还一个下载器
	Total() uint32                  //下载器总数
	Used() uint32                   //正在使用的数量
}

type GenPageDownloader func() PageDownloader

type myDownloaderPool struct {
	pool  middleware.Pool //实体池
	etype reflect.Type    //池内实体的类型
}

func NewDownloaderPool(total uint32, gen GenPageDownloader) (PageDownloaderPool, error) {
	entityType := reflect.TypeOf(gen())
	genEntity := func() middleware.Entity {
		return gen()
	}
	pool, err := middleware.NewPool(total, entityType, genEntity)
	if err != nil {
		return nil, err
	}
	dlPool := &myDownloaderPool{pool: pool, etype: entityType}
	return dlPool, nil
}

func (dlPool *myDownloaderPool) Take() (PageDownloader, error) {
	entity, err := dlPool.pool.Take()
	if err != nil {
		return nil, err
	}
	dl, ok := entity.(PageDownloader)
	if !ok {
		errMsg := fmt.Sprintf("The type of entity is NOT %s!\n", dlPool.etype)
		panic(errors.New(errMsg))
	}
	return dl, nil
}

func (dlPool *myDownloaderPool) Return(dl PageDownloader) error {
	return dlPool.pool.Return(dl)
}

func (dlPool *myDownloaderPool) Total() uint32 {
	return dlPool.pool.Total()
}

func (dlPool *myDownloaderPool) Used() uint32 {
	return dlPool.pool.Used()
}
