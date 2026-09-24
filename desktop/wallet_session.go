package desktop

import (
	"errors"
	"strings"
	"sync"
	"time"

	"github.com/Sheff1981/valdr-core/wallet"
)

const (
	DefaultWalletAutoLock = 15 * time.Minute
	MinWalletAutoLock     = time.Minute
	MaxWalletAutoLock     = 24 * time.Hour
)

var (
	ErrWalletLocked       = errors.New("wallet is locked")
	ErrWalletAutoLock     = errors.New("invalid wallet auto-lock timeout")
	ErrWalletSessionStore = errors.New("wallet session store is required")
)

type walletSession struct {
	passphrase   []byte
	lastActivity time.Time
}

type WalletSessionManager struct {
	mu       sync.Mutex
	store    *wallet.Store
	timeout  time.Duration
	now      func() time.Time
	sessions map[string]*walletSession
}

func NewWalletSessionManager(
	store *wallet.Store,
	timeout time.Duration,
) (*WalletSessionManager, error) {
	return newWalletSessionManager(store, timeout, time.Now)
}

func newWalletSessionManager(
	store *wallet.Store,
	timeout time.Duration,
	now func() time.Time,
) (*WalletSessionManager, error) {
	if store == nil {
		return nil, ErrWalletSessionStore
	}
	if err := validateWalletAutoLock(timeout); err != nil {
		return nil, err
	}
	if now == nil {
		now = time.Now
	}
	return &WalletSessionManager{
		store:    store,
		timeout:  timeout,
		now:      now,
		sessions: make(map[string]*walletSession),
	}, nil
}

func (m *WalletSessionManager) Unlock(
	selector string,
	passphrase []byte,
) (wallet.Metadata, error) {
	secret := append([]byte(nil), passphrase...)
	unlocked, err := m.store.Unlock(strings.TrimSpace(selector), secret)
	if err != nil {
		clearBytes(secret)
		return wallet.Metadata{}, err
	}
	meta := unlocked.Metadata()
	unlocked.PrivateKey = ""

	m.mu.Lock()
	defer m.mu.Unlock()
	if previous := m.sessions[meta.Address]; previous != nil {
		clearBytes(previous.passphrase)
	}
	m.sessions[meta.Address] = &walletSession{
		passphrase:   secret,
		lastActivity: m.now(),
	}
	return meta, nil
}

func (m *WalletSessionManager) Lock(address string) {
	address = strings.TrimSpace(address)
	if address == "" {
		return
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	m.lockAddressLocked(address)
}

func (m *WalletSessionManager) LockAll() {
	m.mu.Lock()
	defer m.mu.Unlock()
	for address := range m.sessions {
		m.lockAddressLocked(address)
	}
}

func (m *WalletSessionManager) IsUnlocked(address string) bool {
	address = strings.TrimSpace(address)
	if address == "" {
		return false
	}
	m.mu.Lock()
	defer m.mu.Unlock()

	session := m.sessions[address]
	if session == nil {
		return false
	}
	if m.expiredLocked(session) {
		m.lockAddressLocked(address)
		return false
	}
	return true
}

func (m *WalletSessionManager) Passphrase(address string) ([]byte, error) {
	address = strings.TrimSpace(address)
	if address == "" {
		return nil, ErrWalletLocked
	}
	m.mu.Lock()
	defer m.mu.Unlock()

	session := m.sessions[address]
	if session == nil {
		return nil, ErrWalletLocked
	}
	if m.expiredLocked(session) {
		m.lockAddressLocked(address)
		return nil, ErrWalletLocked
	}
	session.lastActivity = m.now()
	return append([]byte(nil), session.passphrase...), nil
}

func (m *WalletSessionManager) Timeout() time.Duration {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.timeout
}

func (m *WalletSessionManager) SetTimeout(timeout time.Duration) error {
	if err := validateWalletAutoLock(timeout); err != nil {
		return err
	}

	m.mu.Lock()
	defer m.mu.Unlock()
	m.timeout = timeout
	for address, session := range m.sessions {
		if m.expiredLocked(session) {
			m.lockAddressLocked(address)
		}
	}
	return nil
}

func (m *WalletSessionManager) expiredLocked(session *walletSession) bool {
	if session == nil {
		return true
	}
	return m.now().Sub(session.lastActivity) >= m.timeout
}

func (m *WalletSessionManager) lockAddressLocked(address string) {
	session := m.sessions[address]
	if session == nil {
		return
	}
	clearBytes(session.passphrase)
	delete(m.sessions, address)
}

func validateWalletAutoLock(timeout time.Duration) error {
	if timeout < MinWalletAutoLock || timeout > MaxWalletAutoLock {
		return ErrWalletAutoLock
	}
	return nil
}
