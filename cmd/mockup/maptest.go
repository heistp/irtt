package main

import (
	"fmt"
	"math/rand"
	"sync"
	"time"
)

const purgeMax = 2

const purgeEvery = 1

const spawnUpToNS = 1000000

const accessUpToNTimes = 1000

const waitUpToNS = 10000

const printEvery = 500 * time.Millisecond

var expireTime = 500 * time.Millisecond

var mgr = &sessMan{make(map[int]*sess), sync.RWMutex{}, 0, 0, 0, time.Duration(0)}

type sess struct {
	atime time.Time
}

type sessMan struct {
	smap      map[int]*sess
	mtx       sync.RWMutex
	purgeTick int
	removed   int
	purges    int
	purgeDur  time.Duration
}

func (sm *sessMan) insert(k int, s *sess) {
	sm.mtx.Lock()
	defer sm.mtx.Unlock()
	sm.smap[k] = s
	sm.purgeTick++
	if sm.purgeTick >= purgeEvery {
		t := time.Now()
		i := 0
		for k, v := range sm.smap {
			if t.Sub(v.atime) > expireTime {
				delete(sm.smap, k)
				sm.removed++
			}
			i++
			if i >= purgeMax {
				break
			}
		}
		sm.purgeDur += time.Since(t)
		sm.purgeTick = 0
		sm.purges++
	}
}

func (sm *sessMan) get(k int) *sess {
	sm.mtx.Lock()
	defer sm.mtx.Unlock()
	s := sm.smap[k]
	s.atime = time.Now()
	return s
}

func (sm *sessMan) stats() (int, int, int, time.Duration) {
	sm.mtx.RLock()
	defer sm.mtx.RUnlock()
	return len(sm.smap), sm.removed, sm.purges, sm.purgeDur
}

func client() {
	k := rand.Int()
	mgr.insert(k, &sess{time.Now()})
	for i := 0; i < int(rand.Uint32())%accessUpToNTimes; i++ {
		time.Sleep(time.Duration(rand.Uint64() % waitUpToNS))
		mgr.get(k)
	}
}

func mapTest() {
	// start N goroutines that insert, get for a while then stop
	rand.Seed(time.Now().UnixNano())

	// start one goroutine that spawns goroutines to simulate clients
	go func() {
		for {
			go client()
			time.Sleep(time.Duration(rand.Uint64() % spawnUpToNS))
		}
	}()

	// start one goroutine to print the length indefinitely
	go func() {
		for {
			len, removed, purges, purgeDur := mgr.stats()
			fmt.Printf("len = %d removed=%d purge time=%s\n", len, removed,
				time.Duration(float64(purgeDur)/float64(time.Duration(purges))))
			time.Sleep(printEvery)
		}
	}()

	time.Sleep(1 * time.Hour)
}
