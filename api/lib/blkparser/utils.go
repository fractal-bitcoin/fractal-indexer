package blkparser

import "encoding/binary"

func DecodeVarIntForBlock(raw []byte) (cnt uint, cnt_size uint) {
	if raw[0] < 0xfd {
		return uint(raw[0]), 1
	} else if raw[0] == 0xfd {
		return uint(binary.LittleEndian.Uint16(raw[1:3])), 3
	} else if raw[0] == 0xfe {
		return uint(binary.LittleEndian.Uint32(raw[1:5])), 5
	} else {
		return uint(binary.LittleEndian.Uint64(raw[1:9])), 9
	}
}

func DecodeVarIntForTx(raw []byte) (cnt uint, cnt_size uint) {
	if len(raw) < 1 {
		return 0, 0
	}
	if raw[0] < 0xfd {
		return uint(raw[0]), 1
	}

	if raw[0] == 0xfd {
		if len(raw) < 3 {
			return 0, 0
		}
		return uint(binary.LittleEndian.Uint16(raw[1:3])), 3

	} else if raw[0] == 0xfe {
		if len(raw) < 5 {
			return 0, 0
		}
		return uint(binary.LittleEndian.Uint32(raw[1:5])), 5
	}

	if len(raw) < 9 {
		return 0, 0
	}
	return uint(binary.LittleEndian.Uint64(raw[1:9])), 9
}
