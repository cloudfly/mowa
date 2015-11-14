package mowa

import (
	"github.com/stretchr/testify/assert"
	"net/http"
	"testing"
)

var testC *Context

func init() {
	req, _ := http.NewRequest("Get", "http://localhost:1234/hello/world?name=chen&age=25&name=yun", nil)
	testC = &Context{
		Request: req,
	}
}

func TestContext_Query(t *testing.T) {
	name := testC.Query("name", "")
	assert.Equal(t, name, "chen", "name should be chen")
}

func TestContext_QuerySlice(t *testing.T) {
	names := testC.QuerySlice("name", []string{})
	t.Log(names)
	assert.Equal(t, names, []string{"chen", "yun"}, "name shouldbe [chen, yun]")
}

func TestContext_AssertNil(t *testing.T) {
	testC.AssertNil(nil)
}

func TestContext_AssertEqual(t *testing.T) {
	testC.AssertEqual(nil, nil)
	testC.AssertEqual(12, 12)
	testC.AssertEqual("hello", "hello")
	testC.AssertEqual([]int{1}, []int{1})
}

func TestContext_AssertNotEqual(t *testing.T) {
	testC.AssertNotEqual(nil, map[string]string{})
	testC.AssertNotEqual(12, 12.0)
	testC.AssertNotEqual("hell", "hello")
	testC.AssertNotEqual([]int{1, 2}, []int{1})
}

func TestContext_AssertZero(t *testing.T) {
	var (
		i  int
		s  string
		b  bool
		f  float32
		m  map[int]string
		sl []int
	)
	testC.AssertEmpty(i)
	testC.AssertEmpty(s)
	testC.AssertEmpty(b)
	testC.AssertEmpty(m)
	testC.AssertEmpty(f)
	testC.AssertEmpty(sl)
}
