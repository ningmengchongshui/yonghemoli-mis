package session

import (
	"crypto/rand"
	"encoding/hex"
	"sync"
	"time"
)

// Session 会话数据
type Session struct {
	AdminID     uint   // Auth 用户 ID（兼容既有业务操作者字段）
	Username    string // 显示名
	Email       string
	Avatar      string
	Roles       []string
	Permissions []string
	CreatedAt   time.Time
	ExpiresAt   time.Time
}

var (
	store = make(map[string]*Session)
	mu    sync.RWMutex
	ttl   = 24 * time.Hour
)

// Create 创建新会话，返回 sessionID
func Create(adminID uint, username, email, avatar string, roles, permissions []string) string {
	sid := generateID()
	mu.Lock()
	defer mu.Unlock()
	store[sid] = &Session{
		AdminID:     adminID,
		Username:    username,
		Email:       email,
		Avatar:      avatar,
		Roles:       roles,
		Permissions: permissions,
		CreatedAt:   time.Now(),
		ExpiresAt:   time.Now().Add(ttl),
	}
	return sid
}

// Get 获取会话，同时检查过期
func Get(sid string) *Session {
	mu.RLock()
	defer mu.RUnlock()
	s, ok := store[sid]
	if !ok {
		return nil
	}
	if time.Now().After(s.ExpiresAt) {
		go Destroy(sid)
		return nil
	}
	return s
}

// Destroy 销毁会话
func Destroy(sid string) {
	mu.Lock()
	defer mu.Unlock()
	delete(store, sid)
}

// generateID 生成随机 session ID
func generateID() string {
	b := make([]byte, 32)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}
