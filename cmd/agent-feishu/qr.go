package main

import (
	"errors"
	"fmt"
)

type qrSegment struct {
	data []byte
}

var qrDataCodewordsL = []int{
	0, 19, 34, 55, 80, 108, 136, 156, 194, 232, 274,
	324, 370, 428, 461, 523, 589, 647, 721, 795, 861,
	932, 1006, 1094, 1174, 1276, 1370, 1468, 1531, 1631, 1735,
	1843, 1955, 2071, 2191, 2306, 2434, 2566, 2702, 2812, 2956,
}

var qrECCCodewordsPerBlockL = []int{
	0, 7, 10, 15, 20, 26, 18, 20, 24, 30, 18,
	20, 24, 26, 30, 22, 24, 28, 30, 28, 28,
	28, 28, 30, 30, 26, 28, 30, 30, 30, 30,
	30, 30, 30, 30, 30, 30, 30, 30, 30, 30,
}

var qrNumBlocksL = []int{
	0, 1, 1, 1, 1, 1, 2, 2, 2, 2, 4,
	4, 4, 4, 4, 6, 6, 6, 6, 7, 8,
	8, 9, 9, 10, 12, 12, 12, 13, 14, 15,
	16, 17, 18, 19, 19, 20, 21, 22, 24, 25,
}

func makeQRCodeMatrix(text string) ([][]bool, error) {
	data := []byte(text)
	if len(data) == 0 {
		return nil, errors.New("empty QR content")
	}
	version := chooseQRVersion(len(data))
	if version == 0 {
		return nil, fmt.Errorf("QR content too long: %d bytes", len(data))
	}
	payload, err := makeQRDataCodewords(version, qrSegment{data: data})
	if err != nil {
		return nil, err
	}
	codewords := addQRErrorCorrection(version, payload)
	qr := newQR(version)
	qr.drawFunctionPatterns()
	qr.drawCodewords(codewords)
	mask := qr.chooseMask()
	qr.applyMask(mask)
	qr.drawFormatBits(mask)
	return qr.toBoolMatrix(), nil
}

func chooseQRVersion(byteLen int) int {
	for version := 1; version <= 40; version++ {
		capacityBits := qrDataCodewordsL[version] * 8
		ccBits := qrCharCountBits(version)
		neededBits := 4 + ccBits + byteLen*8
		if neededBits <= capacityBits {
			return version
		}
	}
	return 0
}

func qrCharCountBits(version int) int {
	if version <= 9 {
		return 8
	}
	return 16
}

func makeQRDataCodewords(version int, seg qrSegment) ([]byte, error) {
	capacityBits := qrDataCodewordsL[version] * 8
	bits := make([]bool, 0, capacityBits)
	appendBits := func(value, count int) {
		for i := count - 1; i >= 0; i-- {
			bits = append(bits, ((value>>i)&1) != 0)
		}
	}
	appendBits(0x4, 4)
	appendBits(len(seg.data), qrCharCountBits(version))
	for _, b := range seg.data {
		appendBits(int(b), 8)
	}
	terminator := minInt(4, capacityBits-len(bits))
	appendBits(0, terminator)
	for len(bits)%8 != 0 {
		bits = append(bits, false)
	}
	for pad := 0xec; len(bits) < capacityBits; pad ^= 0xec ^ 0x11 {
		appendBits(pad, 8)
	}
	result := make([]byte, len(bits)/8)
	for i, bit := range bits {
		if bit {
			result[i/8] |= 1 << uint(7-i%8)
		}
	}
	return result, nil
}

func addQRErrorCorrection(version int, data []byte) []byte {
	numBlocks := qrNumBlocksL[version]
	eccLen := qrECCCodewordsPerBlockL[version]
	rawCodewords := qrRawCodewords(version)
	numShortBlocks := numBlocks - rawCodewords%numBlocks
	shortBlockDataLen := rawCodewords/numBlocks - eccLen
	blocks := make([][]byte, numBlocks)
	k := 0
	for i := 0; i < numBlocks; i++ {
		dataLen := shortBlockDataLen
		if i >= numShortBlocks {
			dataLen++
		}
		blockData := append([]byte(nil), data[k:k+dataLen]...)
		k += dataLen
		ecc := qrReedSolomonRemainder(blockData, qrReedSolomonGenerator(eccLen))
		if i < numShortBlocks {
			blockData = append(blockData, 0)
		}
		blocks[i] = append(blockData, ecc...)
	}
	result := make([]byte, 0, rawCodewords)
	for i := 0; i < len(blocks[0]); i++ {
		for j, block := range blocks {
			if i == shortBlockDataLen && j < numShortBlocks {
				continue
			}
			if i < len(block) {
				result = append(result, block[i])
			}
		}
	}
	return result
}

