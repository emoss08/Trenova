package expression

import (
	"fmt"
	"reflect"
	"sync"
	"unsafe"
)

// Arena is a memory arena for efficient allocation during expression evaluation
type Arena struct {
	// Current block of memory
	block     []byte
	offset    int
	blockSize int

	// List of all allocated blocks
	blocks [][]byte

	// Object pool for common types
	floatPool     []float64
	stringPool    []string
	boolPool      []bool
	interfacePool []any

	// String interning map
	strings map[string]string

	// Metrics
	allocations int64
	bytesUsed   int64

	mu sync.Mutex
}

// ArenaStats contains arena memory statistics
type ArenaStats struct {
	BlocksAllocated int
	BytesAllocated  int64
	BytesUsed       int64
	Allocations     int64
	StringsInterned int
}

// NewArena creates a new memory arena with the specified block size
func NewArena(blockSize int) *Arena {
	if blockSize <= 0 {
		blockSize = 64 * 1024 // 64KB default
	}

	a := &Arena{
		blockSize: blockSize,
		blocks:    make([][]byte, 0, 4),
		strings:   make(map[string]string),

		// Pre-allocate pools
		floatPool:     make([]float64, 0, 128),
		stringPool:    make([]string, 0, 64),
		boolPool:      make([]bool, 0, 32),
		interfacePool: make([]any, 0, 64),
	}

	a.allocateBlock()
	return a
}

// allocateBlock allocates a new memory block
func (a *Arena) allocateBlock() {
	block := make([]byte, a.blockSize)
	a.blocks = append(a.blocks, block)
	a.block = block
	a.offset = 0
}

// Alloc allocates n bytes from the arena
func (a *Arena) Alloc(n int) unsafe.Pointer {
	a.mu.Lock()
	defer a.mu.Unlock()

	// Align to 8 bytes for better performance
	n = (n + 7) &^ 7

	// Check if we need a new block
	if a.offset+n > len(a.block) {
		// If requested size is larger than block size, allocate a special block
		if n > a.blockSize {
			specialBlock := make([]byte, n)
			a.blocks = append(a.blocks, specialBlock)
			a.bytesUsed += int64(n)
			a.allocations++
			return unsafe.Pointer(&specialBlock[0])
		}

		a.allocateBlock()
	}

	// Allocate from current block
	ptr := unsafe.Pointer(&a.block[a.offset])
	a.offset += n
	a.bytesUsed += int64(n)
	a.allocations++

	return ptr
}

// AllocFloat64 allocates a float64 from the arena
func (a *Arena) AllocFloat64(v float64) *float64 {
	a.mu.Lock()

	// Try to reuse from pool
	if len(a.floatPool) < cap(a.floatPool) {
		a.floatPool = append(a.floatPool, v)
		result := &a.floatPool[len(a.floatPool)-1]
		a.allocations++
		a.bytesUsed += 8
		a.mu.Unlock()
		return result
	}

	a.mu.Unlock()

	// Allocate new (Alloc will handle its own locking)
	ptr := (*float64)(a.Alloc(int(unsafe.Sizeof(v))))
	*ptr = v
	return ptr
}

// AllocString allocates a string from the arena
func (a *Arena) AllocString(s string) string {
	a.mu.Lock()
	defer a.mu.Unlock()

	// Check if already interned
	if interned, ok := a.strings[s]; ok {
		return interned
	}

	// Allocate string data
	n := len(s)
	if n == 0 {
		return ""
	}

	// Use string pool if possible
	if len(a.stringPool) < cap(a.stringPool) {
		a.stringPool = append(a.stringPool, s)
		interned := a.stringPool[len(a.stringPool)-1]
		a.strings[s] = interned
		a.bytesUsed += int64(n)
		a.allocations++
		return interned
	}

	// Allocate new string (don't use Alloc to avoid deadlock)
	data := make([]byte, n)
	copy(data, s)
	result := string(data)
	a.strings[s] = result
	a.bytesUsed += int64(n)
	a.allocations++
	return result
}

// AllocBool allocates a bool from the arena
func (a *Arena) AllocBool(v bool) *bool {
	a.mu.Lock()

	// Try to reuse from pool
	if len(a.boolPool) < cap(a.boolPool) {
		a.boolPool = append(a.boolPool, v)
		result := &a.boolPool[len(a.boolPool)-1]
		a.allocations++
		a.bytesUsed++
		a.mu.Unlock()
		return result
	}

	a.mu.Unlock()

	// Allocate new (Alloc will handle its own locking)
	ptr := (*bool)(a.Alloc(int(unsafe.Sizeof(v))))
	*ptr = v
	return ptr
}

