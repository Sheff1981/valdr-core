package desktop

import (
	"encoding/base64"
	"errors"
	"fmt"
	"strings"

	valdrcrypto "github.com/Sheff1981/valdr-core/crypto"
)

const (
	qrVersion            = 3
	qrSize               = 29
	qrDataCodewords      = 34
	qrBlockDataCodewords = 17
	qrECCCodewords       = 18
	qrRemainderBits      = 7
	qrQuietZone          = 4
)

const qrAlphanumeric = "0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZ $%*+-./:"

var ErrQRCodePayload = errors.New("unsupported VALDR QR payload")

func AddressQRCodeDataURI(address string) (string, error) {
	address = strings.TrimSpace(address)
	if !valdrcrypto.ValidateAddress(address) {
		return "", valdrcrypto.ErrInvalidAddress
	}
	matrix, err := addressQRMatrix(address)
	if err != nil {
		return "", err
	}
	return qrMatrixDataURI(matrix), nil
}

func addressQRMatrix(address string) ([][]bool, error) {
	data, err := qrPayloadCodewords(address)
	if err != nil {
		return nil, err
	}

	blocks := [][]byte{
		append([]byte(nil), data[:qrBlockDataCodewords]...),
		append([]byte(nil), data[qrBlockDataCodewords:]...),
	}
	ecc := [][]byte{
		qrReedSolomonRemainder(blocks[0], qrECCCodewords),
		qrReedSolomonRemainder(blocks[1], qrECCCodewords),
	}

	codewords := make([]byte, 0, 70)
	for i := 0; i < qrBlockDataCodewords; i++ {
		for _, block := range blocks {
			codewords = append(codewords, block[i])
		}
	}
	for i := 0; i < qrECCCodewords; i++ {
		for _, blockECC := range ecc {
			codewords = append(codewords, blockECC[i])
		}
	}

	bits := make([]bool, 0, len(codewords)*8+qrRemainderBits)
	for _, value := range codewords {
		qrAppendBits(&bits, uint(value), 8)
	}
	for i := 0; i < qrRemainderBits; i++ {
		bits = append(bits, false)
	}

	matrix := make([][]bool, qrSize)
	function := make([][]bool, qrSize)
	for y := 0; y < qrSize; y++ {
		matrix[y] = make([]bool, qrSize)
		function[y] = make([]bool, qrSize)
	}

	setFunction := func(x, y int, dark bool) {
		if x < 0 || y < 0 || x >= qrSize || y >= qrSize {
			return
		}
		matrix[y][x] = dark
		function[y][x] = true
	}

	for i := 0; i < qrSize; i++ {
		setFunction(6, i, i%2 == 0)
		setFunction(i, 6, i%2 == 0)
	}

	drawFinder := func(cx, cy int) {
		for dy := -4; dy <= 4; dy++ {
			for dx := -4; dx <= 4; dx++ {
				x, y := cx+dx, cy+dy
				if x < 0 || y < 0 || x >= qrSize || y >= qrSize {
					continue
				}
				dist := qrMax(qrAbs(dx), qrAbs(dy))
				setFunction(x, y, dist != 2 && dist != 4)
			}
		}
	}
	drawFinder(3, 3)
	drawFinder(qrSize-4, 3)
	drawFinder(3, qrSize-4)

	for dy := -2; dy <= 2; dy++ {
		for dx := -2; dx <= 2; dx++ {
			dist := qrMax(qrAbs(dx), qrAbs(dy))
			setFunction(22+dx, 22+dy, dist != 1)
		}
	}

	format := qrFormatBits()
	for i := 0; i <= 5; i++ {
		setFunction(8, i, qrGetBit(format, i))
	}
	setFunction(8, 7, qrGetBit(format, 6))
	setFunction(8, 8, qrGetBit(format, 7))
	setFunction(7, 8, qrGetBit(format, 8))
	for i := 9; i < 15; i++ {
		setFunction(14-i, 8, qrGetBit(format, i))
	}
	for i := 0; i < 8; i++ {
		setFunction(qrSize-1-i, 8, qrGetBit(format, i))
	}
	for i := 8; i < 15; i++ {
		setFunction(8, qrSize-15+i, qrGetBit(format, i))
	}
	setFunction(8, qrSize-8, true)

	bitIndex := 0
	for right := qrSize - 1; right >= 1; right -= 2 {
		if right == 6 {
			right = 5
		}
		for vertical := 0; vertical < qrSize; vertical++ {
			upward := ((right + 1) & 2) == 0
			y := vertical
			if upward {
				y = qrSize - 1 - vertical
			}
			for offset := 0; offset < 2; offset++ {
				x := right - offset
				if function[y][x] {
					continue
				}
				dark := false
				if bitIndex < len(bits) {
					dark = bits[bitIndex]
				}
				bitIndex++
				if (x+y)%2 == 0 {
					dark = !dark
				}
				matrix[y][x] = dark
			}
		}
	}

	if bitIndex != len(bits) {
		return nil, ErrQRCodePayload
	}
	return matrix, nil
}

