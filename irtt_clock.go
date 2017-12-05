package irtt

import (
	"time"
)

func runClock(args []string) {
	printf("Testing wall vs monotonic clocks...")
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
}