// AllocInterface allocates an interface{} from the arena
func (a *Arena) AllocInterface(v any) any {
	// Special handling for common types (don't lock yet to avoid deadlock)
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

	// Use interface pool if possible
	if len(a.interfacePool) < cap(a.interfacePool) {
		a.interfacePool = append(a.interfacePool, v)
		return a.interfacePool[len(a.interfacePool)-1]
	}

	// For other types, just return the value
	// (arena allocation for arbitrary types is complex)
	return v
}

// AllocSlice allocates a slice with the given length and capacity
func (a *Arena) AllocSlice(elemSize, length, capacity int) unsafe.Pointer {
	if capacity < length {
		capacity = length
	}

	// Calculate sizes
	arraySize := elemSize * capacity
	sliceSize := int(unsafe.Sizeof(reflect.SliceHeader{}))
	totalSize := arraySize + sliceSize

	// Allocate both array and header in one go
	ptr := a.Alloc(totalSize)

	// Set up slice header
	slicePtr := ptr
	arrayPtr := unsafe.Pointer(uintptr(ptr) + uintptr(sliceSize))

	header := (*reflect.SliceHeader)(slicePtr)
	header.Data = uintptr(arrayPtr)
	header.Len = length
	header.Cap = capacity

	return slicePtr
}

// Reset clears the arena for reuse
func (a *Arena) Reset() {
	a.mu.Lock()
	defer a.mu.Unlock()

	// Keep only the first block
	if len(a.blocks) > 0 {
		a.block = a.blocks[0]
		a.blocks = a.blocks[:1]
		a.offset = 0
	}

	// Clear pools
	a.floatPool = a.floatPool[:0]
	a.stringPool = a.stringPool[:0]
	a.boolPool = a.boolPool[:0]
	a.interfacePool = a.interfacePool[:0]

	// Clear string intern map
	for k := range a.strings {
		delete(a.strings, k)
	}

	// Reset metrics
	a.bytesUsed = 0
	a.allocations = 0
}

// Stats returns arena statistics
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

// ArenaPool manages a pool of reusable arenas
type ArenaPool struct {
	pool      sync.Pool
	blockSize int
}

// NewArenaPool creates a new arena pool
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

// Get obtains an arena from the pool
func (p *ArenaPool) Get() *Arena {
	arena, _ := p.pool.Get().(*Arena)
	arena.Reset()
	return arena
}

// Put returns an arena to the pool
func (p *ArenaPool) Put(arena *Arena) {
	arena.Reset()
	p.pool.Put(arena)
}

// WithArena executes a function with a pooled arena
func (p *ArenaPool) WithArena(fn func(*Arena) error) error {
	arena := p.Get()
	defer p.Put(arena)
	return fn(arena)
}

// Global arena pool for expression evaluation
var globalArenaPool = NewArenaPool(64 * 1024) // 64KB blocks

// GetArena gets an arena from the global pool
func GetArena() *Arena {
	return globalArenaPool.Get()
}

// PutArena returns an arena to the global pool
func PutArena(arena *Arena) {
	globalArenaPool.Put(arena)
}

// ArenaAllocator provides an allocation interface using an arena
type ArenaAllocator struct {
	arena *Arena
}

// NewArenaAllocator creates a new arena allocator
func NewArenaAllocator(arena *Arena) *ArenaAllocator {
	return &ArenaAllocator{arena: arena}
}

// AllocFloat64 allocates a float64
func (a *ArenaAllocator) AllocFloat64(v float64) any {
	return a.arena.AllocFloat64(v)
}

// AllocString allocates a string
func (a *ArenaAllocator) AllocString(s string) any {
	return a.arena.AllocString(s)
}

// AllocBool allocates a bool
func (a *ArenaAllocator) AllocBool(v bool) any {
	return a.arena.AllocBool(v)
}

// AllocArray allocates an array
func (a *ArenaAllocator) AllocArray(elements []any) any {
	// For now, just return the slice
	// TODO: Implement proper arena allocation for arrays
	return elements
}

// String returns a debug string for the arena
func (a *Arena) String() string {
	stats := a.Stats()
	return fmt.Sprintf("Arena{blocks=%d, allocated=%d, used=%d, allocations=%d}",
		stats.BlocksAllocated, stats.BytesAllocated, stats.BytesUsed, stats.Allocations)
}
