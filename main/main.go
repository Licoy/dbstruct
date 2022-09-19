package main

import (
	"fmt"
	"github.com/Licoy/dbstruct"
)

func main() {
	err := dbstruct.NewDBStruct().
		Dsn("root:123qweasd@tcp(127.0.0.1:3306)/atome_afterpay_cms?charset=utf8").
		StructNameFmt(dbstruct.FmtUnderlineToStartUpHump).
		FieldNameFmt(dbstruct.FmtUnderlineToStartUpHump).
		FileNameFmt(dbstruct.FmtUnderline).
		AppendTable("user_group").
		//SingleFile(true).
		//GenTableNameFunc(true).
		//GenTableName("user_group").
		//TagJson(true).
		PackageName("entity").
		ModelPath("./entity").
		GenComment(false).
		StructNameSuffix("Entity").
		FileNameSuffix("_entity").
		TagOrm(false).
		AppendTag(dbstruct.NewTag("json", dbstruct.FmtUnderlineToStartLowHump)).
		AppendTag(dbstruct.NewTag("mapstructure", dbstruct.FmtDefault)).
		Generate()
	if err != nil {
		fmt.Println(err)
	}

}
