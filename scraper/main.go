package main

import (
	"bufio"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"regexp"
	"runtime"
	"sync"
	"time"
	"github.com/wcharczuk/go-chart/v2"
)

const (
	channel = "yourragegaming"
)

var (
	maxWorkers = runtime.NumCPU()
	totalSubs  int
	primeSubs  int
	tier1Subs  int
	tier2Subs  int
	tier3Subs  int
	mu         sync.Mutex
)

func main() {
	scrapeFlag := flag.Bool("scrape", false, "Scrape logs from server")
	readFlag := flag.Bool("read", false, "Read logs from files")
	flag.Parse()

	now := time.Now()
	year := now.Year()
	month := int(now.Month())
	today := now.Day()

	subRegex := regexp.MustCompile(`(?i)subscribed (with|at) (Prime|Tier \d)|subscribed for \d+ months`)
	primeRegex := regexp.MustCompile(`(?i)subscribed (with|at) Prime`)
	tier1Regex := regexp.MustCompile(`(?i)subscribed (with|at) Tier 1`)
	tier2Regex := regexp.MustCompile(`(?i)subscribed (with|at) Tier 2`)
	tier3Regex := regexp.MustCompile(`(?i)subscribed (with|at) Tier 3`)


	os.Mkdir("./logs", 0755)

	if *scrapeFlag {
		scrapeLogs(year, month, today, subRegex, primeRegex, tier1Regex, tier2Regex, tier3Regex)
	} else if *readFlag {
		readLogs(year, month, today, subRegex, primeRegex, tier1Regex, tier2Regex, tier3Regex)
	} else {
		if logsExist(year, month, today) {
			readLogs(year, month, today, subRegex, primeRegex, tier1Regex, tier2Regex, tier3Regex)
		} else {
			scrapeLogs(year, month, today, subRegex, primeRegex, tier1Regex, tier2Regex, tier3Regex)
			readLogs(year, month, today, subRegex, primeRegex, tier1Regex, tier2Regex, tier3Regex)
		}
	}

	log.Println("✅ Done processing logs for the month.")
	log.Printf("📊 Total Subscriptions: %d\n", totalSubs)
	log.Printf("   🔹 Prime Subs: %d\n", primeSubs)
	log.Printf("   🔸 Tier 1 Subs: %d\n", tier1Subs)
	log.Printf("   🟠 Tier 2 Subs: %d\n", tier2Subs)
	log.Printf("   🔴 Tier 3 Subs: %d\n", tier3Subs)
	generateBarChart(primeSubs, tier1Subs, tier2Subs, tier3Subs)
}

func logsExist(year, month, today int) bool {
	for day := 1; day <= today; day++ {
		logFileName := fmt.Sprintf("./logs/subs_%d_%02d_%02d.log", year, month, day)
		if _, err := os.Stat(logFileName); err == nil {
			return true
		}
	}
	return false
}

func scrapeLogs(year, month, today int, subRegex, primeRegex, tier1Regex, tier2Regex, tier3Regex *regexp.Regexp) {
	days := make(chan int, today)
	var wg sync.WaitGroup

	for i := range maxWorkers {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			for day := range days {
				processDay(workerID, year, month, day, subRegex, primeRegex, tier1Regex, tier2Regex, tier3Regex)
			}
		}(i)
	}

	for day := 1; day <= today; day++ {
		days <- day
	}
	close(days)

	wg.Wait()
}

