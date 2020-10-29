package dbstruct

import (
	"fmt"
	"testing"
)

func TestNewDBStruct(t *testing.T) {
	dbStruct := NewDBStruct()
	err := dbStruct.
		Dsn("root:root@tcp(127.0.0.1:3306)/hm?charset=utf8").
		StructNameFmt(FmtUnderlineToStartUpHump).
		FieldNameFmt(FmtUnderlineToStartUpHump).
		FileNameFmt(FmtUnderline).
		SingleFile(true).
		GenTableNameFunc(true).
		GenTableName("TableName").
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

func TestGetNameFormat(t *testing.T) {
	dbStruct := NewDBStruct()
	fmt.Println(dbStruct.getFormatName("user_name", FmtUnderlineToStartLowHump))
	fmt.Println(dbStruct.getFormatName("user_all_name", FmtUnderlineToStartUpHump))
	fmt.Println(dbStruct.getFormatName("PlayerInfo", FmtUnderline))
	fmt.Println(dbStruct.getFormatName("user_name", FmtDefault))
}
