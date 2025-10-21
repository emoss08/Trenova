/*
 * Copyright 2025 Eric Moss
 * Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
 * Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md */

/**
 * BloomFilter implementation for fast negative permission checks
 * Uses MurmurHash3 for hashing
 */
export class BloomFilter {
  private bits: Uint8Array;
  private size: number;
  private hashCount: number;

  constructor(base64Data?: string) {
    if (base64Data) {
      // Decode base64 data from server
      const decoded = atob(base64Data);
      this.bits = new Uint8Array(decoded.length);
      for (let i = 0; i < decoded.length; i++) {
        this.bits[i] = decoded.charCodeAt(i);
      }
      this.size = this.bits.length * 8;
      this.hashCount = 3; // Matches server implementation
    } else {
      // Create empty bloom filter (1KB default)
      this.size = 8192;
      this.bits = new Uint8Array(this.size / 8);
      this.hashCount = 3;
    }
  }

  /**
   * Test if a key might exist in the set
   * Returns false if definitely not present, true if possibly present
   */
  test(key: string): boolean {
    const hashes = this.getHashes(key);

    for (let i = 0; i < this.hashCount; i++) {
      const bitIndex = hashes[i] % this.size;
      const byteIndex = Math.floor(bitIndex / 8);
      const bitOffset = bitIndex % 8;

      if ((this.bits[byteIndex] & (1 << bitOffset)) === 0) {
        return false; // Definitely not present
      }
    }

    return true; // Possibly present
  }

  /**
   * Add a key to the bloom filter
   */
  add(key: string): void {
    const hashes = this.getHashes(key);

    for (let i = 0; i < this.hashCount; i++) {
      const bitIndex = hashes[i] % this.size;
      const byteIndex = Math.floor(bitIndex / 8);
      const bitOffset = bitIndex % 8;

      this.bits[byteIndex] |= 1 << bitOffset;
    }
  }

  /**
   * Generate hash values for a key using MurmurHash3
   */
  private getHashes(key: string): number[] {
    const hash1 = this.murmurHash3(key, 0);
    const hash2 = this.murmurHash3(key, hash1);

    const hashes: number[] = [];
    for (let i = 0; i < this.hashCount; i++) {
      hashes.push(Math.abs((hash1 + i * hash2) >>> 0));
    }

    return hashes;
  }

  /**
   * MurmurHash3 32-bit implementation
   */
  private murmurHash3(key: string, seed: number): number {
    const data = new TextEncoder().encode(key);
    const c1 = 0xcc9e2d51;
    const c2 = 0x1b873593;
    let h1 = seed;
    const roundedEnd = Math.floor(data.length / 4) * 4;

    for (let i = 0; i < roundedEnd; i += 4) {
      let k1 =
        (data[i] & 0xff) |
        ((data[i + 1] & 0xff) << 8) |
        ((data[i + 2] & 0xff) << 16) |
        ((data[i + 3] & 0xff) << 24);

      k1 = Math.imul(k1, c1);
      k1 = (k1 << 15) | (k1 >>> 17);
      k1 = Math.imul(k1, c2);

      h1 ^= k1;
      h1 = (h1 << 13) | (h1 >>> 19);
      h1 = Math.imul(h1, 5) + 0xe6546b64;
    }

    let k1 = 0;
    const remaining = data.length % 4;

    if (remaining >= 3) k1 ^= (data[roundedEnd + 2] & 0xff) << 16;
    if (remaining >= 2) k1 ^= (data[roundedEnd + 1] & 0xff) << 8;
    if (remaining >= 1) {
      k1 ^= data[roundedEnd] & 0xff;
      k1 = Math.imul(k1, c1);
      k1 = (k1 << 15) | (k1 >>> 17);
      k1 = Math.imul(k1, c2);
      h1 ^= k1;
    }

    h1 ^= data.length;
    h1 ^= h1 >>> 16;
    h1 = Math.imul(h1, 0x85ebca6b);
    h1 ^= h1 >>> 13;
    h1 = Math.imul(h1, 0xc2b2ae35);
    h1 ^= h1 >>> 16;

    return h1 >>> 0;
  }

  /**
   * Export bloom filter to base64 (for storage)
   */
  toBase64(): string {
    let binary = "";
    for (let i = 0; i < this.bits.length; i++) {
      binary += String.fromCharCode(this.bits[i]);
    }
    return btoa(binary);
  }
}
