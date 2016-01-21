Mowa
====

My Own Web Api Framework

This is a very simple golang web api framework for personal usage.

##Demo

```golang

<<<<<<< HEAD
api := mowa.New()
=======
func preLog(c *mowa.Context) {
	logrus.Infof("%s %s", c.Request.Method, c.Request.URL)
}
>>>>>>> 21f39f5fcaf6bbe787b1e76592689ca012d4147e

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
