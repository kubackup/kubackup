package model

import (
	"fmt"
	"github.com/kataras/iris/v12/context"
)

const (
	PageSize = "pageSize"
	PageNum  = "pageNum"
)

//默认值
const (
	PageSizeD = 10
	PageNumD  = 1
)

type Page struct {
	Total    int         `json:"total"`    // 数据总量
	PageSize int         `json:"pageSize"` // 每页大小
	PageNum  int         `json:"pageNum"`  // 页码
	Items    interface{} `json:"items"`    // 数据列表
}

func PageParam(ctx *context.Context) *Page {
	pageNum, err1 := ctx.URLParamInt(PageNum)
	pageSize, err2 := ctx.URLParamInt(PageSize)
	if err1 != nil {
		pageNum = PageNumD
	}
	if err2 != nil {
		pageSize = PageSizeD
	}
	res := &Page{
		PageNum:  pageNum,
		PageSize: pageSize,
	}
	return res
}

// PageFilter 分页
func PageFilter(num, size int, data []interface{}) (int, []interface{}, error) {
	total := len(data)
	result := make([]interface{}, 0)
	if num < 1 {
		return 0, result, fmt.Errorf("num不能小于1")
	}
	if num*size < total {
		result = data[(num-1)*size : (num * size)]
	} else {
		if (num-1)*size < total {
			result = data[(num-1)*size:]
		}
	}
	return total, result, nil
}