func qrPayloadCodewords(text string) ([]byte, error) {
	if len(text) != 43 {
		return nil, ErrQRCodePayload
	}

	bits := make([]bool, 0, qrDataCodewords*8)
	qrAppendBits(&bits, 0b0010, 4)
	qrAppendBits(&bits, uint(len(text)), 9)

	for i := 0; i+1 < len(text); i += 2 {
		first := strings.IndexByte(qrAlphanumeric, text[i])
		second := strings.IndexByte(qrAlphanumeric, text[i+1])
		if first < 0 || second < 0 {
			return nil, ErrQRCodePayload
		}
		qrAppendBits(&bits, uint(first*45+second), 11)
	}
	if len(text)%2 != 0 {
		last := strings.IndexByte(qrAlphanumeric, text[len(text)-1])
		if last < 0 {
			return nil, ErrQRCodePayload
		}
		qrAppendBits(&bits, uint(last), 6)
	}

	capacity := qrDataCodewords * 8
	if len(bits) > capacity {
		return nil, ErrQRCodePayload
	}
	for i := 0; i < 4 && len(bits) < capacity; i++ {
		bits = append(bits, false)
	}
	for len(bits)%8 != 0 {
		bits = append(bits, false)
	}

	data := make([]byte, 0, qrDataCodewords)
	for i := 0; i < len(bits); i += 8 {
		var value byte
		for _, bit := range bits[i : i+8] {
			value <<= 1
			if bit {
				value |= 1
			}
		}
		data = append(data, value)
	}

	pads := []byte{0xEC, 0x11}
	for pad := 0; len(data) < qrDataCodewords; pad++ {
		data = append(data, pads[pad%len(pads)])
	}
	return data, nil
}

func qrReedSolomonRemainder(data []byte, degree int) []byte {
	generator := []byte{1}
	root := byte(1)
	for i := 0; i < degree; i++ {
		next := make([]byte, len(generator)+1)
		for j, coefficient := range generator {
			next[j] ^= coefficient
			next[j+1] ^= qrGFMultiply(coefficient, root)
		}
		generator = next
		root = qrGFMultiply(root, 2)
	}

	remainder := make([]byte, degree)
	for _, value := range data {
		factor := value ^ remainder[0]
		copy(remainder, remainder[1:])
		remainder[len(remainder)-1] = 0
		for i, coefficient := range generator[1:] {
			remainder[i] ^= qrGFMultiply(coefficient, factor)
		}
	}
	return remainder
}

func qrGFMultiply(x, y byte) byte {
	var result uint16
	a := uint16(x)
	b := uint16(y)
	for b != 0 {
		if b&1 != 0 {
			result ^= a
		}
		b >>= 1
		a <<= 1
		if a&0x100 != 0 {
			a ^= 0x11D
		}
	}
	return byte(result)
}

func qrFormatBits() uint16 {
	data := uint16(0b11 << 3)
	remainder := data
	for i := 0; i < 10; i++ {
		remainder = (remainder << 1) ^ ((remainder >> 9) * 0x537)
	}
	return ((data << 10) | remainder) ^ 0x5412
}

func qrMatrixDataURI(matrix [][]bool) string {
	total := qrSize + 2*qrQuietZone
	var path strings.Builder
	for y, row := range matrix {
		for x, dark := range row {
			if dark {
				fmt.Fprintf(
					&path,
					"M%d %dh1v1h-1z",
					x+qrQuietZone,
					y+qrQuietZone,
				)
			}
		}
	}

	svg := fmt.Sprintf(
		`<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 %d %d" shape-rendering="crispEdges"><rect width="100%%" height="100%%" fill="white"/><path d="%s" fill="black"/></svg>`,
		total,
		total,
		path.String(),
	)
	return "data:image/svg+xml;base64," +
		base64.StdEncoding.EncodeToString([]byte(svg))
}

func qrAppendBits(target *[]bool, value uint, count int) {
	for i := count - 1; i >= 0; i-- {
		*target = append(*target, ((value>>i)&1) != 0)
	}
}

func qrGetBit(value uint16, index int) bool {
	return ((value >> index) & 1) != 0
}

func qrAbs(value int) int {
	if value < 0 {
		return -value
	}
	return value
}

func qrMax(a, b int) int {
	if a > b {
		return a
	}
	return b
}
