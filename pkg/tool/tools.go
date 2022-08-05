package tool

func S264B(s string) [64]uint8 {
	var bs [64]byte
	slen := len(s)

	if slen == 0 {
		return bs
	}

	if slen >= 64 {
		for i := 0; i < 64; i++ {
			bs[i] = s[i]
		}
		return bs
	}

	if slen < 64 {
		for i := 0; i < slen; i++ {
			bs[i] = s[i]
		}
		for i := 0; i < 64-slen; i++ {
			bs[slen+i] = 0
		}
		return bs
	}

	return bs
}

func B642S(bs [64]uint8) string {
	ba := make([]byte, 0, 64)
	for _, b := range bs {
		if b == 0 {
			break
		}
		ba = append(ba, b)
	}
	return string(ba)
}
