// dynamic etc management based on etcd service

// each caller should provide
// 1. etcd addr, prefix
// 2. same constructor to unmarshal data from etcd
package dconf

import (
	"context"
	"errors"
	"fmt"
	"path"
	"strings"
	"sync"

	"go.etcd.io/etcd/clientv3"
	"go.etcd.io/etcd/clientv3/clientv3util"
	"go.etcd.io/etcd/mvcc/mvccpb"
)

// DConf3 synchronizes etcd with in-memory data structures
type DConf3 struct {
	ctx context.Context
	// data stores data synchronized with etcd
	// key format: `/prefix/key` is a leaf node of etcd
	// value is a struct instance pointer synchronized from etcd
	data sync.Map

	prefix      string // etcd watcher prefix
	watcher     clientv3.Watcher
	kv          clientv3.KV
	latestIndex uint64
}

// NewV3 inits a DConf3 instance and reads data from etcd
func NewV3(ctx context.Context, addrs []string, prefix string) (*DConf3, error) {
	if prefix == "" {
		prefix = "/dconf"
	}
	if prefix[0] != '/' {
		prefix = "/" + prefix
	}
	if prefix[len(prefix)-1] != '/' {
		prefix = prefix + "/"
	}
	conf := &DConf3{
		ctx:    ctx,
		prefix: prefix,
	}

	c, err := clientv3.New(clientv3.Config{Endpoints: addrs})
	if err != nil {
		return nil, err
	}
	conf.kv = clientv3.NewKV(c)
	conf.watcher = clientv3.NewWatcher(c)

	// initial sync
	if err := conf.sync(); err != nil {
		return nil, err
	}

	go conf.watch()

	return conf, nil
}

func (conf *DConf3) fullpath(key string) string {
	return path.Join(conf.prefix, key)
}

func (conf *DConf3) keyname(path string) string {
	if path == "" || conf.prefix == "" {
		return path
	}
	if strings.HasPrefix(path, conf.prefix) {
		return path[len(conf.prefix):]
	}
	return path
}

func (conf *DConf3) sync() error {
	resp, err := conf.kv.Get(context.Background(), conf.prefix, clientv3.WithPrefix())
	if err != nil {
		return err
	}
	for _, kv := range resp.Kvs {
		realKey := conf.keyname(string(kv.Key))
		if realKey == "" {
			continue
		}
		conf.data.Store(realKey, string(kv.Value))
	}
	return nil
}

func (conf *DConf3) watch() error {
	ch := conf.watcher.Watch(conf.ctx, conf.prefix, clientv3.WithPrefix(), clientv3.WithPrevKV())
	for resp := range ch {
		for _, event := range resp.Events {
			switch event.Type {
			case mvccpb.PUT:
				realKey := conf.keyname(string(event.Kv.Key))
				if realKey == "" {
					break
				}
				conf.data.Store(realKey, string(event.Kv.Value))
			case mvccpb.DELETE:
				realKey := conf.keyname(string(event.PrevKv.Key))
				if realKey == "" {
					break
				}
				conf.data.Delete(realKey)
			}
		}
	}
	return nil
}

// Set stores an entry into etcd
func (conf *DConf3) Set(key string, value string, preExist ...bool) error {
	key = conf.fullpath(key)
	if len(preExist) > 0 && preExist[0] {
		resp, err := conf.kv.Txn(context.Background()).If(clientv3util.KeyExists(key)).Then(clientv3.OpPut(key, value)).Commit()
		if err != nil {
			return err
		}
		if !resp.Succeeded {
			return errors.New("transition failed")
		}
		return nil
	}
	_, err := conf.kv.Put(context.Background(), key, value)
	return err
}

// Get gets an entry from memory
func (conf *DConf3) Get(key string) (string, error) {
	value, ok := conf.data.Load(key)
	if !ok {
		return "", Error{Code: ErrorKeyNotFound, Message: fmt.Sprintf("key %s not found", key)}
	}

	return value.(string), nil
}

// Keys loads all keys from data
func (conf *DConf3) Keys(prefix string) []string {
	keys := make([]string, 0, 32)
	conf.data.Range(func(key, value interface{}) bool {
		s := key.(string)
		if strings.HasPrefix(s, prefix) {
			keys = append(keys, key.(string))
		}
		return true
	})
	return keys
}

// Del deletes an entry in etcd
func (conf *DConf3) Del(key string) error {
	_, err := conf.kv.Delete(context.Background(), conf.fullpath(key))
	return err
}

// Data loads all keys from data
func (conf *DConf3) Data(prefix string) map[string]string {
	data := make(map[string]string)
	conf.data.Range(func(key, value interface{}) bool {
		if prefix == "" || strings.HasPrefix(key.(string), prefix) {
			data[key.(string)] = value.(string)
		}
		return true
	})
	return data
}
