package main

import (
	"crypto/aes"
	"crypto/hmac"
	"crypto/md5"
	"crypto/rand"
	"fmt"
	"io"
	mrand "math/rand"
	"time"

	"github.com/peteheist/irtt"
	"golang.org/x/crypto/nacl/box"
)

// so copy builtin can be used for fast zero-ing
const zeroesLen = 64 * 1024

var zeroes = make([]byte, zeroesLen)

func zero(b []byte) {
	if len(b) > len(zeroes) {
		zeroes = make([]byte, len(b)*2)
	}
	copy(b, zeroes)
}

func zeroLoop(b []byte) {
	for i := 0; i < len(b); i++ {
		b[i] = 0
	}
}

func testNacl() {
	start := time.Now()
	senderPublicKey, senderPrivateKey, err := box.GenerateKey(rand.Reader)
	if err != nil {
		panic(err)
	}
	fmt.Printf("generate sender key: %s\n", time.Since(start))

	start = time.Now()
	recipientPublicKey, recipientPrivateKey, err := box.GenerateKey(rand.Reader)
	if err != nil {
		panic(err)
	}
	fmt.Printf("generate recipient key: %s\n", time.Since(start))

	start = time.Now()
	// You must use a different nonce for each message you encrypt with the
	// same key. Since the nonce here is 192 bits long, a random value
	// provides a sufficiently small probability of repeats.
	var nonce [24]byte
	if _, err := io.ReadFull(rand.Reader, nonce[:]); err != nil {
		panic(err)
	}
	fmt.Printf("generate nonce: %s\n", time.Since(start))

	for i := 0; i < 10; i++ {
		var msg [16]byte
		if _, err := io.ReadFull(rand.Reader, msg[:]); err != nil {
			panic(err)
		}
		// This encrypts msg and appends the result to the nonce.
		start = time.Now()
		encrypted := box.Seal(nonce[:], msg[:], &nonce, recipientPublicKey, senderPrivateKey)
		fmt.Printf("seal: %s %d\n", time.Since(start), len(encrypted))

		// The recipient can decrypt the message using their private key and the
		// sender's public key. When you decrypt, you must use the same nonce you
		// used to encrypt the message. One way to achieve this is to store the
		// nonce alongside the encrypted message. Above, we stored the nonce in the
		// first 24 bytes of the encrypted text.
		start = time.Now()
		var decryptNonce [24]byte
		copy(decryptNonce[:], encrypted[:24])
		decrypted, ok := box.Open(nil, encrypted[24:], &decryptNonce, senderPublicKey, recipientPrivateKey)
		fmt.Printf("decrypt: %s\n", time.Since(start))
		if !ok {
			panic("decryption error")
		}
		fmt.Printf("%x\n", decrypted)
	}
}

func midpoint(t1 time.Time, t2 time.Time) time.Time {
	// we'll live without nanosecond rounding here
	return t1.Add(t2.Sub(t1) / 2)
}

func testHMAC() {
	iterations := 1000
	lengths := []int{28, 36, 160, 1450, 9000}

	key := make([]byte, 16)
	if _, err := io.ReadFull(rand.Reader, key[:]); err != nil {
		panic(err)
	}
	md5Hash := hmac.New(md5.New, key)
	total := time.Duration(0)
	for _, l := range lengths {
		buf := make([]byte, l)
		for i := 0; i < iterations; i++ {
			if _, err := io.ReadFull(rand.Reader, buf); err != nil {
				panic(err)
			}
			start := time.Now()
			md5Hash.Reset()
			md5Hash.Write(buf)
			md5Hash.Sum(nil)
			total += time.Since(start)
		}
		fmt.Printf("len %d, %s per iteration, %s per byte, %.0f Mbps\n", l,
			time.Duration(total/time.Duration(iterations)),
			time.Duration(total/time.Duration(l*iterations)),
			8000.0*float64(l)*float64(iterations)/float64(total))
	}
}

func midpointTest() {
	start := time.Now()
	time.Sleep(100 * time.Millisecond)
	end := time.Now()
	mid := midpoint(start, end)
	fmt.Printf("start %s\n", start)
	fmt.Printf("mid %s\n", mid)
	fmt.Printf("end %s\n", end)
}

func clockTest() {
	start := time.Now()
	for {
		now := time.Now()
		sinceStartMono := now.Sub(start)
		sinceStartWall := now.Round(0).Sub(start)
		wallMonoDiff := time.Duration(sinceStartWall - sinceStartMono)
		driftPerSecond := time.Duration(float64(wallMonoDiff) * float64(1000000000) /
			float64(sinceStartMono))
		driftPerMinute := time.Duration(float64(wallMonoDiff) * float64(64000000000) /
			float64(sinceStartMono))
		fmt.Printf("\n")
		fmt.Printf("since start mono: %s\n", sinceStartMono)
		fmt.Printf("since start wall: %s\n", sinceStartWall)
		fmt.Printf("       wall-mono: %s\n", wallMonoDiff)
		fmt.Printf("drift per second: %s\n", driftPerSecond)
		fmt.Printf("drift per minute: %s\n", driftPerMinute)
		time.Sleep(1 * time.Second)
	}
}

