package internal

// 内置一些默认的 path，提供一些通用的功能

import (
	"encoding/json"

	"github.com/cloudfly/golang/cluster"
	"github.com/cloudfly/golang/dconf"
	"github.com/cloudfly/mowa"
	"github.com/valyala/fasthttp"
)

type ctxKey string

var (
	clt  *cluster.Cluster
	conf dconf.DConf
)

// ClusterNodes 返回 cluster 节点列表
func ClusterNodes(ctx *fasthttp.RequestCtx) interface{} {
	return mowa.Data(clt.Nodes())
}

// ClusterAddNode 向集群增加节点
func ClusterAddNode(ctx *fasthttp.RequestCtx) interface{} {
	var node cluster.Node
	if err := json.Unmarshal(ctx.ReadBody(), &node); err != nil {
		return mowa.ErrorWithCode(400, err)
	}
	if err := clt.AddNode(node); err != nil {
		return mowa.ErrorWithCode(400, err)
	}
	return mowa.Data("CREATED")
}

// ClusterUpdateNode 更新集群节点
func ClusterUpdateNode(ctx *fasthttp.RequestCtx) interface{} {
	var node cluster.Node
	if err := json.Unmarshal(ctx.ReadBody(), &node); err != nil {
		return mowa.ErrorWithCode(400, err)
	}
	if err := clt.UpdateNode(node); err != nil {
		return mowa.ErrorWithCode(400, err)
	}
	return mowa.Data("UPDATED")
}

// ClusterRemoveNode 删除集群节点
func ClusterRemoveNode(ctx *fasthttp.RequestCtx) interface{} {
	name := mowa.StringValue(ctx, "name", "")
	if name == "" {
		return mowa.ErrorWithCode(400, "node name required")
	}
	if err := clt.RemoveNode(name); err != nil {
		return mowa.ErrorWithCode(400, err)
	}
	return mowa.Data("DELETED")
}

// ConfigRead read a value from dconf
func ConfigRead(ctx *fasthttp.RequestCtx) interface{} {
	key := mowa.StringValue(ctx, "key", "")
	if key == "" {
		return mowa.ErrorWithCode(400, "key required")
	}
	v, err := conf.Get(key)
	if err != nil {
		return mowa.ErrorWithCode(404, err)
	}
	return mowa.Data(v)
}

// ConfigWrite write a key-value into dconf
func ConfigWrite(ctx *fasthttp.RequestCtx) interface{} {
	key := mowa.StringValue(ctx, "key", "")
	if key == "" {
		return mowa.ErrorWithCode(400, "key required")
	}
	if err := conf.Set(key, string(ctx.ReadBody()), ctx.Query("pre_exist", "false") == "true"); err != nil {
		return mowa.ErrorWithCode(404, err)
	}
	return mowa.Data("OK")
}

// ConfigDelete del a key-value from dconf
func ConfigDelete(ctx *fasthttp.RequestCtx) interface{} {
	key := mowa.StringValue(ctx, "key", "")
	if key == "" {
		return mowa.ErrorWithCode(400, "key required")
	}
	if err := conf.Del(key); err != nil {
		return mowa.ErrorWithCode(500, err)
	}
	return mowa.Data("OK")
}

// ConfigKeys get keys list by prefix
func ConfigKeys(ctx *fasthttp.RequestCtx) interface{} {
	return mowa.Data(conf.Keys(mowa.StringValue(ctx, "prefix", "")))
}

// ConfigData get key-values by prefix
func ConfigData(ctx *fasthttp.RequestCtx) interface{} {
	return mowa.Data(conf.Data(mowa.StringValue(ctx, "prefix", "")))
}

// WithCluster add http api for cluster management
func WithCluster(api *mowa.Mowa, c *cluster.Cluster) *mowa.Mowa {
	if c == nil {
		return api
	}
	clt = c
	api.Get("/debug/cluster/nodes", ClusterNodes)
	api.Post("/debug/cluster/nodes", ClusterAddNode)
	api.Put("/debug/cluster/nodes", ClusterUpdateNode)
	api.Delete("/debug/cluster/nodes/:name", ClusterRemoveNode)
	return api
}

// WithDConf add http api for dconf
func WithDConf(api *mowa.Mowa, c dconf.DConf) *mowa.Mowa {
	if conf == nil {
		return api
	}
	conf = c
	api.Get("/debug/config/:key", ConfigRead)
	api.Post("/debug/config/:key", ConfigWrite)
	api.Delete("/debug/config/:key", ConfigDelete)
	api.Get("/debug/config_keys/:prefix", ConfigKeys)
	api.Get("/debug/config_data/:prefix", ConfigData)
	return api
}
