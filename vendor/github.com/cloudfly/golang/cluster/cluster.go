package cluster

import (
	"context"
	"encoding/json"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/pkg/errors"
	"go.etcd.io/etcd/clientv3"
	"go.etcd.io/etcd/mvcc/mvccpb"
)

// global variables
var (
	ZeroNode = Node{}
)

// Cluster 代表一个集群
type Cluster struct {
	rwmu      sync.RWMutex
	ctx       context.Context
	cancel    func()
	endpoints []string
	prefix    string
	client    *clientv3.Client
	kv        clientv3.KV
	nodes     Nodes
	chans     []chan Nodes
}

// New 创建一个新集群，并将自己加入到节点中
func New(ctx context.Context, endpoints []string, prefix string, nodes Nodes) (*Cluster, error) {
	client, err := clientv3.New(clientv3.Config{
		Endpoints:   endpoints,
		DialTimeout: time.Second * 3,
	})
	if err != nil {
		return nil, err
	}

	ctx, cancel := context.WithCancel(ctx)

	c := &Cluster{
		ctx:       ctx,
		cancel:    cancel,
		endpoints: endpoints,
		prefix:    prefix,
		client:    client,
		kv:        clientv3.NewKV(client),
	}

	if err := c.init(nodes); err != nil {
		return nil, err
	}
	go c.activate()
	return c, nil
}

func (c *Cluster) init(nodes Nodes) error {
START:
	resp, err := c.kv.Get(context.Background(), c.prefix, clientv3.WithPrefix())
	if err != nil {
		return err
	}
	if len(resp.Kvs) == 0 { // set the nodes into etcd
		if len(nodes) == 0 {
			return errors.New("no nodes in cluster")
		}
		for _, node := range nodes {
			content, err := json.Marshal(node)
			if err != nil {
				return err
			}
			if _, err := c.kv.Put(c.ctx, genKey(c.prefix, node.Name), string(content)); err != nil {
				return err
			}
		}
		goto START
	}

	nodes2 := make(Nodes, 0, 32)
	for _, kv := range resp.Kvs {
		var node Node
		if err := json.Unmarshal(kv.Value, &node); err != nil {
			return err
		}
		nodes2 = append(nodes2, node)
	}
	c.nodes = nodes2
	return nil
}

func (c *Cluster) activate() {
	client := c.client
	watcher := clientv3.NewWatcher(client)
	ch := watcher.Watch(c.ctx, c.prefix, clientv3.WithPrefix(), clientv3.WithPrevKV())
	timer := time.NewTimer(time.Second * 3)
	defer timer.Stop()

	willBroadcast := false
	for {
		select {
		case resp := <-ch:
			for _, event := range resp.Events {
				var node Node
				switch event.Type {
				case mvccpb.PUT:
					if err := json.Unmarshal(event.Kv.Value, &node); err != nil {
						break // break the switch
					}
					if b := c.updateNode(node); b {
						willBroadcast = true
						timer.Reset(time.Second * 3)
					}
				case mvccpb.DELETE:
					if err := json.Unmarshal(event.PrevKv.Value, &node); err != nil {
						break // break the switch
					}
					c.removeNode(node)
					willBroadcast = true
					timer.Reset(time.Second * 3)
				}
			}
		case <-timer.C:
			if willBroadcast {
				nodes := c.Nodes()
				for _, ch := range c.chans {
					ch := ch
					select {
					case ch <- nodes:
					}
				}
				willBroadcast = false
			}
		}
	}
}

// updateNode 更新一个集群节点信息，如果发生了节点变化返回 true，如果此次调用没有修改集群节点信息返回 false
func (c *Cluster) updateNode(node Node) bool {
	c.rwmu.Lock()
	defer c.rwmu.Unlock()
	defer func() {
		sort.Sort(c.nodes)
	}()

	for i, n := range c.nodes {
		if n.Name == node.Name {
			c.nodes[i] = node
			return n != node
		}
	}
	c.nodes = append(c.nodes, node)
	return true
}

