package main

import (
	"encoding/binary"
	"testing"
	"unicode/utf16"
)

func TestDecodeRawTextBytesUTF8(t *testing.T) {
	text := "需要清理 Git 标签锁文件"
	if got := decodeRawTextBytes([]byte(text)); got != text {
		t.Fatalf("decodeRawTextBytes() = %q, want %q", got, text)
	}
}

func TestDecodeRawTextBytesUTF16LE(t *testing.T) {
	text := "需要清理 Git 标签锁文件"
	words := utf16.Encode([]rune(text))
	body := []byte{0xff, 0xfe}
	for _, word := range words {
		next := make([]byte, 2)
		binary.LittleEndian.PutUint16(next, word)
		body = append(body, next...)
	}
	if got := decodeRawTextBytes(body); got != text {
		t.Fatalf("decodeRawTextBytes() = %q, want %q", got, text)
	}
}
