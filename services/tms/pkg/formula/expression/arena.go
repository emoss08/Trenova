package expression

import (
	"fmt"
	"sync"
	"unsafe"
)

type Arena struct {
	block         []byte
	offset        int
	blockSize     int
	blocks        [][]byte
	floatPool     []float64
	stringPool    []string
	boolPool      []bool
	interfacePool []any
	strings       map[string]string
	allocations   int64
	bytesUsed     int64
	mu            sync.Mutex
}

type ArenaStats struct {
	BlocksAllocated int
	BytesAllocated  int64
	BytesUsed       int64
	Allocations     int64
	StringsInterned int
}

func NewArena(blockSize int) *Arena {
	if blockSize <= 0 {
		blockSize = 64 * 1024 // 64KB default
	}

	a := &Arena{
		blockSize:     blockSize,
		blocks:        make([][]byte, 0, 4),
		strings:       make(map[string]string),
		floatPool:     make([]float64, 0, 128),
		stringPool:    make([]string, 0, 64),
		boolPool:      make([]bool, 0, 32),
		interfacePool: make([]any, 0, 64),
	}

	a.allocateBlock()
	return a
}

func (a *Arena) allocateBlock() {
	block := make([]byte, a.blockSize)
	a.blocks = append(a.blocks, block)
	a.block = block
	a.offset = 0
}

func (a *Arena) Alloc(n int) unsafe.Pointer {
	a.mu.Lock()
	defer a.mu.Unlock()

	// ! Align to 8 bytes for better performance
	n = (n + 7) &^ 7

	if a.offset+n > len(a.block) {
		if n > a.blockSize {
			specialBlock := make([]byte, n)
			a.blocks = append(a.blocks, specialBlock)
			a.bytesUsed += int64(n)
			a.allocations++
			return unsafe.Pointer(&specialBlock[0])
		}

		a.allocateBlock()
	}

	ptr := unsafe.Pointer(&a.block[a.offset])
	a.offset += n
	a.bytesUsed += int64(n)
	a.allocations++

	return ptr
}

func (a *Arena) AllocFloat64(v float64) *float64 {
	a.mu.Lock()

	if len(a.floatPool) < cap(a.floatPool) {
		a.floatPool = append(a.floatPool, v)
		result := &a.floatPool[len(a.floatPool)-1]
		a.allocations++
		a.bytesUsed += 8
		a.mu.Unlock()
		return result
	}

	a.mu.Unlock()

	ptr := (*float64)(a.Alloc(int(unsafe.Sizeof(v))))
	*ptr = v
	return ptr
}

func (a *Arena) AllocString(s string) string {
	a.mu.Lock()
	defer a.mu.Unlock()

	if interned, ok := a.strings[s]; ok {
		return interned
	}

	n := len(s)
	if n == 0 {
		return ""
	}

	if len(a.stringPool) < cap(a.stringPool) {
		a.stringPool = append(a.stringPool, s)
		interned := a.stringPool[len(a.stringPool)-1]
		a.strings[s] = interned
		a.bytesUsed += int64(n)
		a.allocations++
		return interned
	}

	data := make([]byte, n)
	copy(data, s)
	result := string(data)
	a.strings[s] = result
	a.bytesUsed += int64(n)
	a.allocations++
	return result
}

func (a *Arena) AllocBool(v bool) *bool {
	a.mu.Lock()

	if len(a.boolPool) < cap(a.boolPool) {
		a.boolPool = append(a.boolPool, v)
		result := &a.boolPool[len(a.boolPool)-1]
		a.allocations++
		a.bytesUsed++
		a.mu.Unlock()
		return result
	}

	a.mu.Unlock()

	ptr := (*bool)(a.Alloc(int(unsafe.Sizeof(v))))
	*ptr = v
	return ptr
}

