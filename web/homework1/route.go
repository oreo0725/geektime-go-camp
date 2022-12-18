package web

import (
	"fmt"
	"regexp"
	"strings"
)

type router struct {
	// trees 是按照 HTTP 方法来组织的
	// 如 GET => *node
	trees map[string]*node
}

func newRouter() router {
	return router{
		trees: map[string]*node{},
	}
}

// addRoute 注册路由。
// method 是 HTTP 方法
// - 已经注册了的路由，无法被覆盖。例如 /user/home 注册两次，会冲突
// - path 必须以 / 开始并且结尾不能有 /，中间也不允许有连续的 /
// - 不能在同一个位置注册不同的参数路由，例如 /user/:id 和 /user/:name 冲突
// - 不能在同一个位置同时注册通配符路由和参数路由，例如 /user/:id 和 /user/* 冲突
// - 同名路径参数，在路由匹配的时候，值会被覆盖。例如 /user/:id/abc/:id，那么 /user/123/abc/456 最终 id = 456
func (r *router) addRoute(method string, path string, handler HandleFunc) {
	if path == "" {
		panic("web: path cannot be empty")
	}
	if path[0] != '/' {
		panic("web: path must start with /")
	}
	if path != "/" && path[len(path)-1] == '/' {
		panic("web: path must not end with /")
	}
	methodRoot, ok := r.trees[method]
	if !ok {
		methodRoot = &node{path: "/"}
		r.trees[method] = methodRoot
	}

	if path == "/" {
		if methodRoot.handler != nil {
			panic(fmt.Sprintf("web: the path [%s] are duplicated to assign handler", path))
		}
		methodRoot.handler = handler
		methodRoot.typ = nodeTypeStatic
		return
	}

	segs := strings.Split(path[1:], "/")
	// traverse the whole path, validate the segment then create child node
	var traverseNode = methodRoot
	for _, seg := range segs {
		if seg == "" {
			panic(fmt.Sprintf("web: not accept empty segment in path [%s]", path))
		}
		traverseNode = traverseNode.childOrCreate(seg)
	}
	if traverseNode.handler != nil {
		panic(fmt.Sprintf("web: the path [%s] are duplicated to assign handler", path))
	}
	traverseNode.handler = handler
}

// findRoute 查找对应的节点
// 注意，返回的 node 内部 HandleFunc 不为 nil 才算是注册了路由
func (r *router) findRoute(method string, path string) (*matchInfo, bool) {
	root, ok := r.trees[method]
	if !ok {
		return nil, false
	}
	if path == "/" {
		return &matchInfo{
			n: root,
		}, true
	}

	if root.children == nil {
		return nil, false
	}
	pathTrimed := strings.Trim(path, "/")
	var pathParams map[string]string
	segs := strings.Split(pathTrimed, "/")
	for _, seg := range segs {
		if child, ok := root.childOf(seg); ok {
			root = child
			if root.typ == nodeTypeParam {
				if pathParams == nil {
					pathParams = make(map[string]string)
				}
				pathParams[root.path[1:]] = seg
			}

		} else if root.starChild != nil {
			root = root.starChild
		} else {
			return nil, false
		}
	}
	return &matchInfo{
		n:          root,
		pathParams: pathParams,
	}, true
}

type nodeType int

const (
	// 静态路由
	nodeTypeStatic nodeType = iota
	// 正则路由
	nodeTypeReg
	// 路径参数路由
	nodeTypeParam
	// 通配符路由
	nodeTypeAny
)

// node 代表路由树的节点
// 路由树的匹配顺序是：
// 1. 静态完全匹配
// 2. 正则匹配，形式 :param_name(reg_expr)
// 3. 路径参数匹配：形式 :param_name
// 4. 通配符匹配：*
// 这是不回溯匹配
type node struct {
	typ nodeType

	path string
	// children 子节点
	// 子节点的 path => node
	children map[string]*node
	// handler 命中路由之后执行的逻辑
	handler HandleFunc

	// 通配符 * 表达的节点，任意匹配
	starChild *node

	paramChild *node
	// 正则路由和参数路由都会使用这个字段
	paramName string

	// 正则表达式
	regChild *node
	regExpr  *regexp.Regexp
}

// child 返回子节点
// 第一个返回值 *node 是命中的节点
// 第二个返回值 bool 代表是否命中
func (n *node) childOf(path string) (*node, bool) {
	findNonStaticMatch := func() (*node, bool) {
		if n.paramChild != nil {
			return n.paramChild, true
		}
		if n.starChild != nil {
			return n.starChild, true
		}
		return nil, false
	}
	if n.children == nil {
		return findNonStaticMatch()
	}
	child, ok := n.children[path]
	if !ok {
		return findNonStaticMatch()
	}
	return child, ok
}

// childOrCreate 查找子节点，
// 首先会判断 path 是不是通配符路径
// 其次判断 path 是不是参数路径，即以 : 开头的路径
// 最后会从 children 里面查找，
// 如果没有找到，那么会创建一个新的节点，并且保存在 node 里面
func (n *node) childOrCreate(path string) *node {
	if path[0] == ':' {
		if n.starChild != nil {
			panic(fmt.Sprintf("web: wildcard path is defined. only one of path parameter and wildcard can be defined [%s]", path))
		}
		//if n.paramChild != nil && n.paramChild.path != path {
		//	panic(fmt.Sprintf("web: duplicated define for path parameter [%s]", path))
		//}
		//n.paramChild = &node{path: path, typ: nodeTypeParam}
		if n.paramChild != nil {
			if n.paramChild.path != path {
				panic(fmt.Sprintf("web: duplicated define for path parameter [%s]", path))
			}
		} else {
			n.paramChild = &node{path: path, typ: nodeTypeParam}
		}
		return n.paramChild
	}
	if path == "*" {
		if n.paramChild != nil {
			panic(fmt.Sprintf("web: path parameter is defined. only one of path parameter and wildcard can be defined [%s]", path))
		}
		if n.starChild == nil {
			n.starChild = &node{path: "*", typ: nodeTypeAny}
		}
		return n.starChild
	}
	if n.children == nil {
		n.children = map[string]*node{}
	}
	child, ok := n.children[path]
	if !ok {
		child = &node{
			path: path,
			typ:  nodeTypeStatic,
		}
		n.children[path] = child
	}

	return child
}

type matchInfo struct {
	n          *node
	pathParams map[string]string
}

func (m *matchInfo) addValue(key string, value string) {
	if m.pathParams == nil {
		// 大多数情况，参数路径只会有一段
		m.pathParams = map[string]string{key: value}
	}
	m.pathParams[key] = value
}
