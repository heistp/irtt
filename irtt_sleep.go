package irtt

import (
	"time"
)

func runSleep(args []string) {
	printf("Testing sleep accuracy...")
	printf("")
	durations := []time.Duration{1 * time.Nanosecond,
		10 * time.Nanosecond,
		100 * time.Nanosecond,
		1 * time.Microsecond,
		10 * time.Microsecond,
		100 * time.Microsecond,
		1 * time.Millisecond,
		10 * time.Millisecond,
		100 * time.Millisecond,
		200 * time.Millisecond,
		500 * time.Millisecond,
	}

	printf("Sleep Duration        Mean Error       %% Error")
	for _, d := range durations {
		iterations := int(2 * time.Second / d)
		if iterations < 5 {
			iterations = 5
		}
		errTotal := time.Duration(0)
		start0 := time.Now()
		i := 0
		for ; i < iterations && time.Since(start0) < 2*time.Second; i++ {
			start := time.Now()
			time.Sleep(d)
			elapsed := time.Since(start)
			errTotal += (elapsed - d)
		}
		errorNs := float64(errTotal) / float64(i)
		percentError := 100 * errorNs / float64(d)
		printf("%14s%18s%14.1f", d, time.Duration(errorNs), percentError)
	}

	/*
		start := time.Now()
		i := 0
		for {
			now := time.Now()
			sinceStartMono := now.Sub(start)
			sinceStartWall := now.Round(0).Sub(start)
			wallMonoDiff := time.Duration(sinceStartWall - sinceStartMono)
			driftPerSecond := time.Duration(float64(wallMonoDiff) * float64(1000000000) /
				float64(sinceStartMono))
			if i%10 == 0 {
				printf("")
				printf("         Monotonic              Wall   Wall-Monotonic   Wall Drift / Second\t")
			}
			printf("%18s%18s%17s%22s",
				sinceStartMono, sinceStartWall, wallMonoDiff, driftPerSecond)
			time.Sleep(1 * time.Second)
			i++
		}
	*/
}
