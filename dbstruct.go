package dbstruct

import (
	"bytes"
	"database/sql"
	"errors"
	"fmt"
	_ "github.com/go-sql-driver/mysql"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"
	"unicode"
)

var types = map[string]string{
	"int":                "int",
	"integer":            "int",
	"tinyint":            "int8",
	"smallint":           "int16",
	"mediumint":          "int32",
	"bigint":             "int64",
	"int unsigned":       "int64",
	"integer unsigned":   "int64",
	"tinyint unsigned":   "int64",
	"smallint unsigned":  "int64",
	"mediumint unsigned": "int64",
	"bigint unsigned":    "int64",
	"bit":                "int64",
	"float":              "float64",
	"double":             "float64",
	"decimal":            "float64",
	"binary":             "string",
	"varbinary":          "string",
	"enum":               "string",
	"set":                "string",
	"varchar":            "string",
	"char":               "string",
	"tinytext":           "string",
	"mediumtext":         "string",
	"text":               "string",
	"longtext":           "string",
	"blob":               "string",
	"tinyblob":           "string",
	"mediumblob":         "string",
	"longblob":           "string",
	"bool":               "bool",
	"date":               "time.Time",
	"datetime":           "time.Time",
	"timestamp":          "time.Time",
	"time":               "time.Time",
}

type FmtMode uint16

const (
	FmtDefault                 FmtMode = iota //默认(和表名一致)
	FmtUnderlineToStartUpHump                 //下划线转开头大写驼峰
	FmtUnderlineToStartLowHump                //下划线开头小写驼峰
	FmtUnderline                              //下划线格式
)

type dbStruct struct {
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
	db               *sql.DB
	err              error
}

func NewDBStruct() *dbStruct {
	return &dbStruct{fieldNameFmt: FmtDefault, structNameFmt: FmtDefault, fileNameFmt: FmtDefault,
		genTableName: "TableName"}
}

func (ds *dbStruct) Dsn(v string) *dbStruct {
	ds.dsn = v
	return ds
}

func (ds *dbStruct) GenTableName(v string) *dbStruct {
	ds.genTableName = v
	return ds
}

func (ds *dbStruct) PackageName(v string) *dbStruct {
	ds.packageName = v
	return ds
}

func (ds *dbStruct) GenTableNameFunc(v bool) *dbStruct {
	ds.genTableNameFunc = v
	return ds
}

func (ds *dbStruct) ModelPath(v string) *dbStruct {
	ds.modelPath = v
	return ds
}

func (ds *dbStruct) SingleFile(v bool) *dbStruct {
	ds.singleFile = v
	return ds
}

func (ds *dbStruct) FileNameFmt(v FmtMode) *dbStruct {
	ds.fileNameFmt = v
	return ds
}

func (ds *dbStruct) FieldNameFmt(v FmtMode) *dbStruct {
	ds.fieldNameFmt = v
	return ds
}

func (ds *dbStruct) StructNameFmt(v FmtMode) *dbStruct {
	ds.structNameFmt = v
	return ds
}

func (ds *dbStruct) AppendTable(v string) *dbStruct {
	if ds.tables == nil {
		ds.tables = make([]string, 0, 10)
	}
	ds.tables = append(ds.tables, v)
	return ds
}

func (ds *dbStruct) TagJson(v bool) *dbStruct {
	ds.tagJson = v
	return ds
}

func (ds *dbStruct) TagOrm(v bool) *dbStruct {
	ds.tagOrm = v
	return ds
}

func (ds *dbStruct) AppendTag(v *Tag) *dbStruct {
	if ds.tags == nil {
		ds.tags = make([]*Tag, 0, 10)
	}
	ds.tags = append(ds.tags, v)
	return ds
}

type Tag struct {
	TagName string
	Mode    FmtMode
}

type column struct {
	Name     string
	Type     string
	Nullable string
	Table    string
	Comment  string
}

func NewTag(tagName string, mode FmtMode) *Tag {
	return &Tag{TagName: tagName, Mode: mode}
}

func (ds *dbStruct) connectDB() {
	if ds.db == nil {
		ds.db, ds.err = sql.Open("mysql", ds.dsn)
	}
}

type genStructRes struct {
	structName        string
	content           string
	err               error
	needImportTimePkg bool
}