func (c *Cluster) removeNode(node Node) {
	c.rwmu.Lock()
	defer c.rwmu.Unlock()
	defer func() {
		sort.Sort(c.nodes)
	}()

	for i, m := range c.nodes {
		if m.Name == node.Name {
			c.nodes = append(c.nodes[:i], c.nodes[i+1:]...)
			return
		}
	}
}

// Distribute 将一个任务 id 分配给其中一个 Node 节点，并返回这个 Node
func (c *Cluster) Distribute(id int64) Node {
	return c.nodes.Distribute(id)
}

// Nodes 获取集群节点列表
func (c *Cluster) Nodes() Nodes {
	c.rwmu.RLock()
	defer c.rwmu.RUnlock()
	nodes := make(Nodes, 0, len(c.nodes))
	for _, node := range c.nodes {
		node := node
		nodes = append(nodes, node)
	}
	return nodes
}

// NodesChan 返回一个 Nodes 类型 channel，用来实时获取 node 的变化情况
func (c *Cluster) NodesChan() <-chan Nodes {
	ch := make(chan Nodes, 8)
	go func() {
		ch <- c.Nodes()
	}()
	c.chans = append(c.chans, ch)
	return ch
}

// HasNode 判断一个 node 是否在集群中; 通常用于判断自身是否在集群中
func (c *Cluster) HasNode(name string) bool {
	c.rwmu.RLock()
	defer c.rwmu.RUnlock()
	for _, node := range c.nodes {
		if node.Name == name {
			return true
		}
	}
	return false
}

// AddNode 向集群中增加一个新节点
func (c *Cluster) AddNode(node Node) error {
	content, err := json.Marshal(node)
	if err != nil {
		return err
	}
	key := genKey(c.prefix, node.Name)
	req := clientv3.OpPut(key, string(content))
	condition := clientv3.Compare(clientv3.Version(key), "=", 0)
	resp, err := c.kv.Txn(c.ctx).If(condition).Then(req).Commit()
	if err != nil {
		return err
	}
	if !resp.Succeeded {
		return errors.New("node already exist")
	}
	return nil
}

// RemoveNode 从集群中删除一个节点
func (c *Cluster) RemoveNode(nodeName string) error {
	_, err := c.kv.Delete(c.ctx, genKey(c.prefix, nodeName))
	return err
}

// UpdateNode 更新一个节点信息
func (c *Cluster) UpdateNode(node Node) error {
	content, err := json.Marshal(node)
	if err != nil {
		return err
	}
	key := genKey(c.prefix, node.Name)
	req := clientv3.OpPut(key, string(content))
	condition := clientv3.Compare(clientv3.Version(key), ">", 0)
	resp, err := c.kv.Txn(c.ctx).If(condition).Then(req).Commit()
	if err != nil {
		return err
	}
	if !resp.Succeeded {
		return errors.New("node not exist")
	}
	return nil
}

// Destroy 删除 cluster 中所有 node，所有节点都会接收到这个删除事件
func (c *Cluster) Destroy() error {
	_, err := c.kv.Delete(c.ctx, c.prefix, clientv3.WithPrefix())
	if err != nil {
		return err
	}
	c.cancel()
	return c.client.Close()
}

// Node 代表一个集群节点
type Node struct {
	Name   string
	Weight int
}

// Nodes 代表集群的所有节点
type Nodes []Node

func (nodes Nodes) Len() int {
	return len(nodes)
}

func (nodes Nodes) Less(i, j int) bool {
	return strings.Compare(nodes[i].Name, nodes[j].Name) < 0
}

func (nodes Nodes) Swap(i, j int) {
	nodes[i], nodes[j] = nodes[j], nodes[i]
}

// Distribute 将一个任务 id 分配给其中一个 Node 节点，并返回这个 Node
func (nodes Nodes) Distribute(id int64) Node {
	return nodes[id%int64(len(nodes))]
}

func genKey(items ...string) string {
	builder := strings.Builder{}
	for _, item := range items {
		if item[0] != '/' {
			builder.WriteByte('/')
		}
		builder.WriteString(item)
	}
	return builder.String()
}
