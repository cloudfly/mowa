Mowa
====

My Own Web Api Framework

This is a golang web api framework for personal using.

##TODO

1. parse application/json body
2. context use google context
3. log use logrus
4. complete context's TestValue function

##Demo

```golang

func preLog(c *mowa.Context) {
	logrus.Infof("%s %s", c.Request.Method, c.Request.URL)
}

func postLog(c *mowa.Context) {
	logrus.Infof("Response %d, %s", c.Code, c.Data)
}



func main(){
        api := mowa.New()
	server.PreHook(preLog)
	server.PostHook(postLog)

        api.Get("/debug", func(c *mowa.Context) (int, interface{}) {
            return 200, "debug"
        })

        v1 := api.Group("/api/v1")
        v1.Get("/hello", func(c *mowa.Context) (int, interface{}) {
            return 200, "hello world!"
        })

        api.Run(":10000")
}
```