func copyTest() {
	iterations := 100000
	length := 1500
	dst := make([]byte, length)
	src := make([]byte, length)
	if _, err := io.ReadFull(rand.Reader, src); err != nil {
		panic(err)
	}
	start := time.Now()
	for i := 0; i < iterations; i++ {
		copy(dst[0:], src)
	}
	elapsed := time.Since(start)
	fmt.Printf("copy %s, %.0f\n", elapsed, 8000.0*float64(iterations)*float64(length)/float64(elapsed))

	start = time.Now()
	for i := 0; i < iterations; i++ {
		for j := 0; j < length; j++ {
			dst[j] = src[j]
		}
	}
	elapsed = time.Since(start)
	fmt.Printf("loop %s, %.0f\n", elapsed, 8000.0*float64(iterations)*float64(length)/float64(elapsed))

	start = time.Now()
	for i := 0; i < iterations; i++ {
		zeroLoop(dst)
	}
	elapsed = time.Since(start)
	fmt.Printf("zero loop %s, %.0f\n", elapsed, 8000.0*float64(iterations)*float64(length)/float64(elapsed))

	start = time.Now()
	for i := 0; i < iterations; i++ {
		zero(dst)
	}
	elapsed = time.Since(start)
	fmt.Printf("zero copy %s, %.0f\n", elapsed, 8000.0*float64(iterations)*float64(length)/float64(elapsed))
}

func ifaceTest() {
	x := []int{1, 2, 3}
	fmt.Println(x)
}

func testPattern() {
	iterations := 1000
	patlen := 2
	length := 1484
	pattern := make([]byte, patlen)
	for i := 0; i < patlen; i++ {
		pattern[i] = byte(i)
	}
	bp := irtt.NewPatternFiller(pattern)
	b := make([]byte, length)
	start := time.Now()
	for i := 0; i < iterations; i++ {
		bp.Read(b)
	}
	elapsed := time.Since(start)
	fmt.Printf("pattern %s, %.0f\n", elapsed, 8000.0*float64(iterations)*float64(length)/float64(elapsed))
}

func testRand() {
	iterations := 1000
	length := 1472
	r := mrand.New(mrand.NewSource(time.Now().UnixNano()))
	b := make([]byte, length)
	start := time.Now()
	for i := 0; i < iterations; i++ {
		r.Read(b)
	}
	elapsed := time.Since(start)
	fmt.Printf("rand %s, %.0f\n", elapsed, 8000.0*float64(iterations)*float64(length)/float64(elapsed))
}

type filter interface {
	acceptInt(i int) bool

	acceptString(s string) bool
}

type myFilter struct {
}

func (f *myFilter) acceptInt(i int) bool {
	return (i & 0x10) != 0
}

func (f *myFilter) acceptString(s string) bool {
	//return strings.HasPrefix(s, "Drop-")
	return (s[0:5] == "Drop-")
}

func testStringVsIntCompare() {
	iterations := 1
	code := 0x10
	code++
	s := "Drop-BadMagic"
	f := &myFilter{}
	start := time.Now()
	for i := 0; i < iterations; i++ {
		_ = f.acceptString(s)
	}
	elapsed := time.Since(start)
	fmt.Printf("string %s, %.0f ns/op\n", elapsed, float64(elapsed)/float64(iterations))
	start = time.Now()
	for i := 0; i < iterations; i++ {
		_ = f.acceptInt(i)
	}
	elapsed = time.Since(start)
	fmt.Printf("int %s, %.0f ns/op\n", elapsed, float64(elapsed)/float64(iterations))
}

func testAES() {
	iterations := 1000
	block := 32 * 1024
	key := make([]byte, 16)
	_, err := rand.Read(key)
	if err != nil {
		panic(err)
	}
	bs, err := aes.NewCipher(key)
	if err != nil {
		panic(err)
	}
	data := make([]byte, block)
	_, err = rand.Read(data)
	if err != nil {
		panic(err)
	}
	start := time.Now()
	for i := 0; i < iterations; i++ {
		for j := 0; j < block; j += 32 {
			bs.Encrypt(data[j:j+32], data[j:j+32])
		}
	}
	elapsed := time.Since(start)
	fmt.Printf("aes %s, %.0f ns/op\n", elapsed, float64(elapsed)/float64(iterations*1024))
}

func main() {
	testPattern()
}
