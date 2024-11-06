package app_test

import (
	"fmt"
	"testing"

	"github.com/alibabacloud-go/tea/tea"
	"github.com/xops-infra/jms/model"
)

// TEST SHOW CONFIG
func TestConfig(t *testing.T) {
	fmt.Println(tea.Prettify(model.Conf))
}