func qrRawCodewords(version int) int {
	result := (16*version+128)*version + 64
	if version >= 2 {
		numAlign := version/7 + 2
		result -= (25*numAlign-10)*numAlign - 55
		if version >= 7 {
			result -= 36
		}
	}
	return result / 8
}

func qrReedSolomonGenerator(degree int) []byte {
	result := make([]byte, degree)
	result[degree-1] = 1
	root := byte(1)
	for i := 0; i < degree; i++ {
		for j := 0; j < len(result); j++ {
			result[j] = qrGFMul(result[j], root)
			if j+1 < len(result) {
				result[j] ^= result[j+1]
			}
		}
		root = qrGFMul(root, 0x02)
	}
	return result
}

func qrReedSolomonRemainder(data, generator []byte) []byte {
	result := make([]byte, len(generator))
	for _, b := range data {
		factor := b ^ result[0]
		copy(result, result[1:])
		result[len(result)-1] = 0
		for i, coef := range generator {
			result[i] ^= qrGFMul(coef, factor)
		}
	}
	return result
}

func qrGFMul(x, y byte) byte {
	z := 0
	a := int(x)
	b := int(y)
	for i := 7; i >= 0; i-- {
		z = (z << 1) ^ ((z >> 7) * 0x11d)
		if ((b >> i) & 1) != 0 {
			z ^= a
		}
	}
	return byte(z)
}

type qrCode struct {
	version int
	size    int
	modules [][]int8
	isFunc  [][]bool
}

func newQR(version int) *qrCode {
	size := version*4 + 17
	modules := make([][]int8, size)
	isFunc := make([][]bool, size)
	for y := 0; y < size; y++ {
		modules[y] = make([]int8, size)
		isFunc[y] = make([]bool, size)
		for x := 0; x < size; x++ {
			modules[y][x] = -1
		}
	}
	return &qrCode{version: version, size: size, modules: modules, isFunc: isFunc}
}

func (q *qrCode) drawFunctionPatterns() {
	for i := 0; i < q.size; i++ {
		q.setFunctionModule(6, i, i%2 == 0)
		q.setFunctionModule(i, 6, i%2 == 0)
	}
	q.drawFinder(3, 3)
	q.drawFinder(q.size-4, 3)
	q.drawFinder(3, q.size-4)
	align := q.alignmentPatternPositions()
	for _, y := range align {
		for _, x := range align {
			if (x == 6 && y == 6) || (x == 6 && y == q.size-7) || (x == q.size-7 && y == 6) {
				continue
			}
			q.drawAlignment(x, y)
		}
	}
	q.drawFormatBits(0)
	if q.version >= 7 {
		q.drawVersion()
	}
}

func (q *qrCode) drawFinder(cx, cy int) {
	for dy := -4; dy <= 4; dy++ {
		for dx := -4; dx <= 4; dx++ {
			x := cx + dx
			y := cy + dy
			if 0 <= x && x < q.size && 0 <= y && y < q.size {
				dist := maxInt(absInt(dx), absInt(dy))
				q.setFunctionModule(x, y, dist != 2 && dist != 4)
			}
		}
	}
}

func (q *qrCode) drawAlignment(cx, cy int) {
	for dy := -2; dy <= 2; dy++ {
		for dx := -2; dx <= 2; dx++ {
			q.setFunctionModule(cx+dx, cy+dy, maxInt(absInt(dx), absInt(dy)) != 1)
		}
	}
}

func (q *qrCode) alignmentPatternPositions() []int {
	if q.version == 1 {
		return nil
	}
	numAlign := q.version/7 + 2
	step := 0
	if q.version == 32 {
		step = 26
	} else {
		step = ((q.version*4 + numAlign*2 + 1) / (numAlign*2 - 2)) * 2
	}
	result := make([]int, numAlign)
	result[0] = 6
	for i, pos := numAlign-1, q.size-7; i >= 1; i, pos = i-1, pos-step {
		result[i] = pos
	}
	return result
}

func (q *qrCode) drawVersion() {
	rem := q.version
	for i := 0; i < 12; i++ {
		rem = (rem << 1) ^ ((rem >> 11) * 0x1f25)
	}
	bits := q.version<<12 | rem
	for i := 0; i < 18; i++ {
		bit := ((bits >> i) & 1) != 0
		a := q.size - 11 + i%3
		b := i / 3
		q.setFunctionModule(a, b, bit)
		q.setFunctionModule(b, a, bit)
	}
}

