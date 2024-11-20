package errors

import (
	"crowdfunding-backend/internal/common"
)

// RecoveryStrategy 错误恢复策略接口
type RecoveryStrategy interface {
	CanRecover(err error) bool
	Recover(err error) error
}

// AutoRecovery 自动恢复处理器
type AutoRecovery struct {
	strategies []RecoveryStrategy
}

// NewAutoRecovery 创建自动恢复处理器
func NewAutoRecovery() *AutoRecovery {
	return &AutoRecovery{
		strategies: make([]RecoveryStrategy, 0),
	}
}

// AddStrategy 添加恢复策略
func (ar *AutoRecovery) AddStrategy(strategy RecoveryStrategy) {
	ar.strategies = append(ar.strategies, strategy)
}

// TryRecover 尝试恢复错误
func (ar *AutoRecovery) TryRecover(err error) error {
	for _, strategy := range ar.strategies {
		if strategy.CanRecover(err) {
			return strategy.Recover(err)
		}
	}
	return err
}

// DatabaseRecoveryStrategy 数据库错误恢复策略
type DatabaseRecoveryStrategy struct {
	maxRetries int
}

func (s *DatabaseRecoveryStrategy) CanRecover(err error) bool {
	// 判断是否是数据库连接错误
	return common.IsTemporary(err)
}

func (s *DatabaseRecoveryStrategy) Recover(err error) error {
	// 实现数据库重连逻辑
	return common.WithRetry(func() error {
		// 重试操作
		return nil
	}, s.maxRetries)
}
