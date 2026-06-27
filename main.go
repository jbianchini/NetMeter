package main

/*
#cgo darwin CFLAGS: -x objective-c -fobjc-arc
#cgo darwin LDFLAGS: -framework Cocoa
#import <Cocoa/Cocoa.h>

void setStatusTitle(const char *title);
void setMenuDetails(const char *details);
void runStatusApp(void);
*/
import "C"

import (
	"fmt"
	"net"
	"os/exec"
	"strconv"
	"strings"
	"sync"
	"time"
	"unsafe"
)

type Counters struct {
	Rx uint64
	Tx uint64
}

type Baseline struct {
	Iface    string
	SSID     string
	Started  time.Time
	Counters Counters
	Err      string
}

var (
	mu       sync.Mutex
	baseline Baseline
)

func main() {
	resetBaseline()
	go tickLoop()
	C.runStatusApp()
}

func tickLoop() {
	for {
		updateMenuBar()
		time.Sleep(1 * time.Second)
	}
}

func updateMenuBar() {
	stats := currentStats()
	setTitle(fmt.Sprintf("↓ %s ↑ %s", humanBytes(stats.Down), humanBytes(stats.Up)))

	details := fmt.Sprintf(
		"Interface: %s\nNetwork: %s\nConnected for: %s\nDownloaded: %s\nUploaded: %s\nCurrent speed: ↓ %s/s ↑ %s/s",
		emptyAs(stats.Iface, "—"),
		emptyAs(stats.SSID, "—"),
		humanDuration(stats.Duration),
		humanBytes(stats.Down),
		humanBytes(stats.Up),
		humanBytes(stats.DownSpeed),
		humanBytes(stats.UpSpeed),
	)
	if stats.Err != "" {
		details += "\n\nError: " + stats.Err
	}
	setDetails(details)
}

type LiveStats struct {
	Iface     string
	SSID      string
	Duration  time.Duration
	Down      uint64
	Up        uint64
	DownSpeed uint64
	UpSpeed   uint64
	Err       string
}

var lastSample struct {
	at time.Time
	c  Counters
}

func currentStats() LiveStats {
	iface, ssid, c, errText := currentNetworkState()

	mu.Lock()
	defer mu.Unlock()

	if baseline.Started.IsZero() || iface != baseline.Iface || ssid != baseline.SSID {
		baseline = Baseline{Iface: iface, SSID: ssid, Started: time.Now(), Counters: c, Err: errText}
		lastSample.at = time.Now()
		lastSample.c = c
	}

	down := safeSub(c.Rx, baseline.Counters.Rx)
	up := safeSub(c.Tx, baseline.Counters.Tx)

	var downSpeed, upSpeed uint64
	now := time.Now()
	if !lastSample.at.IsZero() {
		seconds := now.Sub(lastSample.at).Seconds()
		if seconds > 0 {
			downSpeed = uint64(float64(safeSub(c.Rx, lastSample.c.Rx)) / seconds)
			upSpeed = uint64(float64(safeSub(c.Tx, lastSample.c.Tx)) / seconds)
		}
	}
	lastSample.at = now
	lastSample.c = c

	baseline.Err = errText
	return LiveStats{
		Iface:     baseline.Iface,
		SSID:      baseline.SSID,
		Duration:  time.Since(baseline.Started),
		Down:      down,
		Up:        up,
		DownSpeed: downSpeed,
		UpSpeed:   upSpeed,
		Err:       firstNonEmpty(errText, baseline.Err),
	}
}

//export GoResetCounter
func GoResetCounter() {
	resetBaseline()
	updateMenuBar()
}

func resetBaseline() {
	iface, ssid, c, errText := currentNetworkState()
	mu.Lock()
	baseline = Baseline{Iface: iface, SSID: ssid, Started: time.Now(), Counters: c, Err: errText}
	lastSample.at = time.Now()
	lastSample.c = c
	mu.Unlock()
}

func currentNetworkState() (string, string, Counters, string) {
	iface, err := defaultInterface()
	if err != nil || iface == "" {
		return "", "", Counters{}, "Could not detect the active interface: " + errString(err)
	}
	ssid := wifiSSID(iface)
	c, err := readCounters(iface)
	if err != nil {
		return iface, ssid, Counters{}, "Could not read network counters: " + err.Error()
	}
	return iface, ssid, c, ""
}

