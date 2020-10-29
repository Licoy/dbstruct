## 介绍
`dbstruct`是一款将数据库表一键转换为Golang Struct的应用程序，支持自定义Tag和多种命名格式配置。
## 参数列表
```go
        dsn              string   //数据库链接
	tables           []string //自定义表
	tagJson          bool     //json tag
	tagOrm           bool     //orm tag
	fieldNameFmt     FmtMode  //字段名称格式
	structNameFmt    FmtMode  //结构名格式
	fileNameFmt      FmtMode  //文件名格式
	genTableName     string   //TableName方法名，
	genTableNameFunc bool     //是否生成TableName方法
	modelPath        string   //model保存的路径，若singleFile==true，则填写model.go的完整路径，默认为当前路径
	singleFile       bool     //是否合成一个单文件
	packageName      string   //包名
	tags             []*Tag   //自定义Tag列表
```
## 使用方法
### 安装依赖
```shell script
go get github.com/Licoy/dbstruct
```
### 使用
```go
func TestNewDBStruct(t *testing.T) {
	dbStruct := NewDBStruct()
	err := dbStruct.
		Dsn("root:root@tcp(127.0.0.1:3306)/hm?charset=utf8").
		StructNameFmt(FmtUnderlineToStartUpHump).
		FieldNameFmt(FmtUnderlineToStartUpHump).
		FileNameFmt(FmtUnderline).
		SingleFile(true).
		GenTableNameFunc(true).
		GenTableName("MyTableName").
		TagJson(true).
		PackageName("model").
		TagOrm(true).
		AppendTag(NewTag("xml", FmtDefault)).
		Generate()
	if err != nil {
		t.Error(err)
	} else {
		fmt.Println("ok.")
	}
}
```
### 生成结果示例
```go
package model

type Mart struct {
	Id         int64 `xml:"id" json:"id" orm:"id" `
	ActId      int64 `xml:"act_id" json:"act_id" orm:"act_id" `
	Created    int64 `xml:"created" json:"created" orm:"created" `
	TemplateId int   `xml:"template_id" json:"template_id" orm:"template_id" `
	Number     int   `xml:"number" json:"number" orm:"number" `
	Price      int   `xml:"price" json:"price" orm:"price" `
	Day        int   `xml:"day" json:"day" orm:"day" `
}

func (m *Mart) MyTableName() string {
	return "mart"
}
```
## 授权
[MIT](./LICENSE)