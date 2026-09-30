package store

// crypto_fuzz_test.go —— S3C2 / S3C3 加密信封解析的原生 fuzz（stdlib `testing.F`，无新依赖）。
//
// 为什么在这里：`parseEnvelope` 直接消费磁盘上的**不可信字节**（accounts.json / accounts.json.enc），
// 历史上这里修过两类问题——坏 magic 静默降级、S3C3 的 KDF 参数被篡改成「读放大 DoS」
// （见 crypto.go 上界注释）。既有无测试是手写用例，只覆盖已知输入；本文件对输入空间做探索。
//
// 断言的性质（property）：
//  1. 任意字节都不 panic；
//  2. 失败必须 **fail-closed**——三个返回值同时为零值/空，绝不返回部分解析结果；
//  3. 成功时：magic 受支持、盐恰为 `encSaltLen` 字节、KDF 参数非零且不越上界、
//     密文恰为数据尾部切片（无重叠/无截断）。
//
// 运行（有界；CI 见 .github/workflows/fuzz.yml）：
//
//	cd apps/server && go test ./internal/store/ -run '^$' -fuzz '^FuzzParseEnvelope$' -fuzztime=10s
//	cd apps/server && go test ./internal/store/ -run '^$' -fuzz '^FuzzEnvelopeRoundTrip$' -fuzztime=10s

import (
	"bytes"
	"testing"
)

// FuzzParseEnvelope 对任意磁盘字节验证 S3C2/S3C3 信封解析的 fail-closed 性质。
func FuzzParseEnvelope(f *testing.F) {
	// 种子语料：两类合法信封 + 全部已知的攻击/边界形态。
	f.Add(envelope(randomSalt(), []byte("ciphertext")))
	f.Add(append(append([]byte{}, encMagicV2...), append(randomSalt(), []byte("ct")...)...))
	f.Add([]byte{})
	f.Add([]byte("S3C"))
	f.Add([]byte("S3C2"))
	f.Add(append([]byte("S3C2"), randomSalt()...)) // S3C2 无密文段
	f.Add([]byte("XXXX" + string(make([]byte, 64))))
	// S3C3 头：零 KDF 参数（弱密钥）与全 0xFF 参数（读放大 DoS）都必须被拒。
	f.Add([]byte("S3C3\x00\x00\x00\x00\x00\x00\x00\x00\x00" + string(randomSalt())))
	f.Add([]byte("S3C3\xff\xff\xff\xff\xff\xff\xff\xff\xff" + string(randomSalt())))
	// S3C3 合法参数（t=2, m=64KiB, p=4）。
	f.Add([]byte("S3C3\x00\x00\x00\x02\x00\x01\x00\x00\x04" + string(randomSalt())))

	f.Fuzz(func(t *testing.T, data []byte) {
		params, salt, ciphertext, err := parseEnvelope(data)
		if err != nil {
			if params != (kdfParams{}) || len(salt) != 0 || len(ciphertext) != 0 {
				t.Fatalf("parseEnvelope 失败却返回部分结果: params=%+v saltLen=%d ctLen=%d err=%v",
					params, len(salt), len(ciphertext), err)
			}
			return
		}

		if !isEncryptedBlob(data) {
			t.Fatalf("parseEnvelope 接受了非加密信封（magic=%q）", data[:min(4, len(data))])
		}
		if len(salt) != encSaltLen {
			t.Fatalf("盐长度 = %d，期望 %d", len(salt), encSaltLen)
		}
		if !kdfParamsValid(params) {
			t.Fatalf("parseEnvelope 接受了非法 KDF 参数（越过上界即为读放大 DoS）: %+v", params)
		}
		switch string(data[:4]) {
		case string(encMagicV2):
			if params != legacyParams {
				t.Fatalf("S3C2 参数 = %+v，期望 legacy %+v", params, legacyParams)
			}
			if !bytes.Equal(ciphertext, data[len(encMagicV2)+encSaltLen:]) {
				t.Fatal("S3C2 密文不是数据尾部切片")
			}
		case string(encMagicV3):
			const head = 4 + 4 + 4 + 1
			if !bytes.Equal(ciphertext, data[head+encSaltLen:]) {
				t.Fatal("S3C3 密文不是数据尾部切片")
			}
		default:
			t.Fatalf("parseEnvelope 接受了未知 magic %q", data[:4])
		}
	})
}

// FuzzEnvelopeRoundTrip 验证写入端 `envelope` 与读取端 `parseEnvelope` 的往返一致性。
func FuzzEnvelopeRoundTrip(f *testing.F) {
	f.Add(randomSalt(), []byte("ciphertext"))
	f.Add(make([]byte, encSaltLen), []byte{})
	f.Add(randomSalt(), []byte{})
	f.Add(randomSalt(), bytes.Repeat([]byte{0xff}, 64))

	f.Fuzz(func(t *testing.T, salt, ciphertext []byte) {
		// 真实调用方只传 randomSalt()（恰 encSaltLen 字节）；其余长度不属 envelope 契约。
		if len(salt) != encSaltLen {
			t.Skip()
		}
		blob := envelope(salt, ciphertext)
		params, gotSalt, gotCT, err := parseEnvelope(blob)
		if err != nil {
			t.Fatalf("envelope 写出的信封无法读回: %v (saltLen=%d ctLen=%d)",
				err, len(salt), len(ciphertext))
		}
		if params != currentParams {
			t.Fatalf("读回参数 = %+v，期望写入参数 %+v", params, currentParams)
		}
		if !bytes.Equal(gotSalt, salt) {
			t.Fatal("往返后盐不一致")
		}
		if !bytes.Equal(gotCT, ciphertext) {
			t.Fatal("往返后密文不一致")
		}
		if !isEncryptedBlob(blob) {
			t.Fatal("envelope 产物未通过 isEncryptedBlob 判定")
		}
	})
}
