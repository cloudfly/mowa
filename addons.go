package mowa

// 内置一些默认的 path，提供一些通用的功能

import (
	"context"
	"encoding/json"

	"github.com/cloudfly/golang/cluster"
	"github.com/cloudfly/golang/dconf"
)

type ctxKey string

var (
	clusterContextKey ctxKey = "cluster"
	dconfContextKey   ctxKey = "dconf"
)

// ClusterNodes 返回 cluster 节点列表
func ClusterNodes(ctx *Context) interface{} {
	clt, ok := ctx.Value(clusterContextKey).(*cluster.Cluster)
	if !ok {
		return ErrorWithCode(404, "not found")
	}
	return Data(clt.Nodes())
}

// ClusterAddNode 向集群增加节点
func ClusterAddNode(ctx *Context) interface{} {
	clt := ctx.Value(clusterContextKey).(*cluster.Cluster)
	var node cluster.Node
	if err := json.Unmarshal(ctx.ReadBody(), &node); err != nil {
		return ErrorWithCode(400, err)
	}
	if err := clt.AddNode(node); err != nil {
		return ErrorWithCode(400, err)
	}
	return Data("CREATED")
}

// ClusterUpdateNode 更新集群节点
func ClusterUpdateNode(ctx *Context) interface{} {
	clt := ctx.Value(clusterContextKey).(*cluster.Cluster)
	var node cluster.Node
	if err := json.Unmarshal(ctx.ReadBody(), &node); err != nil {
		return ErrorWithCode(400, err)
	}
	if err := clt.UpdateNode(node); err != nil {
		return ErrorWithCode(400, err)
	}
	return Data("UPDATED")
}

// ClusterRemoveNode 删除集群节点
func ClusterRemoveNode(ctx *Context) interface{} {
	clt := ctx.Value(clusterContextKey).(*cluster.Cluster)
	name := ctx.String("name", "")
	if name == "" {
		return ErrorWithCode(400, "node name required")
	}
	if err := clt.RemoveNode(name); err != nil {
		return ErrorWithCode(400, err)
	}
	return Data("DELETED")
}

// ConfigRead read a value from dconf
func ConfigRead(ctx *Context) interface{} {
	conf := ctx.Value(dconfContextKey).(dconf.DConf)
	key := ctx.String("key", "")
	if key == "" {
		return ErrorWithCode(400, "key required")
	}
	v, err := conf.Get(key)
	if err != nil {
		return ErrorWithCode(404, err)
	}
	return Data(v)
}

// ConfigWrite write a key-value into dconf
func ConfigWrite(ctx *Context) interface{} {
	conf := ctx.Value(dconfContextKey).(dconf.DConf)
	key := ctx.String("key", "")
	if key == "" {
		return ErrorWithCode(400, "key required")
	}
	if err := conf.Set(key, string(ctx.ReadBody()), ctx.Query("pre_exist", "false") == "true"); err != nil {
		return ErrorWithCode(404, err)
	}
	return Data("OK")
}

// ConfigDelete del a key-value from dconf
func ConfigDelete(ctx *Context) interface{} {
	conf := ctx.Value(dconfContextKey).(dconf.DConf)
	key := ctx.String("key", "")
	if key == "" {
		return ErrorWithCode(400, "key required")
	}
	if err := conf.Del(key); err != nil {
		return ErrorWithCode(500, err)
	}
	return Data("OK")
}

// ConfigKeys get keys list by prefix
func ConfigKeys(ctx *Context) interface{} {
	conf := ctx.Value(dconfContextKey).(dconf.DConf)
	return Data(conf.Keys(ctx.String("prefix", "")))
}

// ConfigData get key-values by prefix
func ConfigData(ctx *Context) interface{} {
	conf := ctx.Value(dconfContextKey).(dconf.DConf)
	return Data(conf.Data(ctx.String("prefix", "")))
}

// WithCluster add http api for cluster management
func WithCluster(api *Mowa, clt *cluster.Cluster) *Mowa {
	if clt == nil {
		return api
	}
	api.ctx = context.WithValue(api.ctx, clusterContextKey, clt)
	api.Get("/debug/cluster/nodes", ClusterNodes)
	api.Post("/debug/cluster/nodes", ClusterAddNode)
	api.Put("/debug/cluster/nodes", ClusterUpdateNode)
	api.Delete("/debug/cluster/nodes/:name", ClusterRemoveNode)
	return api
}

// WithDConf add http api for dconf
func WithDConf(api *Mowa, conf *dconf.DConf) *Mowa {
	if conf == nil {
		return api
	}
	api.ctx = context.WithValue(api.ctx, dconfContextKey, conf)
	api.Get("/debug/config/:key", ConfigRead)
	api.Post("/debug/config/:key", ConfigWrite)
	api.Delete("/debug/config/:key", ConfigDelete)
	api.Get("/debug/config_keys/:prefix", ConfigKeys)
	api.Get("/debug/config_data/:prefix", ConfigData)
	return api
}
