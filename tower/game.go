package tower

import (
	"log"
	"sync"
	"time"

	"github.com/solanashuffle/backend/utility"
)

type lockType struct {
	mu       *sync.Mutex
	lastUsed time.Time
}

var (
	gameLockMap   = make(map[string]*lockType)
	muGameLockMap = sync.RWMutex{}
)

func init() {
	go func() {
		for {
			log.Println("checking all locks")
			muGameLockMap.Lock()
			for id, lock := range gameLockMap {
				if time.Since(lock.lastUsed) < time.Minute*15 {
					delete(gameLockMap, id)
				}
			}
			muGameLockMap.Unlock()
			time.Sleep(time.Minute * 15)
		}
	}()
}

func lockGameID(id string) {
	muGameLockMap.RLock()
	lock, ok := gameLockMap[id]
	muGameLockMap.RUnlock()
	if !ok {
		lock = &lockType{
			mu: &sync.Mutex{},
		}
		muGameLockMap.Lock()
		gameLockMap[id] = lock
		muGameLockMap.Unlock()
	}
	lock.mu.Lock()

	lock.lastUsed = time.Now()
	muGameLockMap.Lock()
	gameLockMap[id] = lock
	muGameLockMap.Unlock()
}

func unlockGameID(id string) {
	muGameLockMap.RLock()
	lock, ok := gameLockMap[id]
	muGameLockMap.RUnlock()
	if !ok {
		return
	}

	if !utility.MutexLocked(lock.mu) {
		return
	}

	lock.mu.Unlock()
}
