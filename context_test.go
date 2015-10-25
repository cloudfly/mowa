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

func TestTestValue(t *testing.T) {
}

func TestString(t *testing.T) {
}

func TestInt(t *testing.T) {
}

func TestQuery(t *testing.T) {
	name := testC.Query("name", "")
	assert.Equal(t, name, "chen", "name should be chen")
}

func TestQuerySlice(t *testing.T) {
}

func TestForm(t *testing.T) {
}
