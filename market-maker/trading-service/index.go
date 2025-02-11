package tsclient

import "sync"

var (
	indexStore = make(map[string]int32)
	indexMutex sync.Mutex
)

func SaveIndex(instrument string, index int32) {
	indexMutex.Lock()
	indexStore[instrument] = index
	indexMutex.Unlock()
}

func GetIndex(instrument string) int32 {
	indexMutex.Lock()
	defer indexMutex.Unlock()

	index, exist := indexStore[instrument]
	if !exist {
		return 0
	}
	return index
}

func ClearIndexStore() {
	indexMutex.Lock()
	defer indexMutex.Unlock()

	if len(indexStore) > 0 {
		for key := range indexStore {
			delete(indexStore, key)
		}
	}
	return
}
