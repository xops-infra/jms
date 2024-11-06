package model_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/xops-infra/jms/model"
)

// Test ArrayString Contains
func TestArrayString_Contains(t *testing.T) {
	{
		a := model.ArrayString{"zhangsan"}
		assert.True(t, a.Contains("zhangsan"))
		assert.False(t, a.Contains("lisi"))
	}
	{
		a := model.ArrayString{"*"}
		assert.True(t, a.Contains("zhangsan"))
		assert.True(t, a.Contains("lisi"))
	}
	{
		a := model.ArrayString{"!zhangsan"}
		assert.True(t, a.Contains("lisi"))
		assert.False(t, a.Contains("zhangsan"))
	}
	{
		a := model.ArrayString{"zhang*"}
		assert.False(t, a.Contains("lisi"))
		assert.True(t, a.Contains("zhangsan"))
	}
	{
		a := model.ArrayString{"zhang*", "*"}
		assert.True(t, a.Contains("!lisi"))
		assert.True(t, a.Contains("zhangsan"))
	}
}
