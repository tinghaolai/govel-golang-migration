package govel_migration

import (
	"fmt"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"plugin"
	"regexp"
	"strings"
)

func Migrate(mysqlDsn string) {
	db, err := gorm.Open(mysql.Open(mysqlDsn), &gorm.Config{})
	if err != nil {
		panic(err)
	}

	number := getMigrateNumber(db)
	println(number)

	cwd, err := os.Getwd()
	if err != nil {
		panic(err)
	}

	migrationPath := path.Join(cwd, "migrations")
	if _, err := os.Stat(migrationPath); os.IsNotExist(err) {
		fmt.Println("migration folder does not exist: " + migrationPath)
		return
	}

	cmd := exec.Command("go", "build", "-buildmode=plugin")
	cmd.Dir = migrationPath
	err = cmd.Run()
	if err != nil {
		panic("build error")
	}

	soPath := path.Join(migrationPath, "migrations.so")
	plug, err := plugin.Open(soPath)
	if err != nil {
		panic(err)
	}

	err = filepath.Walk(migrationPath, func(migrationPath string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		r := regexp.MustCompile(`^\d+_(.+)\.go$`)
		match := r.FindStringSubmatch(info.Name())

		if len(match) > 1 {
			println(info.Name())
			fmt.Println(match[1])
			fmt.Println(toCamelCase(match[1]))

			functionName := "Up" + toCamelCase(match[1])
			runLib, err := plug.Lookup(functionName)
			if err != nil {
				panic(err)
			}

			runLib.(func())()
		}

		return nil
	})

	if err != nil {
		fmt.Println(err)
	}

}

func getMigrateNumber(db *gorm.DB) int {
	var lastRecord Migration
	err := db.Last(&lastRecord).Error
	if err != nil {
		return 1
	}

	return lastRecord.Batch + 1
}

func toCamelCase(s string) string {
	words := strings.Split(s, "_")
	for i, word := range words {
		words[i] = strings.Title(word)
	}

	return strings.Join(words, "")
}