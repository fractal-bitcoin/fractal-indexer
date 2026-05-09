package script

func ExtractPkScriptForTxo(pkScript, scriptType []byte) (addressData *AddressData) {
	addressData = &AddressData{}

	if len(pkScript) == 0 {
		return addressData
	}

	if isPubkeyHash(scriptType) {
		addressData.HasAddress = true
		addressData.CodeType = CodeType_P2PKH
		copy(addressData.AddressPkh[:], pkScript[3:23])
		return addressData
	}

	if isPayToWitnessPubKeyHash(scriptType) {
		addressData.HasAddress = true
		addressData.CodeType = CodeType_P2WPKH
		copy(addressData.AddressPkh[:], GetHash160(pkScript))
		return addressData
	}

	if isPayToWitnessScriptHash(scriptType) {
		addressData.HasAddress = true
		addressData.CodeType = CodeType_P2WSH
		copy(addressData.AddressPkh[:], GetHash160(pkScript))
		return addressData
	}

	if isPayToTaproot(scriptType) {
		addressData.HasAddress = true
		addressData.CodeType = CodeType_P2TR
		copy(addressData.AddressPkh[:], GetHash160(pkScript))
		return addressData
	}

	if isPayToAnchor(scriptType) {
		addressData.HasAddress = true
		addressData.CodeType = CodeType_P2A
		copy(addressData.AddressPkh[:], GetHash160(pkScript))
		return addressData
	}

	if isPayToScriptHash(scriptType) {
		addressData.HasAddress = true
		addressData.CodeType = CodeType_P2SH
		copy(addressData.AddressPkh[:], GetHash160(pkScript))
		return addressData
	}

	if isPubkey(scriptType) {
		addressData.HasAddress = true
		addressData.CodeType = CodeType_P2PK
		copy(addressData.AddressPkh[:], GetHash160(pkScript))
		return addressData
	}

	// if isMultiSig(scriptType) {
	// 	return pkScript[:]
	// }

	if IsOpreturn(scriptType) {
		return addressData
	}

	return addressData
}

func GetLockingScriptType(pkScript []byte) (scriptType []byte) {
	length := len(pkScript)
	if length == 0 {
		return
	}
	scriptType = make([]byte, 0)

	lenType := 0
	p := uint(0)
	e := uint(length)

	for p < e && lenType < 32 {
		c := pkScript[p]
		if 0 < c && c < 0x4f {
			cnt, cntsize := SafeDecodeVarIntForScript(pkScript[p:])
			p += cnt + cntsize
			if p > e {
				break
			}
		} else {
			p += 1
		}
		scriptType = append(scriptType, c)
		lenType += 1
	}
	return
}