//生成
func (ds *dbStruct) Generate() (err error) {
	startTime := time.Now().UnixNano()
	if ds.dsn == "" {
		return errors.New("DSN未配置")
	}
	ds.connectDB()
	if ds.err != nil {
		return ds.err
	}
	if ds.tagJson {
		ds.AppendTag(NewTag("json", FmtDefault))
	}
	if ds.tagOrm {
		ds.AppendTag(NewTag("orm", FmtDefault))
	}
	//tables := make(map[string][]column)
	tables, err := ds.getTables()
	if err != nil {
		return
	}

	writes := make(map[string]string)
	gch := make(chan *genStructRes, len(tables))

	group := &sync.WaitGroup{}
	group.Add(len(tables))

	for table, columns := range tables {
		go ds.genStruct(table, columns, gch, group)
	}

	group.Wait()

	close(gch)

	needImportTimePkg := false

	for {
		gchRes, ok := <-gch
		if !ok {
			break
		}
		writes[gchRes.structName] = gchRes.content
		if !needImportTimePkg {
			needImportTimePkg = gchRes.needImportTimePkg
		}
	}

	if ds.modelPath == "" {
		ds.modelPath, err = os.Getwd()
		if err != nil {
			return err
		}
		if ds.singleFile {
			ds.modelPath += "/model/models.go"
		} else {
			ds.modelPath += "/model"
		}
	}

	if ds.singleFile {

		dir, _ := filepath.Split(ds.modelPath)

		err = os.MkdirAll(dir, os.ModePerm)
		if err != nil {
			log.Println("base path create fail.")
			return err
		}

		_, err := os.Create(ds.modelPath)
		if err != nil {
			return err
		}

		finalContent := bytes.Buffer{}
		finalContent.WriteString(fmt.Sprintf("package %s\n\n", ds.packageName))
		if needImportTimePkg {
			finalContent.WriteString("import \"time\"\n\n")
		}
		for _, content := range writes {
			finalContent.WriteString(content)
			finalContent.WriteString("\n\n\n")
		}
		err = ds.writeStruct(ds.modelPath, finalContent.String())
		if err != nil {
			log.Fatalf("write struct fail(%s) : %s ", ds.modelPath, err.Error())
			return err
		}

		cmd := exec.Command("gofmt", "-w", ds.modelPath)
		_ = cmd.Run()

	} else {

		err = os.MkdirAll(ds.modelPath, os.ModePerm)
		if err != nil {
			log.Println("base path create fail.")
			return err
		}

		group.Add(len(writes))

		for name, content := range writes {
			go ds.writeManyFile(name, content, group)
		}

		group.Wait()

	}

	endTime := time.Now().UnixNano()

	log.Printf("DbStruct生成完成，累计耗时 %d 毫秒", (endTime-startTime)/1e6)

	return
}

func (ds *dbStruct) writeManyFile(name string, content string, group *sync.WaitGroup) {
	defer group.Done()
	filename := ds.getFormatName(name, ds.fileNameFmt)
	filename = fmt.Sprintf("%s/%s.go", ds.modelPath, filename)
	err := ds.writeStruct(filename, content)
	if err != nil {
		log.Fatalf("write struct fail(%s) : %s ", filename, err.Error())
	}
	cmd := exec.Command("gofmt", "-w", filename)
	_ = cmd.Run()
}

func (ds *dbStruct) getFormatName(s string, m FmtMode) (res string) {
	switch m {
	case FmtUnderlineToStartUpHump:
		{
			split := strings.Split(s, "_")
			res = ""
			for _, v := range split {
				res += strings.ToUpper(v[0:1]) + v[1:]
			}
		}
	case FmtUnderlineToStartLowHump:
		{
			split := strings.Split(s, "_")
			res = ""
			for i, v := range split {
				if i == 0 {
					res += strings.ToLower(v[0:1])
				} else {
					res += strings.ToUpper(v[0:1])
				}
				res += v[1:]
			}
		}
	case FmtUnderline:
		{
			b := bytes.Buffer{}
			for i, v := range s {
				if unicode.IsUpper(v) {
					if i != 0 {
						b.WriteString("_")
					}
					b.WriteString(string(unicode.ToLower(v)))
				} else {
					b.WriteString(string(v))
				}
			}
			res = b.String()
		}
	case FmtDefault:
		res = s
	}
	return
}

