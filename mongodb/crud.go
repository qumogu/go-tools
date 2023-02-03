package mongodb

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"reflect"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/qumogu/go-tools/httpserver"
	"github.com/xuri/excelize/v2"
	"go.mongodb.org/mongo-driver/bson"
)

// type CRUD interface {
// 	Create(ctx context.Context, data interface{}) (string, error)
// 	DeleteByIds(ctx context.Context, ids []string) (int64, error)
// 	UpdateByID(ctx context.Context, id string, data interface{}) error
// 	FindWithPage(ctx context.Context, filter interface{}, pageOrder PageOrderIntfc, results interface{}) (int64, error)
// }

// All of the methods are the same type as HandlerFunc
// if you don't want to support any methods of CRUD, then don't implement it
type CreateSupported interface {
	CreateController(c *gin.Context)
}

type ListSupported interface {
	ListController(c *gin.Context)
}

type InfoSupported interface {
	InfoController(c *gin.Context)
}

type UpdateSupported interface {
	UpdateController(c *gin.Context)
}

type DeleteSupported interface {
	DeleteController(c *gin.Context)
}

type ExportExcelSupported interface {
	ExportExcelController(c *gin.Context)
}

type UploadExcelSupported interface {
	UploadExcelController(c *gin.Context)
}

// Tile对应的json key
type ExcelColumn struct {
	Key          string
	Name         string
	ExportFormat func(interface{}) interface{} // TODO: 增加参数 elemValue, 见 ExportExcelController
	ImportFormat func(interface{}) interface{}
}

type Crud struct {
	// newModel func()interface{}
	// newCreate func()interface{}
	// search interface{}
	// newSearch func()interface{}
	// newUpdate func()interface{}
	// listResult func()interface{}
	param Param

	tileMap       map[string]int // 保存key的顺序序号来计算列索引
	exportColumns []ExcelColumn  // excel导出保存列的属性, 格式化函数等

	collection string
	database   string
}

func (d *Crud) getMongoBase(c *gin.Context) *MongoBase {

	database := d.database

	mongoDB, err := DefaultMongoMgr.GetDB(database)
	if err != nil {
		return nil
	}

	collection := mongoDB.Collection(d.collection)
	return NewCrudBase(collection)
}

type ResulListtVO struct {
	// Data []model.Route `json:"data"`
	Data       interface{} `json:"data"` // 切片
	TotalCount int64       `json:"total_count"`
}

type DeleteParam struct {
	IDs []string `json:"ids" form:"ids" bson:"ids"`
}

type Param interface {
	NewModel(c *gin.Context) interface{}
	NewCreate(c *gin.Context) interface{}
	NewSearch(c *gin.Context) interface{}
	NewUpdate(c *gin.Context) interface{}
	NewListResult(c *gin.Context) interface{}
	ExcelColumns() []ExcelColumn
	ExcelName() string
}

// CRUD 库
// database 为空说明分多租户, 租户ID为数据库名称
func NewCrud(database, collection string, param Param) *Crud {
	// tileMap:= map[string]int{
	// 	"name": 1,
	// 	"age":2,
	// 	"location": 3,
	// 	"update_time": 4,
	// }
	columns := param.ExcelColumns()
	if len(columns) == 0 {
		panic("export columns should not be empty")
	}

	tileMap := make(map[string]int, len(columns))

	for idx, col := range columns {
		tileMap[col.Key] = idx + 1
	}

	return &Crud{
		collection: collection,
		// newModel :newModel,
		// newCreate:newCreate,
		// newSearch: newSearch,
		// newUpdate: newUpdate,
		// listResult: list,
		database:      database,
		param:         param,
		tileMap:       tileMap,
		exportColumns: columns,
	}
}

func (d *Crud) CreateController(c *gin.Context) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()
	data := d.param.NewCreate(c)
	err := c.ShouldBind(data)
	if err != nil {
		httpserver.Failure(c, ErrParamParseCodeKey, err.Error())

		return
	}

	mongobase := d.getMongoBase(c)
	_, err = mongobase.Create(ctx, data)
	if err != nil {
		httpserver.Failure(c, ErrCreateFailedCodeKey, err.Error())

		return
	}

	httpserver.Success(c, data)
}

