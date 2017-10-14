package irtt

import (
	"crypto/hmac"
	"crypto/md5"
	"crypto/rand"
	"io"
	mrand "math/rand"
	"time"
)

func runBenchBufTest(fn func([]byte)) {
	lengths := []int{16, 32, 64, 172, 1472, 8972}
	printf("")
	for _, l := range lengths {
		buf := make([]byte, l)
		end := time.Now().Add(1 * time.Second)
		i := 0
		elapsed := time.Duration(0)
		for time.Now().Before(end) {
			if _, err := io.ReadFull(rand.Reader, buf); err != nil {
				panic(err)
			}
			start := time.Now()
			fn(buf)
			elapsed += time.Since(start)
			i++
		}
		printf("len %d, %d iterations, %.0f ns/op, %.0f Mbps", l,
			i, float64(elapsed)/float64(i),
			8000.0*float64(l)*float64(i)/float64(elapsed))
	}
}

func testHMAC() {
	printf("Testing HMAC...")
	key := make([]byte, 16)
	if _, err := io.ReadFull(rand.Reader, key[:]); err != nil {
		panic(err)
	}
	md5Hash := hmac.New(md5.New, key)
	runBenchBufTest(func(b []byte) {
		md5Hash.Reset()
		md5Hash.Write(b)
		md5Hash.Sum(nil)
	})
}

func testPatternFill() {
	printf("Testing pattern fill...")
	patlen := 4
	pattern := make([]byte, patlen)
	if _, err := io.ReadFull(rand.Reader, pattern[:]); err != nil {
		panic(err)
	}
	bp := NewPatternFiller(pattern)
	runBenchBufTest(func(b []byte) {
		bp.Read(b)
	})
}

func testRandFill() {
	printf("Testing random fill...")
	r := mrand.New(mrand.NewSource(time.Now().UnixNano()))
	runBenchBufTest(func(b []byte) {
		r.Read(b)
	})
}

func runBench(args []string) {
	testHMAC()
	printf("")
	testPatternFill()
	printf("")
	testRandFill()
}