func (ds *dbStruct) getColumnGoType(dbType string) (res string) {
	res, has := types[dbType]
	if !has {
		res = "string"
		return
	}
	return
}

func (ds *dbStruct) genStruct(table string, columns []column, ch chan *genStructRes, group *sync.WaitGroup) {
	defer group.Done()
	buffer := bytes.Buffer{}
	res := &genStructRes{"", "", nil, false}
	structName := ds.getFormatName(table, ds.structNameFmt)
	res.structName = structName
	if !ds.singleFile {
		buffer.WriteString(fmt.Sprintf("package %s\n\n {@importTimePkg@}\n\n ", ds.packageName))
	}
	buffer.WriteString(fmt.Sprintf("type %s struct {\n", structName))
	importTimePkgStr := ""
	for _, column := range columns {
		columnName := ds.getFormatName(column.Name, ds.fieldNameFmt)
		goType := ds.getColumnGoType(column.Type)
		if importTimePkgStr == "" && goType == "time.Time" && !ds.singleFile {
			importTimePkgStr = "import \"time\"\n\n"
		} else if !res.needImportTimePkg && ds.singleFile && goType == "time.Time" {
			res.needImportTimePkg = true
		}
		tagString := ""
		if ds.tags != nil && len(ds.tags) > 0 {
			tagString = "`"
			for i, tag := range ds.tags {
				if i == len(ds.tags)-1 {
					tagString += fmt.Sprintf("%s:\"%s\"", tag.TagName, ds.getFormatName(column.Name, tag.Mode))
				} else {
					tagString += fmt.Sprintf("%s:\"%s\" ", tag.TagName, ds.getFormatName(column.Name, tag.Mode))
				}
			}
			tagString += "`"
		}
		//字段释义
		if column.Comment != "" {
			buffer.WriteString(fmt.Sprintf("\t// %s\n", column.Comment))
		}
		//字段结构
		buffer.WriteString(fmt.Sprintf("\t%s %s %s\n", columnName, goType, tagString))
	}
	buffer.WriteString("}\n\n")
	if ds.genTableNameFunc && ds.genTableName != "" {
		buffer.WriteString(fmt.Sprintf("func (%s *%s) %s() string {\n\treturn \"%s\"\n}", strings.ToLower(structName[0:1]),
			structName, ds.genTableName, table))
	}
	content := buffer.String()
	content = strings.Replace(content, "{@importTimePkg@}", importTimePkgStr, 1)
	res.content = content
	ch <- res
}

func (ds *dbStruct) getTables() (tables map[string][]column, err error) {
	tableIn := ""
	if ds.tables != nil && len(ds.tables) > 0 {
		buff := bytes.Buffer{}
		buff.WriteString("AND TABLE_NAME IN (")
		for i, tableName := range ds.tables {
			buff.WriteString("'")
			buff.WriteString(tableName)
			buff.WriteString("'")
			if i != len(ds.tables)-1 {
				buff.WriteString(", ")
			}
		}
		buff.WriteString(")")
		tableIn = buff.String()
	}
	sqlString := fmt.Sprintf("SELECT COLUMN_NAME AS `Name`,DATA_TYPE AS `Type`,IS_NULLABLE AS `Nullable`,TABLE_NAME AS "+
		"`Table`,COLUMN_COMMENT AS `Comment` FROM information_schema.COLUMNS WHERE table_schema=DATABASE () %s ORDER BY"+
		" TABLE_NAME ASC", tableIn)
	rows, err := ds.db.Query(sqlString)
	if err != nil {
		return nil, err
	}

	defer func() {
		qerr := rows.Close()
		if qerr != nil {
			log.Fatalf("关闭数据查询结果异常：%s", qerr.Error())
		}
	}()

	tables = make(map[string][]column, 3)

	for rows.Next() {
		c := column{}
		err := rows.Scan(&c.Name, &c.Type, &c.Nullable, &c.Table, &c.Comment)
		if err != nil {
			return nil, err
		}
		_, has := tables[c.Table]
		if !has {
			tables[c.Table] = make([]column, 0, 3)
		}
		tables[c.Table] = append(tables[c.Table], c)
	}

	return
}

func (ds *dbStruct) writeStruct(filepath string, content string) (err error) {
	b := []byte(content)
	err = ioutil.WriteFile(filepath, b, os.ModePerm)
	if err != nil {
		return err
	}
	return nil
}