// 分页查询
//
// 只适用于 mongodb
//
// 查询参数结构体属性为指针, 反射遍历, 拼接mongo filter为 ==
// Limit 与 offset 字段
func (d *Crud) ListController(c *gin.Context) {
	// s := d.search
	// s := Stu{}
	total, results, err := d.list(c)
	if err != nil {
		httpserver.Failure(c, ErrParamParseCodeKey, err.Error())

		return
	}

	resultVo := ResulListtVO{
		Data:       results,
		TotalCount: total,
	}

	httpserver.Success(c, resultVo)
}

func (d *Crud) list(c *gin.Context) (int64, interface{}, error) {
	mongobase := d.getMongoBase(c)
	s := d.param.NewSearch(c)
	err := c.BindQuery(s)
	if err != nil {
		return 0, nil, err
	}

	var (
		limit  = int64(10)
		offset = int64(0)
	)

	filter := bson.M{}
	refType := reflect.TypeOf(s).Elem()
	refValue := reflect.ValueOf(s).Elem()

	fieldsN := refValue.NumField()
	for i := 0; i < fieldsN; i++ {
		sf := refType.Field(i)
		rv := refValue.Field(i)

		if sf.Type.Kind() == reflect.Ptr {
			// sf.Type.Elem().
			if rv.IsNil() {
				continue
			}

			if sf.Name == "Limit" {
				limit = rv.Elem().Int()
				continue
			}

			if sf.Name == "Offset" {
				offset = rv.Elem().Int()
				continue
			}

			bsTagStr := sf.Tag.Get("bson")
			bsTags := strings.Split(bsTagStr, ",")
			name := sf.Name
			if bsTags[0] != "" {
				name = bsTags[0]
			}

			v := rv.Elem().Interface()
			// fmt.Printf("%v %s:%v\n", len(bsTags), name, v)
			filter[name] = v
		}
	}

	pageOrder := NewPageOrder(limit, offset)

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()

	list := d.param.NewListResult(c)
	total, results, err := mongobase.FindWithPage(ctx, filter, pageOrder, list)
	if err != nil {
		return 0, nil, err
	}

	return total, results, nil
}

func (d *Crud) InfoController(c *gin.Context) {
	mongobase := d.getMongoBase(c)
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()

	ID, ok := c.Params.Get("id")
	if !ok {
		httpserver.Failure(c, ErrParamParseCodeKey, "")

		return
	}

	// fmt.Printf("id: %s\n", ID)

	result := d.param.NewModel(c)
	// result := Student{}
	err := mongobase.FindByID(ctx, ID, &result)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			httpserver.Success(c, map[string]string{})

			return
		}

		httpserver.Failure(c, FailureExit, err.Error())

		return
	}

	// fmt.Printf("%+v\n", result)

	httpserver.Success(c, result)
}

func (d *Crud) UpdateController(c *gin.Context) {
	mongobase := d.getMongoBase(c)
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()
	updateParam := d.param.NewUpdate(c)
	if err := c.ShouldBind(&updateParam); err != nil {
		httpserver.Failure(c, ErrParamParseCodeKey, err.Error())

		return
	}

	ID, ok := c.Params.Get("id")
	if !ok {
		httpserver.Failure(c, ErrParamParseCodeKey, "")

		return
	}

	err := mongobase.UpdateByID(ctx, ID, updateParam)
	if err != nil {
		httpserver.Failure(c, FailureExit, err.Error())

		return
	}

	httpserver.Success(c, nil)
}

func (d *Crud) DeleteController(c *gin.Context) {
	mongobase := d.getMongoBase(c)
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()
	param := &DeleteParam{}
	if err := c.ShouldBind(&param); err != nil {
		httpserver.Failure(c, ErrParamParseCodeKey, err.Error())

		return
	}

	ids := param.IDs
	_, err := mongobase.DeleteByIds(ctx, ids)
	if err != nil {
		httpserver.Failure(c, FailureExit, err.Error())

		return
	}

	httpserver.Success(c, nil)
}

