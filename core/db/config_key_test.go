package db_test

import (
	"encoding/base64"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/alibabacloud-go/tea/tea"
	"github.com/xops-infra/jms/app"
	"github.com/xops-infra/jms/core/db"
)

// AddKey
func TestAddKey(t *testing.T) {
	// 读取pem目录下所有.pem文件
	_ = filepath.Walk("./pem", func(path string, info os.FileInfo, err error) error {
		if info.IsDir() {
			return nil
		}
		// 读取文件内容
		fs, err := os.Open(path)
		if err != nil {
			t.Fatal(err)
		}
		defer fs.Close()
		// base64编码
		pem, err := ioutil.ReadAll(fs)
		if err != nil {
			t.Fatal(err)
		}
		base64Pem := base64.StdEncoding.EncodeToString(pem)
		// 添加到数据库
		_, err = app.App.DBService.AddKey(db.AddKeyRequest{
			KeyName:   tea.String(info.Name()),
			PemBase64: tea.String(base64Pem),
		})
		if err != nil {
			t.Fatal(err)
		}
		return nil
	})

}

// ListKey
func TestListKey(t *testing.T) {
	keys, err := app.App.DBService.ListKey()
	if err != nil {
		t.Fatal(err)
	}
	for _, key := range keys {
		t.Log(tea.Prettify(key))
	}
}
