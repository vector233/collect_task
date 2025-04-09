package utility

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"

	"github.com/shengdoushi/base58"
)

// Base58ToHexAddress 将Base58格式的波场地址转换为十六进制格式
func Base58ToHexAddress(base58Address string) (string, error) {
	// 解码
	decoded, err := base58.Decode(base58Address, base58.BitcoinAlphabet)
	if err != nil {
		return "", fmt.Errorf("Base58解码失败: %v", err)
	}

	// 长度检查
	if len(decoded) < 4 {
		return "", fmt.Errorf("解码后的地址长度不正确")
	}

	// 分离地址和校验和
	addressBytes := decoded[:len(decoded)-4]
	checksumBytes := decoded[len(decoded)-4:]

	// 校验和验证
	firstSHA := sha256.Sum256(addressBytes)
	secondSHA := sha256.Sum256(firstSHA[:])
	expectedChecksum := secondSHA[:4]

	// 比较校验和
	for i := 0; i < 4; i++ {
		if checksumBytes[i] != expectedChecksum[i] {
			return "", fmt.Errorf("校验和不匹配，地址可能无效")
		}
	}

	// 转hex
	hexAddress := hex.EncodeToString(addressBytes)

	return hexAddress, nil
}

// HexAddressToBase58 十六进制地址转Base58
func HexAddressToBase58(hexAddress string) (string, error) {
	addressBytes, err := hex.DecodeString(hexAddress)
	if err != nil {
		return "", fmt.Errorf("解码地址失败: %v", err)
	}

	firstSHA := sha256.Sum256(addressBytes)
	secondSHA := sha256.Sum256(firstSHA[:])
	checksum := secondSHA[:4]

	addressWithChecksum := append(addressBytes, checksum...)
	base58Address := base58.Encode(addressWithChecksum, base58.BitcoinAlphabet)

	return base58Address, nil
}
