package tracker

import (
	"log"
	"sync"
	"time"
)

type SymbolChange struct {
	Symbol            string
	PriceChange       string
	PriceChangePct    float64
	AddedAt           time.Time
	LastMessageSentAt time.Time
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
	log.Printf("Added symbol to tracked list: %s", symbolChange.Symbol)
}

func (t *Tracker) RemoveTrackedSymbol(symbol string) {
	t.mu.Lock()
	defer t.mu.Unlock()
	delete(t.trackedSymbols, symbol)
	log.Printf("Removed symbol from tracked list: %s", symbol)
}

func (t *Tracker) GetTrackedSymbols() map[string]SymbolChange {
	t.mu.RLock()
	defer t.mu.RUnlock()
	copiedSymbols := make(map[string]SymbolChange)
	for k, v := range t.trackedSymbols {
		copiedSymbols[k] = v
	}
	return copiedSymbols
}

func (t *Tracker) MarkMessageSent(symbol string) {
	t.mu.Lock()
	defer t.mu.Unlock()

	if trackedSymbol, ok := t.trackedSymbols[symbol]; ok {
		trackedSymbol.LastMessageSentAt = time.Now()
		t.trackedSymbols[symbol] = trackedSymbol
	}
}
