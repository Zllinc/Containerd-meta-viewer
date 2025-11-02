package utils

import (
	"encoding/binary"
)

// ReadID reads a uint64 ID from bucket data
func ReadID(data []byte) uint64 {
	id, _ := binary.Uvarint(data)
	return id
}

// ReadSize reads a int64 size from bucket data
func ReadSize(data []byte) int64 {
	size, _ := binary.Varint(data)
	return size
}

// ReadInodes reads a int64 inodes count from bucket data
func ReadInodes(data []byte) int64 {
	inodes, _ := binary.Varint(data)
	return inodes
}

// EncodeID encodes a uint64 ID to bytes
func EncodeID(buf []byte, id uint64) int {
	return binary.PutUvarint(buf, id)
}

// EncodeSize encodes a int64 size to bytes
func EncodeSize(buf []byte, size int64) int {
	return binary.PutVarint(buf, size)
}