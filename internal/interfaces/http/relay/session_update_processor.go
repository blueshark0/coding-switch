package relay

import (
	"log"

	sessioninfra "codeswitch/internal/sessions/infrastructure"
)

type sessionUpdateProcessor struct {
	cache *sessioninfra.Cache
}

func newSessionUpdateProcessor(cache *sessioninfra.Cache) *sessionUpdateProcessor {
	return &sessionUpdateProcessor{cache: cache}
}

func (p *sessionUpdateProcessor) Process(batch []sessionUpdateRequest) error {
	for _, req := range batch {
		if err := p.cache.UpdateSessionSuccessGeneration(req.platform, req.sessionID, req.generation); err != nil {
			log.Printf("[WARN] 异步更新会话时间失败: %v\n", err)
		}
	}
	return nil
}

func (p *sessionUpdateProcessor) ProcessSingle(item sessionUpdateRequest) error {
	if err := p.cache.UpdateSessionSuccessGeneration(item.platform, item.sessionID, item.generation); err != nil {
		log.Printf("[WARN] 异步更新会话时间失败: %v\n", err)
		return err
	}
	return nil
}
