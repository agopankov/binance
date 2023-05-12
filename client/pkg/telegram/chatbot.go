package telegram

import (
	"sync"
	"time"
)

type State int

const (
	StateNone State = iota
	StateAwaitingPercent
	StateAwaitingWaitTime
)

type ChatState struct {
	FirstChatID  int64
	SecondChatID int64
	State        State
	mu           sync.Mutex
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

func (cs *ChatState) SetFirstChatID(id int64) {
	cs.mu.Lock()
	defer cs.mu.Unlock()
	cs.FirstChatID = id
}

func (cs *ChatState) SetSecondChatID(id int64) {
	cs.mu.Lock()
	defer cs.mu.Unlock()
	cs.SecondChatID = id
}

func (cs *ChatState) GetFirstChatID() int64 {
	cs.mu.Lock()
	defer cs.mu.Unlock()
	return cs.FirstChatID
}

func (cs *ChatState) GetSecondChatID() int64 {
	cs.mu.Lock()
	defer cs.mu.Unlock()
	return cs.SecondChatID
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

func (cs *ChatState) SetState(state State) {
	cs.mu.Lock()
	defer cs.mu.Unlock()
	cs.State = state
}

func (cs *ChatState) GetState() State {
	cs.mu.Lock()
	defer cs.mu.Unlock()
	return cs.State
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
