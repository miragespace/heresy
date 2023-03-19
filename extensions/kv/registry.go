package kv

var kvBacking = []matched{}

type matched struct {
	matcher     Matcher
	constructor KVConstructor
}

type KVConstructor func(uri string) (KV, error)

type Matcher func(uri string) bool

func Register(constructor KVConstructor, matcher Matcher) {
	kvBacking = append(kvBacking, matched{
		constructor: constructor,
		matcher:     matcher,
	})
}
