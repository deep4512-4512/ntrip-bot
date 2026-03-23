package main

import (
	"bufio"
	"fmt"
	"io"
)

type BitReader struct {
	data []byte
	pos  int
}

func (b *BitReader) ReadBits(n int) (uint64, error) {
	if n < 0 {
		return 0, fmt.Errorf("invalid bit count: %d", n)
	}
	if b.pos+n > len(b.data)*8 {
		return 0, io.ErrUnexpectedEOF
	}

	var v uint64
	for i := 0; i < n; i++ {
		byteIndex := b.pos / 8
		bitIndex := 7 - (b.pos % 8)
		bit := (b.data[byteIndex] >> bitIndex) & 1
		v = (v << 1) | uint64(bit)
		b.pos++
	}
	return v, nil
}

func parseMSM(p []byte, t int) ([]int, []float64, error) {
	br := &BitReader{data: p}
	if _, err := br.ReadBits(12 + 12 + 30 + 1 + 3 + 7 + 2 + 2); err != nil {
		return nil, nil, err
	}

	var sats []int
	for i := 0; i < 64; i++ {
		bit, err := br.ReadBits(1)
		if err != nil {
			return nil, nil, err
		}
		if bit == 1 {
			sats = append(sats, i+1)
		}
	}

	sigCount := 0
	for i := 0; i < 32; i++ {
		bit, err := br.ReadBits(1)
		if err != nil {
			return nil, nil, err
		}
		if bit == 1 {
			sigCount++
		}
	}

	cellMask := make([]bool, len(sats)*sigCount)
	for i := range cellMask {
		bit, err := br.ReadBits(1)
		if err != nil {
			return nil, nil, err
		}
		cellMask[i] = bit == 1
	}

	for range sats {
		if _, err := br.ReadBits(8 + 4 + 10); err != nil {
			return nil, nil, err
		}
		if t%10 >= 5 {
			if _, err := br.ReadBits(14); err != nil {
				return nil, nil, err
			}
		}
	}

	if t%10 != 7 {
		return sats, nil, nil
	}

	var snrs []float64
	for _, active := range cellMask {
		if !active {
			continue
		}
		if _, err := br.ReadBits(20 + 24 + 10); err != nil {
			return nil, nil, err
		}
		cnr, err := br.ReadBits(10)
		if err != nil {
			return nil, nil, err
		}
		if _, err := br.ReadBits(15); err != nil {
			return nil, nil, err
		}
		snrs = append(snrs, float64(cnr)*0.0625)
	}

	res := make([]float64, len(sats))
	cnt := make([]int, len(sats))

	idx := 0
	for i := range sats {
		for j := 0; j < sigCount && idx < len(snrs); j++ {
			res[i] += snrs[idx]
			cnt[i]++
			idx++
		}
	}

	for i := range res {
		if cnt[i] > 0 {
			res[i] /= float64(cnt[i])
		}
	}

	return sats, res, nil
}

func system(msg int) string {
	switch {
	case msg >= 1070 && msg < 1080:
		return "GPS"
	case msg >= 1080 && msg < 1090:
		return "GLO"
	case msg >= 1090 && msg < 1100:
		return "GAL"
	case msg >= 1120 && msg < 1130:
		return "BDS"
	default:
		return ""
	}
}

func readRTCM(r *bufio.Reader) (int, []int, []float64, error) {
	for {
		b, err := r.ReadByte()
		if err != nil {
			return 0, nil, nil, err
		}
		if b != 0xD3 {
			continue
		}

		head := make([]byte, 2)
		if _, err := io.ReadFull(r, head); err != nil {
			return 0, nil, nil, err
		}

		payloadLen := int(head[0]&0x03)<<8 | int(head[1])
		if payloadLen < 2 {
			return 0, nil, nil, fmt.Errorf("rtcm payload too short: %d", payloadLen)
		}

		payload := make([]byte, payloadLen)
		if _, err := io.ReadFull(r, payload); err != nil {
			return 0, nil, nil, err
		}

		crc := make([]byte, 3)
		if _, err := io.ReadFull(r, crc); err != nil {
			return 0, nil, nil, err
		}

		msg := int(payload[0])<<4 | int(payload[1]>>4)
		if msg < 1070 || msg > 1137 {
			continue
		}

		sats, snr, err := parseMSM(payload, msg)
		if err != nil {
			return 0, nil, nil, fmt.Errorf("parse msm %d: %w", msg, err)
		}
		return msg, sats, snr, nil
	}
}
