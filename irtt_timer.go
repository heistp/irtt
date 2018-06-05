package irtt

import (
	"math"
	"runtime"
	"runtime/debug"
	"time"
)

const minHistBins = 25

const maxHistBins = 10

func runTimer(args []string) {
	debug.SetGCPercent(-1)
	runtime.GC()
	printf("Testing timer characteristics...")

	min := time.Duration(math.MaxInt64)
	max := time.Duration(math.MinInt64)
	start := time.Now()
	last := time.Now()
	for {
		d := time.Since(last)
		if d > 0 && d < min {
			min = d
		}
		if d > max {
			max = d
		}
		if time.Since(start) > 1*time.Second {
			break
		}
		last = time.Now()
	}

	zeroes := 0
	hmin := make([]int, minHistBins)
	hmax := make([]int, maxHistBins)
	outliers := 0
	start = time.Now()
	last = time.Now()
	n := 0
	for {
		d := time.Since(last)
		if d == 0 {
			zeroes++
		}
		if d > 0 && d < min*minHistBins {
			hmin[d/min]++
		}
		if d > 0 && d < max {
			hmax[d/(max/maxHistBins)]++
		}
		if d >= max {
			outliers++
		}
		if time.Since(start) > 1*time.Second {
			break
		}
		n++
		last = time.Now()
	}
	printf("")
	printf("Histogram relative to minimum nonzero duration (low high count):")
	setTabWriter(0)
	for i := int64(0); i < minHistBins; i++ {
		l := i * min.Nanoseconds()
		if l == 0 {
			l = 1
		}
		h := (i+1)*min.Nanoseconds() - 1
		printf("%s\t%s\t%d", time.Duration(l), time.Duration(h), hmin[i])
	}
	printf("")
	printf("Histogram relative to maximum duration (low high count):")
	setTabWriter(0)
	for i := int64(0); i < maxHistBins; i++ {
		l := int64(float64(i) * (float64(max.Nanoseconds()) / float64(maxHistBins)))
		if l == 0 {
			l = 1
		}
		h := int64((float64(i)+1)*(float64(max.Nanoseconds())/float64(maxHistBins))) - int64(1)
		printf("%s\t%s\t%d", time.Duration(l), time.Duration(h), hmax[i])
	}
	printf("")
	setTabWriter(0)
	printf("Statistics:")
	printf("Minimum nonzero duration\t%s", min)
	printf("Maximum duration\t%s", max)
	printf("Zeroes\t%d / %d", zeroes, n)
	printf("Outliers\t%d / %d", outliers, n)
	flush()
}
