package services

import "log"

// SessionUpdateProcessor 会话更新批量处理器
type SessionUpdateProcessor struct {
	cache *SessionCache
}

// NewSessionUpdateProcessor 创建会话更新处理器
func NewSessionUpdateProcessor(cache *SessionCache) *SessionUpdateProcessor {
	return &SessionUpdateProcessor{cache: cache}
}

// Process 批量处理会话更新
func (p *SessionUpdateProcessor) Process(batch []sessionUpdateRequest) error {
	for _, req := range batch {
		if err := p.cache.UpdateSessionSuccess(req.platform, req.sessionID); err != nil {
			log.Printf("[WARN] 异步更新会话时间失败: %v\n", err)
		}
	}
	return nil
}

// ProcessSingle 处理单条会话更新
func (p *SessionUpdateProcessor) ProcessSingle(item sessionUpdateRequest) error {
	return p.cache.UpdateSessionSuccess(item.platform, item.sessionID)
}
