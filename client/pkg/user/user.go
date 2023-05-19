package user

import (
	"sync"
	"time"
)

type State int

const (
	StateNone State = iota
	StateAwaitingEmail
	StateAwaitingVerification
	StateAwaitingPercent
	StateAwaitingWaitTime
)

type UserManager struct {
	users map[int64]*User
	mu    sync.Mutex
}

type User struct {
	mu              sync.Mutex
	FirstChatID     int64
	SecondChatID    int64
	Email           string
	State           State
	ChangePercent24 *ChangePercent24
	PumpSettings    *PumpSettings
}

type ChangePercent24 struct {
	percent float64
	mu      sync.Mutex
}

type PumpSettings struct {
	mux         sync.Mutex
	waitTime    time.Duration
	pumpPercent float64
}

func NewUserManager() *UserManager {
	return &UserManager{
		users: make(map[int64]*User),
	}
}

func NewUser() *User {
	return &User{
		ChangePercent24: &ChangePercent24{},
		PumpSettings:    &PumpSettings{},
	}
}

func (cp24 *ChangePercent24) SetPercent(p float64) {
	cp24.mu.Lock()
	defer cp24.mu.Unlock()
	cp24.percent = p
}

func (cp24 *ChangePercent24) GetPercent() float64 {
	cp24.mu.Lock()
	defer cp24.mu.Unlock()
	return cp24.percent
}

func (p *PumpSettings) SetWaitTime(waitTime time.Duration) {
	p.mux.Lock()
	defer p.mux.Unlock()
	p.waitTime = waitTime
}

func (p *PumpSettings) GetWaitTime() time.Duration {
	p.mux.Lock()
	defer p.mux.Unlock()
	return p.waitTime
}

func (p *PumpSettings) SetPumpPercent(pumpPercent float64) {
	p.mux.Lock()
	defer p.mux.Unlock()
	p.pumpPercent = pumpPercent
}

func (p *PumpSettings) GetPumpPercent() float64 {
	p.mux.Lock()
	defer p.mux.Unlock()
	return p.pumpPercent
}

func (u *User) SetState(state State) {
	u.mu.Lock()
	defer u.mu.Unlock()
	u.State = state
}

func (u *User) GetState() State {
	u.mu.Lock()
	defer u.mu.Unlock()
	return u.State
}

func (u *User) SetEmail(email string) {
	u.mu.Lock()
	defer u.mu.Unlock()
	u.Email = email
}

func (u *User) GetEmail() string {
	u.mu.Lock()
	defer u.mu.Unlock()
	return u.Email
}

func (u *User) SetFirstChatID(id int64) {
	u.mu.Lock()
	defer u.mu.Unlock()
	u.FirstChatID = id
}

func (u *User) SetSecondChatID(id int64) {
	u.mu.Lock()
	defer u.mu.Unlock()
	u.SecondChatID = id
}

func (u *User) GetFirstChatID() int64 {
	u.mu.Lock()
	defer u.mu.Unlock()
	return u.FirstChatID
}

func (u *User) GetSecondChatID() int64 {
	u.mu.Lock()
	defer u.mu.Unlock()
	return u.SecondChatID
}

func (m *UserManager) GetUser(id int64) (*User, bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	user, ok := m.users[id]
	return user, ok
}

func (m *UserManager) AddUser(id int64, user *User) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.users[id] = user
}
