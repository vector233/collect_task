package main

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"log"

	"github.com/btcsuite/btcd/btcec/v2"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/shengdoushi/base58"
)

func main() {
	// 示例私钥（请替换为您的私钥）
	privateKeyHex := "AFD86F96DE4C43E7361E2AEE40680963FDEB98C658E9F466913B69E5D58652A5"
	targetAddress := "TDxGAVpqCT4R3g4VCb97TinqQndk41HAXE"

	// 从私钥获取波场地址
	// address, err := getAddressFromPrivateKey(privateKeyHex)
	// if err != nil {
	// 	log.Fatalf("获取地址失败: %v", err)
	// }
	// fmt.Printf("波场地址: %s\n", address)
	// 生成地址并验证
	match := verifyTronKeyPair(privateKeyHex, targetAddress)
	fmt.Printf("地址与私钥是否匹配: %v\n", match)

	// 测试十六进制地址转换为Base58格式
	address := "TDqSquXBgUCLYvYC4XZgrprLK589dkhSCf"
	hexAddress := "412a68baf67f1c497d9a4a609276a90dcd6ea77444"
	// hexAddress := "41f340cc053fbaf0d636fc286e9b64ad2150ecb8d7"
	base58Address, err := hexAddressToBase58(hexAddress)
	if err != nil {
		log.Fatalf("地址转换失败: %v", err)
	}
	fmt.Printf("十六进制地址: %s\n", hexAddress)
	fmt.Printf("Base58地址: %s\n", base58Address)
	if base58Address == address {
		fmt.Println("地址转换成功")
	} else {
		fmt.Println("地址转换失败")
	}
}

// 验证私钥与地址是否匹配
func verifyTronKeyPair(privateKeyHex, targetAddress string) bool {
	generatedAddress, err := getAddressFromPrivateKey(privateKeyHex)
	if err != nil {
		log.Fatal(err)
	}
	return generatedAddress == targetAddress
}

func getAddressFromPrivateKey(privateKeyHex string) (string, error) {
	// 1. 解码私钥
	privateKeyBytes, err := hex.DecodeString(privateKeyHex)
	if err != nil {
		return "", fmt.Errorf("解码私钥失败: %v", err)
	}

	// 2. 从私钥生成公钥
	privateKey, _ := btcec.PrivKeyFromBytes(privateKeyBytes)
	publicKey := privateKey.PubKey()
	publicKeyBytes := publicKey.SerializeUncompressed()

	// 3. 只保留X和Y坐标，去掉前缀0x04
	publicKeyBytes = publicKeyBytes[1:]

	// 4. 对公钥进行Keccak-256哈希
	publicKeyHash := crypto.Keccak256(publicKeyBytes)

	// 5. 只保留哈希的最后20字节作为地址
	address := publicKeyHash[len(publicKeyHash)-20:]

	// 6. 添加前缀0x41（波场地址前缀）
	addressWithPrefix := append([]byte{0x41}, address...)

	// 7. 计算校验和（两次SHA-256哈希的前4字节）
	firstSHA := sha256.Sum256(addressWithPrefix)
	secondSHA := sha256.Sum256(firstSHA[:])
	checksum := secondSHA[:4]

	// 8. 将地址和校验和拼接
	addressWithChecksum := append(addressWithPrefix, checksum...)

	// 9. Base58编码得到最终地址
	tronAddress := base58.Encode(addressWithChecksum, base58.BitcoinAlphabet)

	return tronAddress, nil
}

// 将十六进制格式的波场地址转换为Base58格式
func hexAddressToBase58(hexAddress string) (string, error) {
	// 1. 解码十六进制地址
	addressBytes, err := hex.DecodeString(hexAddress)
	if err != nil {
		return "", fmt.Errorf("解码地址失败: %v", err)
	}

	// 2. 计算校验和（两次SHA-256哈希的前4字节）
	firstSHA := sha256.Sum256(addressBytes)
	secondSHA := sha256.Sum256(firstSHA[:])
	checksum := secondSHA[:4]

	// 3. 将地址和校验和拼接
	addressWithChecksum := append(addressBytes, checksum...)

	// 4. Base58编码得到最终地址
	base58Address := base58.Encode(addressWithChecksum, base58.BitcoinAlphabet)

	return base58Address, nil
}
