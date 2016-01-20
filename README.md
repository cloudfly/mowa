Mowa
====

My Own Web Api Framework

This is a very simple golang web api framework for personal usage.

##Demo

```golang

api := mowa.New()

api.Get("/debug", func(c *mowa.Context) (int, interface{}) {
    return 200, "debug"
})

v1 := api.Group("/api/v1")
v1.Get("/hello", func(c *mowa.Context) (int, interface{}) {
    return 200, "hello world!"
})

api.Run(":10000")
```
