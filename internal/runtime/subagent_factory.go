package runtime

import (
	"sync"

	"neo-code/internal/subagent"
)

var serviceSubAgentFactory sync.Map

// defaultSubAgentFactory 返回默认的子代理工厂实例。
func defaultSubAgentFactory() subagent.Factory {
	return subagent.NewWorkerFactory(nil)
}

// SetSubAgentFactory 设置子代理运行时工厂；传入 nil 时回退到默认工厂。
func (s *Service) SetSubAgentFactory(factory subagent.Factory) {
	if s == nil {
		return
	}
	if factory == nil {
		serviceSubAgentFactory.Store(s, defaultSubAgentFactory())
		return
	}
	serviceSubAgentFactory.Store(s, factory)
}

// SubAgentFactory 返回当前 runtime 持有的子代理运行时工厂。
func (s *Service) SubAgentFactory() subagent.Factory {
	if s == nil {
		return defaultSubAgentFactory()
	}
	if factory, ok := serviceSubAgentFactory.Load(s); ok {
		if typed, valid := factory.(subagent.Factory); valid && typed != nil {
			return typed
		}
	}
	defaultFactory := defaultSubAgentFactory()
	serviceSubAgentFactory.Store(s, defaultFactory)
	return defaultFactory
}
