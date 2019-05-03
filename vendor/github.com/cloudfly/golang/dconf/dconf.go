// dynamic etc management based on etcd service

// each caller should provide
// 1. etcd addr, prefix
// 2. same constructor to unmarshal data from etcd
package dconf

import (
	"context"
	"fmt"
	"path"
	"strings"
	"sync"

	etcd "github.com/coreos/etcd/client"
)

// ErrorCode
const (
	ErrorKeyNotFound = 100
)

// Error represents the error returned to client side
type Error struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

func (e Error) Error() string {
	return fmt.Sprintf("%v: %v", e.Code, e.Message)
}

// IsKeyNotFound check the error
func IsKeyNotFound(err error) bool {
	if cErr, ok := err.(Error); ok {
		return cErr.Code == ErrorKeyNotFound
	}
	return false
}

// DConfv2 synchronizes etcd with in-memory data structures
type DConfv2 struct {
	ctx context.Context
	// data stores data synchronized with etcd
	// key format: `/prefix/key` is a leaf node of etcd
	// value is a struct instance pointer synchronized from etcd
	data sync.Map

	prefix      string // etcd watcher prefix
	watcher     etcd.Watcher
	kv          etcd.KeysAPI
	latestIndex uint64
}

// New inits a DConfv2 instance and reads data from etcd
func New(ctx context.Context, addrs []string, prefix string) (*DConfv2, error) {
	if prefix == "" {
		prefix = "/dconf"
	}
	if prefix[0] != '/' {
		prefix = "/" + prefix
	}
	if prefix[len(prefix)-1] != '/' {
		prefix = prefix + "/"
	}
	conf := &DConfv2{
		ctx:    ctx,
		prefix: prefix,
	}

	c, err := etcd.New(etcd.Config{Endpoints: addrs})
	if err != nil {
		return nil, err
	}
	conf.kv = etcd.NewKeysAPI(c)
	conf.watcher = conf.kv.Watcher(prefix, &etcd.WatcherOptions{Recursive: true})

	// initial sync
	if err := conf.sync(); err != nil {
		return nil, err
	}

	go conf.watch()

	return conf, nil
}

func (conf *DConfv2) sync() error {
	resp, err := conf.kv.Get(context.Background(), conf.prefix, &etcd.GetOptions{Recursive: true})
	if err != nil {
		// if key not found
		if etcd.IsKeyNotFound(err) {
			_, err := conf.kv.Set(context.Background(), conf.prefix, "", &etcd.SetOptions{Dir: true})
			if err != nil {
				return err
			}
			return nil
		}
		return err
	}

	if !resp.Node.Dir {
		return fmt.Errorf("etcd node %s is not a dir", conf.prefix)
	}
	conf.readNode(resp.Node)
	return nil
}

func (conf *DConfv2) fullpath(key string) string {
	return path.Join(conf.prefix, key)
}

func (conf *DConfv2) keyname(path string) string {
	if path == "" || conf.prefix == "" {
		return path
	}
	if strings.HasPrefix(path, conf.prefix) {
		return path[len(conf.prefix):]
	}
	return path
}

func (conf *DConfv2) watch() error {

	for {
		if conf.ctx.Err() != nil { // context canceled
			break
		}
		resp, err := conf.watcher.Next(conf.ctx)
		if err != nil {
			continue
		}

		// update index
		if resp.Node.ModifiedIndex < conf.latestIndex {
			continue
		}

		conf.latestIndex = resp.Node.ModifiedIndex

		realKey := conf.keyname(resp.Node.Key)

		switch resp.Action {
		case "delete", "compareAndDelete":
			conf.data.Delete(realKey)
		case "set", "update", "create", "compareAndSwap":
			conf.data.Store(realKey, resp.Node.Value)
		}
	}
	return nil
}

// Set stores an entry into etcd
func (conf *DConfv2) Set(key string, value string, preExist ...bool) error {
	setPreExistOption := etcd.PrevIgnore
	if len(preExist) > 0 {
		if preExist[0] {
			setPreExistOption = etcd.PrevExist
		} else {
			setPreExistOption = etcd.PrevNoExist
		}
	}
	_, err := conf.kv.Set(context.Background(), conf.fullpath(key), value, &etcd.SetOptions{PrevExist: setPreExistOption})
	return err
}

// Get gets an entry from memory
func (conf *DConfv2) Get(key string) (string, error) {
	value, ok := conf.data.Load(key)
	if !ok {
		return "", Error{Code: ErrorKeyNotFound, Message: fmt.Sprintf("key %s not found", key)}
	}

	return value.(string), nil
}

// Keys loads all keys from data
func (conf *DConfv2) Keys(prefix string) []string {
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
func (conf *DConfv2) Del(key string) error {
	_, err := conf.kv.Delete(context.Background(), conf.fullpath(key), &etcd.DeleteOptions{Dir: false})
	return err
}

// Data loads all keys from data
func (conf *DConfv2) Data(prefix string) map[string]string {
	data := make(map[string]string)
	conf.data.Range(func(key, value interface{}) bool {
		if prefix == "" || strings.HasPrefix(key.(string), prefix) {
			data[key.(string)] = value.(string)
		}
		return true
	})
	return data
}

func (conf *DConfv2) readNode(node *etcd.Node) {
	if node.Dir {
		for _, child := range node.Nodes {
			conf.readNode(child)
		}
	} else {
		realKey := conf.keyname(node.Key)
		if realKey == "" {
			return
		}
		conf.data.Store(realKey, node.Value)
		if node.ModifiedIndex > conf.latestIndex {
			conf.latestIndex = node.ModifiedIndex
		}
	}
}

// DConf 接口
type DConf interface {
	Get(key string) (string, error)
	Set(key, value string, preExist ...bool) error
	Del(key string) error
	Keys(prefix string) []string
	Data(prefix string) map[string]string
}
