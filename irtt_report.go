package irtt

import (
	"io/ioutil"
	"encoding/json"
)

func reportUsage() {
	setBufio()
	printf("Usage: report output.json")
	printf("")
}
	
func runReport(argv []string) {
	if (len(argv) != 1) {
		usageAndExit(reportUsage, exitCodeBadCommandLine)
	}

	dat, err := ioutil.ReadFile(argv[0])
	if err != nil {
		panic(err)
	}

	var r *Result
	json.Unmarshal([]byte(dat), &r)
	r.SendRate = calculateBitrate(r.Stats.BytesSent,
		r.Stats.LastSent.Sub(r.Stats.FirstSend))
	r.ReceiveRate = calculateBitrate(r.Stats.BytesReceived,
		r.Stats.LastReceived.Sub(r.Stats.FirstReceived))
	r.ExpectedPacketsSent = pcount(r.Stats.LastSent.Sub(r.Stats.FirstSend),
		r.Config.Interval)
	printResult(r)
}
