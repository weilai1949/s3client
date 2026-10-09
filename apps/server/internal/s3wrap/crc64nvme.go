package s3wrap

// crc64nvme.go —— 流式 CRC-64/NVME 计算（S3 全对象校验和算法）。
//
// 参数（CRC RevEng 目录「CRC-64/NVME」，与 AWS aws-checksums 的 aws_checksums_crc64nvme 一致）：
//
//	width=64 poly=0xad93d23594c93659 init=0xffffffffffffffff refin=true refout=true
//	xorout=0xffffffffffffffff check("123456789")=0xae8b14860a799888
//
// 采用反射（小端位序）逐字节查表：crc = table[(crc ^ byte) & 0xff] ^ (crc >> 8)。
// S3 / RustFS 以 8 字节**大端** base64 返回该值（已用真实对端实测确认）。

import (
	"encoding/binary"
	"hash"
)

// crc64NVMEInit CRC 的初值与最终异或值（均为全 1）。
const crc64NVMEInit uint64 = 0xFFFFFFFFFFFFFFFF

// reflect64 反转 64 位整数的位序（把非反射多项式转为反射形式）。
func reflect64(v uint64) uint64 {
	var r uint64
	for i := 0; i < 64; i++ {
		if v&(1<<i) != 0 {
			r |= 1 << (63 - i)
		}
	}
	return r
}

// crc64NVMETable 反射式查表（256 项，一次构建、只读共享）。
var crc64NVMETable = buildCRC64NVMETable()

func buildCRC64NVMETable() [256]uint64 {
	poly := reflect64(0xad93d23594c93659)
	var table [256]uint64
	for i := range table {
		crc := uint64(i)
		for j := 0; j < 8; j++ {
			if crc&1 != 0 {
				crc = (crc >> 1) ^ poly
			} else {
				crc >>= 1
			}
		}
		table[i] = crc
	}
	return table
}

// crc64NVME 是流式 CRC-64/NVME 计算器（hash.Hash + hash.Hash64，对象内容边读边算）。
type crc64NVME struct{ crc uint64 }

var (
	_ hash.Hash   = (*crc64NVME)(nil)
	_ hash.Hash64 = (*crc64NVME)(nil)
)

func newCRC64NVME() *crc64NVME { return &crc64NVME{crc: crc64NVMEInit} }

func (h *crc64NVME) Write(p []byte) (int, error) {
	crc := h.crc
	for _, b := range p {
		//nolint:gosec // G115：仅取 crc 低 8 位查表，截断即反射式 CRC 的算法定义
		crc = crc64NVMETable[byte(crc)^b] ^ (crc >> 8)
	}
	h.crc = crc
	return len(p), nil
}

func (h *crc64NVME) Sum64() uint64 { return h.crc ^ crc64NVMEInit }

func (h *crc64NVME) Sum(b []byte) []byte {
	var out [8]byte
	binary.BigEndian.PutUint64(out[:], h.Sum64())
	return append(b, out[:]...)
}

func (h *crc64NVME) Reset() { h.crc = crc64NVMEInit }

func (h *crc64NVME) Size() int { return 8 }

// BlockSize 逐字节更新（查表实现按 1 字节推进）。
func (h *crc64NVME) BlockSize() int { return 1 }
