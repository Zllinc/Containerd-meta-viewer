package utils

import (
	"encoding/binary"
	"testing"
)

// encodeUvarint encodes a uint64 to varint bytes
func encodeUvarint(n uint64) []byte {
	buf := make([]byte, binary.MaxVarintLen64)
	size := binary.PutUvarint(buf, n)
	return buf[:size]
}

// encodeVarint encodes a int64 to varint bytes
func encodeVarint(n int64) []byte {
	buf := make([]byte, binary.MaxVarintLen64)
	size := binary.PutVarint(buf, n)
	return buf[:size]
}

func TestReadID(t *testing.T) {
	tests := []struct {
		name     string
		data     []byte
		expected uint64
	}{
		{
			name:     "valid single byte",
			data:     encodeUvarint(1),
			expected: 1,
		},
		{
			name:     "valid multi-byte",
			data:     encodeUvarint(150),
			expected: 150,
		},
		{
			name:     "zero value",
			data:     encodeUvarint(0),
			expected: 0,
		},
		{
			name:     "empty data",
			data:     []byte{},
			expected: 0,
		},
		{
			name:     "large number",
			data:     encodeUvarint(2147483648),
			expected: 2147483648,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ReadID(tt.data)
			if result != tt.expected {
				t.Errorf("ReadID() = %v, expected %v", result, tt.expected)
			}
		})
	}
}

func TestReadSize(t *testing.T) {
	tests := []struct {
		name     string
		data     []byte
		expected int64
	}{
		{
			name:     "positive single byte",
			data:     encodeVarint(2),
			expected: 2,
		},
		{
			name:     "positive multi-byte",
			data:     encodeVarint(150),
			expected: 150,
		},
		{
			name:     "zero value",
			data:     encodeVarint(0),
			expected: 0,
		},
		{
			name:     "negative single byte",
			data:     encodeVarint(-63),
			expected: -63,
		},
		{
			name:     "empty data",
			data:     []byte{},
			expected: 0,
		},
		{
			name:     "large positive",
			data:     encodeVarint(2147483648),
			expected: 2147483648,
		},
		{
			name:     "large negative",
			data:     encodeVarint(-2147483648),
			expected: -2147483648,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ReadSize(tt.data)
			if result != tt.expected {
				t.Errorf("ReadSize() = %v, expected %v", result, tt.expected)
			}
		})
	}
}

func TestReadInodes(t *testing.T) {
	tests := []struct {
		name     string
		data     []byte
		expected int64
	}{
		{
			name:     "valid inodes count",
			data:     encodeVarint(100),
			expected: 100,
		},
		{
			name:     "zero inodes",
			data:     encodeVarint(0),
			expected: 0,
		},
		{
			name:     "empty data",
			data:     []byte{},
			expected: 0,
		},
		{
			name:     "large inodes count",
			data:     encodeVarint(100000),
			expected: 100000,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ReadInodes(tt.data)
			if result != tt.expected {
				t.Errorf("ReadInodes() = %v, expected %v", result, tt.expected)
			}
		})
	}
}

// Benchmark tests
func BenchmarkReadID(b *testing.B) {
	data := encodeUvarint(2147483648)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ReadID(data)
	}
}

func BenchmarkReadSize(b *testing.B) {
	data := encodeVarint(2147483648)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ReadSize(data)
	}
}

func BenchmarkReadInodes(b *testing.B) {
	data := encodeVarint(100000)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ReadInodes(data)
	}
}