func defaultInterface() (string, error) {
	out, err := exec.Command("route", "-n", "get", "default").Output()
	if err == nil {
		for _, line := range strings.Split(string(out), "\n") {
			line = strings.TrimSpace(line)
			if strings.HasPrefix(line, "interface:") {
				return strings.TrimSpace(strings.TrimPrefix(line, "interface:")), nil
			}
		}
	}
	ifs, ifErr := net.Interfaces()
	if ifErr != nil {
		return "", ifErr
	}
	for _, i := range ifs {
		if i.Flags&net.FlagUp != 0 && i.Flags&net.FlagLoopback == 0 {
			return i.Name, nil
		}
	}
	return "", err
}

func wifiSSID(iface string) string {
	out, err := exec.Command("networksetup", "-getairportnetwork", iface).CombinedOutput()
	if err != nil {
		return ""
	}
	s := strings.TrimSpace(string(out))
	if strings.Contains(s, ":") {
		return strings.TrimSpace(strings.SplitN(s, ":", 2)[1])
	}
	return ""
}

func readCounters(iface string) (Counters, error) {
	out, err := exec.Command("netstat", "-ibn").Output()
	if err != nil {
		return Counters{}, err
	}
	var fallback []string
	for _, line := range strings.Split(string(out), "\n") {
		f := strings.Fields(line)
		if len(f) < 10 || f[0] != iface {
			continue
		}
		if len(fallback) == 0 {
			fallback = f
		}
		if strings.HasPrefix(f[2], "<Link#") {
			return parseCounterFields(f)
		}
	}
	if len(fallback) > 0 {
		return parseCounterFields(fallback)
	}
	return Counters{}, fmt.Errorf("no counters found for %s", iface)
}

func parseCounterFields(f []string) (Counters, error) {
	rx, err := strconv.ParseUint(f[6], 10, 64)
	if err != nil {
		return Counters{}, fmt.Errorf("invalid Ibytes %q", f[6])
	}
	tx, err := strconv.ParseUint(f[9], 10, 64)
	if err != nil {
		return Counters{}, fmt.Errorf("invalid Obytes %q", f[9])
	}
	return Counters{Rx: rx, Tx: tx}, nil
}

func setTitle(s string) {
	cs := C.CString(s)
	defer C.free(unsafe.Pointer(cs))
	C.setStatusTitle(cs)
}

func setDetails(s string) {
	cs := C.CString(s)
	defer C.free(unsafe.Pointer(cs))
	C.setMenuDetails(cs)
}

func humanBytes(b uint64) string {
	const unit = 1024.0
	v := float64(b)
	if v < unit {
		return fmt.Sprintf("%d B", b)
	}
	for _, suffix := range []string{"KB", "MB", "GB", "TB"} {
		v /= unit
		if v < unit {
			if v >= 100 {
				return fmt.Sprintf("%.0f %s", v, suffix)
			}
			if v >= 10 {
				return fmt.Sprintf("%.1f %s", v, suffix)
			}
			return fmt.Sprintf("%.2f %s", v, suffix)
		}
	}
	return fmt.Sprintf("%.2f PB", v/unit)
}

func humanDuration(d time.Duration) string {
	d = d.Round(time.Second)
	h := int(d.Hours())
	m := int(d.Minutes()) % 60
	s := int(d.Seconds()) % 60
	if h > 0 {
		return fmt.Sprintf("%dh %dm", h, m)
	}
	if m > 0 {
		return fmt.Sprintf("%dm %ds", m, s)
	}
	return fmt.Sprintf("%ds", s)
}

func safeSub(a, b uint64) uint64 {
	if a < b {
		return 0
	}
	return a - b
}

func emptyAs(s, alt string) string {
	if strings.TrimSpace(s) == "" {
		return alt
	}
	return s
}

func firstNonEmpty(a, b string) string {
	if a != "" {
		return a
	}
	return b
}

func errString(err error) string {
	if err == nil {
		return "unknown error"
	}
	return err.Error()
}
