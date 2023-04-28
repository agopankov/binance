package tracker

import (
	"sync"
	"time"
)

type SymbolChange struct {
	Symbol         string
	PriceChange    string
	PriceChangePct float64
	AddedAt        time.Time
}

type Tracker struct {
	mu             sync.RWMutex
	trackedSymbols map[string]SymbolChange
}

func NewTracker() *Tracker {
	return &Tracker{
		trackedSymbols: make(map[string]SymbolChange),
	}
}

func (t *Tracker) IsTracked(symbol string) bool {
	t.mu.RLock()
	defer t.mu.RUnlock()
	_, ok := t.trackedSymbols[symbol]
	return ok
}

func (t *Tracker) UpdateTrackedSymbol(symbolChange SymbolChange) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.trackedSymbols[symbolChange.Symbol] = symbolChange
}

func (t *Tracker) RemoveTrackedSymbol(symbol string) {
	t.mu.Lock()
	defer t.mu.Unlock()
	delete(t.trackedSymbols, symbol)
}

func (t *Tracker) GetTrackedSymbols() map[string]SymbolChange {
	t.mu.RLock()
	defer t.mu.RUnlock()
	// Создание копии карты перед возвратом, чтобы избежать конкурирующих изменений.
	copiedSymbols := make(map[string]SymbolChange)
	for k, v := range t.trackedSymbols {
		copiedSymbols[k] = v
	}
	return copiedSymbols
}
