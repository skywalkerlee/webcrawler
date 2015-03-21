package analyzer

import (
	"errors"
	"fmt"
	"middleware"
	"reflect"
)

type AnalyzerPool interface {
	Take() (Analyzer, error)        //从池中取出一个分析器
	Return(analyzer Analyzer) error //归还一个下载器
	Total() uint32                  //分析器总数
	Used() uint32                   //正在使用的数量
}

type GenPageAnalyzerPool func() Analyzer

type myAnalyzerPool struct {
	pool  middleware.Pool
	etype reflect.Type
}

func NewAnalyzerPool(total uint32, gen GenPageAnalyzerPool) (AnalyzerPool, error) {
	entityType := reflect.TypeOf(gen())
	genEntity := func() middleware.Entity {
		return gen()
	}
	pool, err := middleware.NewPool(total, entityType, genEntity)
	if err != nil {
		return nil, err
	}
	analyzerPool := &myAnalyzerPool{pool: pool, etype: entityType}
	return analyzerPool, nil
}

func (aPool *myAnalyzerPool) Take() (Analyzer, error) {
	entity, err := aPool.pool.Take()
	if err != nil {
		return nil, err
	}
	a, ok := entity.(Analyzer)
	if !ok {
		errMsg := fmt.Sprintf("The type of entity is NOT %s!\n", aPool.etype)
		panic(errors.New(errMsg))
	}
	return a, nil
}

func (aPool *myAnalyzerPool) Return(a Analyzer) error {
	return aPool.pool.Return(a)
}

func (aPool *myAnalyzerPool) Total() uint32 {
	return aPool.pool.Total()
}

func (aPool *myAnalyzerPool) Used() uint32 {
	return aPool.pool.Used()
}