func (q *qrCode) drawFormatBits(mask int) {
	data := 0x08 | mask
	rem := data
	for i := 0; i < 10; i++ {
		rem = (rem << 1) ^ ((rem >> 9) * 0x537)
	}
	bits := (data<<10 | rem) ^ 0x5412
	for i := 0; i <= 5; i++ {
		q.setFunctionModule(8, i, ((bits>>i)&1) != 0)
	}
	q.setFunctionModule(8, 7, ((bits>>6)&1) != 0)
	q.setFunctionModule(8, 8, ((bits>>7)&1) != 0)
	q.setFunctionModule(7, 8, ((bits>>8)&1) != 0)
	for i := 9; i < 15; i++ {
		q.setFunctionModule(14-i, 8, ((bits>>i)&1) != 0)
	}
	for i := 0; i < 8; i++ {
		q.setFunctionModule(q.size-1-i, 8, ((bits>>i)&1) != 0)
	}
	for i := 8; i < 15; i++ {
		q.setFunctionModule(8, q.size-15+i, ((bits>>i)&1) != 0)
	}
	q.setFunctionModule(8, q.size-8, true)
}

func (q *qrCode) drawCodewords(data []byte) {
	i := 0
	for right := q.size - 1; right >= 1; right -= 2 {
		if right == 6 {
			right--
		}
		for vert := 0; vert < q.size; vert++ {
			y := vert
			if ((right + 1) & 2) == 0 {
				y = q.size - 1 - vert
			}
			for j := 0; j < 2; j++ {
				x := right - j
				if !q.isFunc[y][x] && i < len(data)*8 {
					q.modules[y][x] = int8((data[i>>3] >> uint(7-(i&7))) & 1)
					i++
				}
			}
		}
	}
}

func (q *qrCode) chooseMask() int {
	bestMask := 0
	bestPenalty := int(^uint(0) >> 1)
	for mask := 0; mask < 8; mask++ {
		q.applyMask(mask)
		q.drawFormatBits(mask)
		penalty := q.penaltyScore()
		if penalty < bestPenalty {
			bestPenalty = penalty
			bestMask = mask
		}
		q.applyMask(mask)
	}
	return bestMask
}

func (q *qrCode) applyMask(mask int) {
	for y := 0; y < q.size; y++ {
		for x := 0; x < q.size; x++ {
			if q.isFunc[y][x] {
				continue
			}
			invert := false
			switch mask {
			case 0:
				invert = (x+y)%2 == 0
			case 1:
				invert = y%2 == 0
			case 2:
				invert = x%3 == 0
			case 3:
				invert = (x+y)%3 == 0
			case 4:
				invert = (x/3+y/2)%2 == 0
			case 5:
				invert = x*y%2+x*y%3 == 0
			case 6:
				invert = (x*y%2+x*y%3)%2 == 0
			case 7:
				invert = ((x+y)%2+x*y%3)%2 == 0
			}
			if invert {
				q.modules[y][x] ^= 1
			}
		}
	}
}

func (q *qrCode) penaltyScore() int {
	result := 0
	for y := 0; y < q.size; y++ {
		runColor := int8(-1)
		runLen := 0
		for x := 0; x < q.size; x++ {
			color := q.modules[y][x]
			if color == runColor {
				runLen++
				if runLen == 5 {
					result += 3
				} else if runLen > 5 {
					result++
				}
			} else {
				runColor = color
				runLen = 1
			}
		}
	}
	for x := 0; x < q.size; x++ {
		runColor := int8(-1)
		runLen := 0
		for y := 0; y < q.size; y++ {
			color := q.modules[y][x]
			if color == runColor {
				runLen++
				if runLen == 5 {
					result += 3
				} else if runLen > 5 {
					result++
				}
			} else {
				runColor = color
				runLen = 1
			}
		}
	}
	for y := 0; y < q.size-1; y++ {
		for x := 0; x < q.size-1; x++ {
			c := q.modules[y][x]
			if c == q.modules[y][x+1] && c == q.modules[y+1][x] && c == q.modules[y+1][x+1] {
				result += 3
			}
		}
	}
	black := 0
	for y := 0; y < q.size; y++ {
		for x := 0; x < q.size; x++ {
			if q.modules[y][x] == 1 {
				black++
			}
		}
	}
	total := q.size * q.size
	k := absInt(black*20/total - 10)
	result += k * 10
	return result
}

func (q *qrCode) setFunctionModule(x, y int, dark bool) {
	if 0 <= x && x < q.size && 0 <= y && y < q.size {
		if dark {
			q.modules[y][x] = 1
		} else {
			q.modules[y][x] = 0
		}
		q.isFunc[y][x] = true
	}
}

func (q *qrCode) toBoolMatrix() [][]bool {
	border := 4
	result := make([][]bool, q.size+border*2)
	for y := range result {
		result[y] = make([]bool, q.size+border*2)
	}
	for y := 0; y < q.size; y++ {
		for x := 0; x < q.size; x++ {
			result[y+border][x+border] = q.modules[y][x] == 1
		}
	}
	return result
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func absInt(value int) int {
	if value < 0 {
		return -value
	}
	return value
}
