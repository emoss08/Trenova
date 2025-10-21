package permissionservice

import (
	"hash/fnv"

	"github.com/emoss08/trenova/pkg/utils"
)

type bloomFilter struct {
	bits      []byte
	size      int
	hashCount int
}

func newBloomFilter(size, hashCount int) *bloomFilter {
	return &bloomFilter{
		bits:      make([]byte, size),
		size:      size,
		hashCount: hashCount,
	}
}

func (bf *bloomFilter) Add(item string) {
	for i := 0; i < bf.hashCount; i++ {
		hash := bf.hash(item, i)
		index := hash % utils.ConvertToUint32(bf.size*8)

		bf.bits[index/8] |= 1 << (index % 8)
	}
}

func (bf *bloomFilter) Test(item string) bool {
	for i := 0; i < bf.hashCount; i++ {
		hash := bf.hash(item, i)
		index := hash % utils.ConvertToUint32(bf.size*8)
		if (bf.bits[index/8] & (1 << (index % 8))) == 0 {
			return false
		}
	}

	return true
}

func (bf *bloomFilter) Bytes() []byte {
	return bf.bits
}

func (bf *bloomFilter) hash(item string, seed int) uint32 {
	h := fnv.New32a()
	h.Write([]byte(item))
	h.Write([]byte{byte(seed)})
	return h.Sum32()
}