// 导出 Excel
func (d *Crud) ExportExcelController(c *gin.Context) {
	_, results, err := d.list(c)
	if err != nil {
		httpserver.Failure(c, ErrParamParseCodeKey, err.Error())

		return
	}

	// columns := []ExcelColumn{{Key:"name",Name:"姓名"},{Key:"age",Name:"年龄"},{Key:"location",Name:"住址"}, {Key:"update_time",Name: "更新时间", ExportFormat:timeStampExportFormat}}
	columnsMap := make(map[string]ExcelColumn)

	excelFileName := d.param.ExcelName()

	excelFile := excelize.NewFile()
	excelFile.SetSheetName("Sheet1", excelFileName)

	for i, column := range d.exportColumns {
		col, _ := excelize.ColumnNumberToName(i + 1)
		axis := col + "1"
		// fmt.Printf("%v %v\n", col, axis)
		excelFile.SetCellValue(excelFileName, axis, column.Name)
		columnsMap[column.Key] = column
	}

	// rType :=reflect.TypeOf(results).Elem()
	rValue := reflect.ValueOf(results).Elem()
	// 获取 columnsMap 中部分不是数据库中的字段
	elemLen := rValue.Len()

	for r := 0; r < elemLen; r++ {
		elemValue := rValue.Index(r)
		elemType := elemValue.Type()
		numFields := elemValue.NumField()
		for c := 0; c < numFields; c++ {
			fieldType := elemType.Field(c)
			tagKeys := strings.Split(fieldType.Tag.Get("json"), ",")
			name := tagKeys[0]
			index, ok := d.tileMap[name]
			if !ok {
				continue
			}

			fieldValue := elemValue.Field(c)
			v := fieldValue.Interface()
			column, ok := columnsMap[name]
			if ok && column.ExportFormat != nil {
				v = column.ExportFormat(fieldValue.Interface())
			}

			col, _ := excelize.ColumnNumberToName(index)
			axis := col + strconv.FormatInt(int64(r)+2, 10)
			err := excelFile.SetCellValue(excelFileName, axis, v)
			if err != nil {
				//logger.Warn(err)
				continue
			}
		}
		// TODO: columnsMap 中部分不是数据库中的字段处理
	}
	// excelFile.SaveAs("./"+excelFileName+".xlsx")
	fileName := excelFileName + ".xlsx"
	fileName = url.QueryEscape(fileName) // 防止中文乱码
	c.Writer.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename=%s;filename*=utf-8''%s`, fileName, fileName))

	buff, err := excelFile.WriteToBuffer()
	if err != nil {
		httpserver.Failure(c, FailureExit, err.Error())
	}
	// c.Header("Content-Length", strconv.FormatInt(int64(buff.Len()), 10))
	c.Data(200, "application/vnd.ms-excel;charset=UTF-8", buff.Bytes())
}

func (d *Crud) UploadExcelController(c *gin.Context) {
	// panic("no implement upload execel api")
	c.JSON(200, gin.H{
		"msg": "no implement api",
	})
}

// It defines
//   POST: /path
//   GET:  /path
//   PUT:  /path/:id
//   POST: /path/:id
func CRUD(group *gin.RouterGroup, path string, resource interface{}) {
	if resource, ok := resource.(CreateSupported); ok {
		group.POST(path, resource.CreateController)
	}

	if resource, ok := resource.(ListSupported); ok {
		group.GET(path, resource.ListController)
	}

	if resource, ok := resource.(InfoSupported); ok {
		group.GET(path+"/:id", resource.InfoController)
	}

	if resource, ok := resource.(UpdateSupported); ok {
		group.POST(path+"/upd/:id", resource.UpdateController)
	}

	if resource, ok := resource.(DeleteSupported); ok {
		group.POST(path+"del", resource.DeleteController)
	}

	if resource, ok := resource.(ExportExcelSupported); ok {
		group.GET(path+"/excel", resource.ExportExcelController)
	}

	if resource, ok := resource.(UploadExcelSupported); ok {
		group.POST(path+"/excel", resource.UploadExcelController)
	}
}