func readLogs(year, month, today int, subRegex, primeRegex, tier1Regex, tier2Regex, tier3Regex *regexp.Regexp) {
	for day := 1; day <= today; day++ {
		logFileName := fmt.Sprintf("./logs/subs_%d_%02d_%02d.log", year, month, day)
		file, err := os.Open(logFileName)
		if err != nil {
			log.Printf("No log file for %s, skipping reading", logFileName)
			continue
		}

		subs, prime, tier1, tier2, tier3 := 0, 0, 0, 0, 0
		scanner := bufio.NewScanner(file)
		for scanner.Scan() {
			line := scanner.Text()
			if subRegex.MatchString(line) {
				subs++
				if primeRegex.MatchString(line) {
					prime++
				} else if tier1Regex.MatchString(line) {
					tier1++
				} else if tier2Regex.MatchString(line) {
					fmt.Println(line)
					tier2++
				} else if tier3Regex.MatchString(line) {
					fmt.Println(line)
					tier3++
				}
			}
		}
		file.Close()

		mu.Lock()
		totalSubs += subs
		primeSubs += prime
		tier1Subs += tier1
		tier2Subs += tier2
		tier3Subs += tier3
		mu.Unlock()

		log.Printf("[Read] Processed %s — Subs found: %d (Prime: %d, T1: %d, T2: %d, T3: %d)",
			logFileName, subs, prime, tier1, tier2, tier3)
	}
}

func processDay(workerID, year, month, day int, subRegex, primeRegex, tier1Regex, tier2Regex, tier3Regex *regexp.Regexp) {
	date := time.Date(year, time.Month(month), day, 0, 0, 0, 0, time.UTC)
	dateStr := date.Format("2006-01-02")
	logFileName := fmt.Sprintf("./logs/subs_%d_%02d_%02d.log", year, month, day)

	if _, err := os.Stat(logFileName); err == nil {
		log.Printf("[%02d] Skipping %s — already logged", workerID, dateStr)
		return
	}

	url := fmt.Sprintf("https://logs.potat.app/channel/%s/%d/%d/%d", channel, year, month, day)
	log.Printf("[%02d] Fetching logs for %s → %s", workerID, dateStr, url)

	resp, err := http.Get(url)
	if err != nil {
		log.Printf("[%02d] Error fetching %s: %v", workerID, dateStr, err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		log.Printf("[%02d] No logs found for %s (%s)", workerID, dateStr, resp.Status)
		return
	}

	file, err := os.OpenFile(logFileName, os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		log.Printf("[%02d] Failed to open log file for %s: %v", workerID, dateStr, err)
		return
	}
	defer file.Close()
	writer := bufio.NewWriter(file)

	scanner := bufio.NewScanner(resp.Body)
	subs := 0
	prime := 0
	tier1 := 0
	tier2 := 0
	tier3 := 0

	for scanner.Scan() {
		line := scanner.Text()
		if subRegex.MatchString(line) {
			subs++
			logLine := fmt.Sprintf("[%s] %s\n", dateStr, line)
			writer.WriteString(logLine)

			if primeRegex.MatchString(line) {
				prime++
			} else if tier1Regex.MatchString(line) {
				tier1++
			} else if tier2Regex.MatchString(line) {
				tier2++
			} else if tier3Regex.MatchString(line) {
				tier3++
			}
		}
	}
	writer.Flush()

	if err := scanner.Err(); err != nil {
		log.Printf("[%02d] Scanner error on %s: %v", workerID, dateStr, err)
	}

	mu.Lock()
	totalSubs += subs
	primeSubs += prime
	tier1Subs += tier1
	tier2Subs += tier2
	tier3Subs += tier3
	mu.Unlock()

	log.Printf("[%02d] Finished %s — Subs found: %d (Prime: %d, T1: %d, T2: %d, T3: %d)",
		workerID, dateStr, subs, prime, tier1, tier2, tier3)
}

func generateBarChart(prime, t1, t2, t3 int) {
	graph := chart.BarChart{
		Title: "Subscription Breakdown",
		Height: 512,
		BarWidth: 60,
		Bars: []chart.Value{
			{Value: float64(prime), Label: "Prime"},
			{Value: float64(t1), Label: "Tier 1"},
			{Value: float64(t2), Label: "Tier 2"},
			{Value: float64(t3), Label: "Tier 3"},
		},
	}

	f, err := os.Create("sub_chart.png")
	if err != nil {
		log.Fatalf("Error creating chart file: %v", err)
	}
	defer f.Close()

	err = graph.Render(chart.PNG, f)
	if err != nil {
		log.Fatalf("Error rendering chart: %v", err)
	}
	log.Println("📈 Chart saved as sub_chart.png")
}