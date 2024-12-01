package ninja_go

import (
	"encoding/binary"
	"lukechampine.com/uint128"
)

// RAPIDHASH_NOEXCEPT 在 Go 中不需要，因为 Go 没有异常
// RAPIDHASH_CONSTEXPR 在 Go 中使用 const 替代
// RAPIDHASH_INLINE 在 Go 中使用 inline 替代

// RAPIDHASH_PROTECTED 和 RAPIDHASH_FAST 可以通过编译时标志来控制

// RAPIDHASH_LITTLE_ENDIAN 在 Go 中不需要，因为 Go 已经有了 byte order 包

// RAPID_SEED 是默认种子
const RAPID_SEED uint64 = 0xbdd89aa982704029

// rapid_secret 是默认的密钥参数
var rapid_secret = [3]uint64{0x2d358dccaa6c78a5, 0x8bb84b93962eacc9, 0x4b33a62ed433d4a3}

var RAPIDHASH_PROTECTED = false

// rapid_mum 实现 64*64 -> 128 位乘法
func rapid_mum(a, b *uint64) {
	var low, high uint64
	var r uint128.Uint128 = uint128.From64(*a)
	r = r.Mul(uint128.From64(*b))
	low, high = r.Lo, r.Hi
	if RAPIDHASH_PROTECTED {
		*a ^= low
		*b ^= high
	} else {
		*a = low
		*b = high
	}
}

// rapid_mix 实现乘法和异或混合
func rapid_mix(a, b uint64) uint64 {
	rapid_mum(&a, &b)
	return a ^ b
}

// rapid_read64 读取 64 位整数
func rapid_read64(p []byte) uint64 {
	return binary.LittleEndian.Uint64(p)
}

// rapid_read32 读取 32 位整数
func rapid_read32(p []byte) uint32 {
	return binary.LittleEndian.Uint32(p)
}

// rapid_readSmall 读取并组合 3 个字节的输入
func rapid_readSmall(p []byte, k int) uint64 {
	return uint64(p[0])<<56 | uint64(p[k>>1])<<32 | uint64(p[k-1])
}

var RAPIDHASH_UNROLLED = false

// rapidhash_internal 是 rapidhash 的主要函数
func rapidhash_internal(key []byte, len int, seed uint64, secret [3]uint64) uint64 {
	p := key
	seed ^= rapid_mix(seed^secret[0], secret[1]) ^ uint64(len)
	var a, b uint64

	if len <= 16 {
		if len >= 4 {
			plast := p[len-4:]
			a = uint64((binary.LittleEndian.Uint32(p) << 32) | binary.LittleEndian.Uint32(plast))
			delta := uint64((len & 24) >> (len >> 3))
			b = uint64((binary.LittleEndian.Uint32(p[delta:]) << 32) | binary.LittleEndian.Uint32(plast[-delta:]))
		} else if len > 0 {
			a = rapid_readSmall(p, len)
			b = 0
		} else {
			a = 0
			b = 0
		}
	} else {
		i := len
		if i > 48 {
			see1 := seed
			see2 := seed

			// 假设 RAPIDHASH_UNROLLED 是预定义的
			if RAPIDHASH_UNROLLED {
				for i >= 96 {
					seed = rapid_mix(binary.LittleEndian.Uint64(p)^secret[0], binary.LittleEndian.Uint64(p[8:])^seed)
					see1 = rapid_mix(binary.LittleEndian.Uint64(p[16:])^secret[1], binary.LittleEndian.Uint64(p[24:])^see1)
					see2 = rapid_mix(binary.LittleEndian.Uint64(p[32:])^secret[2], binary.LittleEndian.Uint64(p[40:])^see2)
					seed = rapid_mix(binary.LittleEndian.Uint64(p[48:])^secret[0], binary.LittleEndian.Uint64(p[56:])^seed)
					see1 = rapid_mix(binary.LittleEndian.Uint64(p[64:])^secret[1], binary.LittleEndian.Uint64(p[72:])^see1)
					see2 = rapid_mix(binary.LittleEndian.Uint64(p[80:])^secret[2], binary.LittleEndian.Uint64(p[88:])^see2)
					p = p[96:]
					i -= 96
				}
				if i >= 48 {
					seed = rapid_mix(binary.LittleEndian.Uint64(p)^secret[0], binary.LittleEndian.Uint64(p[8:])^seed)
					see1 = rapid_mix(binary.LittleEndian.Uint64(p[16:])^secret[1], binary.LittleEndian.Uint64(p[24:])^see1)
					see2 = rapid_mix(binary.LittleEndian.Uint64(p[32:])^secret[2], binary.LittleEndian.Uint64(p[40:])^see2)
					p = p[48:]
					i -= 48
				}
			} else {
				for i >= 48 {
					seed = rapid_mix(binary.LittleEndian.Uint64(p)^secret[0], binary.LittleEndian.Uint64(p[8:])^seed)
					see1 = rapid_mix(binary.LittleEndian.Uint64(p[16:])^secret[1], binary.LittleEndian.Uint64(p[24:])^see1)
					see2 = rapid_mix(binary.LittleEndian.Uint64(p[32:])^secret[2], binary.LittleEndian.Uint64(p[40:])^see2)
					p = p[48:]
					i -= 48
				}
			}
			seed ^= see1 ^ see2
		}
		if i > 16 {
			seed = rapid_mix(binary.LittleEndian.Uint64(p)^secret[2], binary.LittleEndian.Uint64(p[8:])^seed^secret[1])
			if i > 32 {
				seed = rapid_mix(binary.LittleEndian.Uint64(p[16:])^secret[2], binary.LittleEndian.Uint64(p[24:])^seed)
			}
		}
		a = binary.LittleEndian.Uint64(p[len-16:])
		b = binary.LittleEndian.Uint64(p[len-8:])
	}
	a ^= secret[1]
	b ^= seed
	rapid_mum(&a, &b)
	return rapid_mix(a^secret[0]^uint64(len), b^secret[1])
}

// rapidhash_withSeed 是使用提供的种子的 rapidhash 函数
func rapidhash_withSeed(key []byte, len int, seed uint64) uint64 {
	return rapidhash_internal(key, len, seed, rapid_secret)
}

// rapidhash 是默认的 rapidhash 函数
func rapidhash(key []byte, len int) uint64 {
	return rapidhash_withSeed(key, len, RAPID_SEED)
}