func (a *Arena) AllocInterface(v any) any {
	// ! Special handling for common types (don't lock yet to avoid deadlock)
	switch val := v.(type) {
	case float64:
		return a.AllocFloat64(val)
	case string:
		return a.AllocString(val)
	case bool:
		return a.AllocBool(val)
	case nil:
		return nil
	}

	a.mu.Lock()
	defer a.mu.Unlock()

	if len(a.interfacePool) < cap(a.interfacePool) {
		a.interfacePool = append(a.interfacePool, v)
		return a.interfacePool[len(a.interfacePool)-1]
	}

	// For other types, just return the value
	// (arena allocation for arbitrary types is complex)
	return v
}

func (a *Arena) AllocSlice(elemSize, length, capacity int) unsafe.Pointer {
	if capacity < length {
		capacity = length
	}

	arraySize := elemSize * capacity
	arrayPtr := a.Alloc(arraySize)
	slice := unsafe.Slice((*byte)(arrayPtr), capacity)

	return unsafe.Pointer(unsafe.SliceData(slice))
}

func (a *Arena) Reset() {
	a.mu.Lock()
	defer a.mu.Unlock()

	if len(a.blocks) > 0 {
		a.block = a.blocks[0]
		a.blocks = a.blocks[:1]
		a.offset = 0
	}

	a.floatPool = a.floatPool[:0]
	a.stringPool = a.stringPool[:0]
	a.boolPool = a.boolPool[:0]
	a.interfacePool = a.interfacePool[:0]

	for k := range a.strings {
		delete(a.strings, k)
	}

	a.bytesUsed = 0
	a.allocations = 0
}

func (a *Arena) Stats() ArenaStats {
	a.mu.Lock()
	defer a.mu.Unlock()

	totalBytes := int64(0)
	for _, block := range a.blocks {
		totalBytes += int64(len(block))
	}

	return ArenaStats{
		BlocksAllocated: len(a.blocks),
		BytesAllocated:  totalBytes,
		BytesUsed:       a.bytesUsed,
		Allocations:     a.allocations,
		StringsInterned: len(a.strings),
	}
}

type ArenaPool struct {
	pool      sync.Pool
	blockSize int
}

func NewArenaPool(blockSize int) *ArenaPool {
	return &ArenaPool{
		blockSize: blockSize,
		pool: sync.Pool{
			New: func() any {
				return NewArena(blockSize)
			},
		},
	}
}

func (p *ArenaPool) Get() *Arena {
	arena, _ := p.pool.Get().(*Arena)
	arena.Reset()
	return arena
}

func (p *ArenaPool) Put(arena *Arena) {
	arena.Reset()
	p.pool.Put(arena)
}

func (p *ArenaPool) WithArena(fn func(*Arena) error) error {
	arena := p.Get()
	defer p.Put(arena)
	return fn(arena)
}

var globalArenaPool = NewArenaPool(64 * 1024) // 64KB blocks

func GetArena() *Arena {
	return globalArenaPool.Get()
}

func PutArena(arena *Arena) {
	globalArenaPool.Put(arena)
}

type ArenaAllocator struct {
	arena *Arena
}

func NewArenaAllocator(arena *Arena) *ArenaAllocator {
	return &ArenaAllocator{arena: arena}
}

func (a *ArenaAllocator) AllocFloat64(v float64) any {
	return a.arena.AllocFloat64(v)
}

func (a *ArenaAllocator) AllocString(s string) any {
	return a.arena.AllocString(s)
}

func (a *ArenaAllocator) AllocBool(v bool) any {
	return a.arena.AllocBool(v)
}

func (a *ArenaAllocator) AllocArray(elements []any) any {
	// ! For now, just return the slice
	// TODO: Implement proper arena allocation for arrays
	return elements
}

func (a *Arena) String() string {
	stats := a.Stats()
	return fmt.Sprintf("Arena{blocks=%d, allocated=%d, used=%d, allocations=%d}",
		stats.BlocksAllocated, stats.BytesAllocated, stats.BytesUsed, stats.Allocations)
}
