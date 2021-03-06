package main

import (
	"flag"
	"fmt"
	"github.com/trezor/trezord-go/server/api"
	"github.com/trezor/trezord-go/server/checker"
	"io"
	"log"
	"net"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"

	"github.com/trezor/trezord-go/core"
	"github.com/trezor/trezord-go/memorywriter"
	"github.com/trezor/trezord-go/server"
	"github.com/trezor/trezord-go/usb"
	"gopkg.in/natefinch/lumberjack.v2"
)

const version = "2.0.30"

type udpTouples []usb.PortTouple

func (i *udpTouples) String() string {
	res := ""
	for i, p := range *i {
		if i > 0 {
			res += ","
		}
		res += strconv.Itoa(p.Normal) + ":" + strconv.Itoa(p.Debug)
	}
	return res
}

func (i *udpTouples) Set(value string) error {
	split := strings.Split(value, ":")
	n, err := strconv.Atoi(split[0])
	if err != nil {
		return err
	}
	d, err := strconv.Atoi(split[1])
	if err != nil {
		return err
	}
	*i = append(*i, usb.PortTouple{
		Normal: n,
		Debug:  d,
	})
	return nil
}

type udpPorts []int

func (i *udpPorts) String() string {
	res := ""
	for i, p := range *i {
		if i > 0 {
			res += ","
		}
		res += strconv.Itoa(p)
	}
	return res
}

func (i *udpPorts) Set(value string) error {
	p, err := strconv.Atoi(value)
	if err != nil {
		return err
	}
	*i = append(*i, p)
	return nil
}

func initUsb(init bool, wr *memorywriter.MemoryWriter, sl *log.Logger) []core.USBBus {
	if init {
		wr.Log("Initing libusb")

		w, err := usb.InitLibUSB(wr, !usb.HIDUse, allowCancel(), detachKernelDriver())
		if err != nil {
			sl.Fatalf("libusb: %s", err)
		}

		if !usb.HIDUse {
			return []core.USBBus{w}
		}

		wr.Log("Initing hidapi")
		h, err := usb.InitHIDAPI(wr)
		if err != nil {
			sl.Fatalf("hidapi: %s", err)
		}
		return []core.USBBus{w, h}
	}
	return nil
}

func tcpListenCheck() {

	checkAddresses := []string{"127.0.0.1:21325", "0.0.0.0:21325"}

	for _, addr := range checkAddresses {
		ln, err := net.Listen("tcp", addr)
		if err != nil {
			if ln != nil {
				_ = ln.Close()
			}
			log.Println(fmt.Sprintf("%s 被监听，未正常启动，请检查", addr))
			// 防止控制台关闭
			select {}
		}
		_ = ln.Close()
	}

}

func main() {
	var nocors bool
	var domains string
	var dbpath string
	var logfile string
	var ports udpPorts
	var touples udpTouples
	var withusb bool
	var verbose bool
	var reset bool
	var versionFlag bool

	flag.BoolVar(
		&nocors,
		"nocors",
		false,
		"Disable Cors check.",
	)
	flag.StringVar(
		&domains,
		"domains",
		"",
		"Domains. Cors allow domains, split by ',' ",
	)
	flag.StringVar(
		&dbpath,
		"db",
		"",
		"Db path. Default, use 'trezord.db' in current directory",
	)
	flag.StringVar(
		&logfile,
		"l",
		"",
		"Log into a file, rotating after 20MB",
	)
	flag.Var(
		&ports,
		"e",
		"Use UDP port for emulator. Can be repeated for more ports. Example: trezord-go -e 21324 -e 21326",
	)
	flag.Var(
		&touples,
		"ed",
		"Use UDP port for emulator with debug link. Can be repeated for more ports. Example: trezord-go -ed 21324:21326",
	)
	flag.BoolVar(
		&withusb,
		"u",
		true,
		"Use USB devices. Can be disabled for testing environments. Example: trezord-go -e 21324 -u=false",
	)
	flag.BoolVar(
		&verbose,
		"v",
		false,
		"Write verbose logs to either stderr or logfile",
	)
	flag.BoolVar(
		&versionFlag,
		"version",
		false,
		"Write version",
	)
	flag.BoolVar(
		&reset,
		"r",
		true,
		"Reset USB device on session acquiring. Enabled by default (to prevent wrong device states); set to false if you plan to connect to debug link outside of bridge.",
	)
	flag.Parse()

	if versionFlag {
		fmt.Printf("trezord version %s", version)
		return
	}

	var stderrWriter io.Writer
	if logfile != "" {
		stderrWriter = &lumberjack.Logger{
			Filename:   logfile,
			MaxSize:    20, // megabytes
			MaxBackups: 3,
		}
	} else {
		stderrWriter = os.Stderr
	}

	stderrLogger := log.New(stderrWriter, "", log.LstdFlags)

	shortMemoryWriter := memorywriter.New(2000, 200, false, nil)

	verboseWriter := stderrWriter
	if !verbose {
		verboseWriter = nil
	}

	longMemoryWriter := memorywriter.New(90000, 200, true, verboseWriter)

	printWelcomeInfo(stderrLogger)
	bus := initUsb(withusb, longMemoryWriter, stderrLogger)

	longMemoryWriter.Log(fmt.Sprintf("UDP port count - %d", len(ports)))

	if len(ports)+len(touples) > 0 {
		for _, t := range ports {
			touples = append(touples, usb.PortTouple{
				Normal: t,
				Debug:  0,
			})
		}
		e, errUDP := usb.InitUDP(touples, longMemoryWriter)
		if errUDP != nil {
			panic(errUDP)
		}
		bus = append(bus, e)
	}

	if len(bus) == 0 {
		stderrLogger.Fatalf("No transports enabled")
	}

	tcpListenCheck()
	// cors domain
	api.InitCors(domains, nocors)

	// 当前程序所在目录下，建立数据库文件
	tmpDbPath := dbpath
	if tmpDbPath == "" {
		currentDir, errDir := filepath.Abs(filepath.Dir(os.Args[0]))
		if errDir != nil {
			stderrLogger.Fatalf("filepath.Abs: %s", errDir)
		}
		currentDir = strings.Replace(currentDir, "\\", "/", -1)
		tmpDbPath = filepath.Join(currentDir, "trezord.db", )
	}
	longMemoryWriter.Log("Creating sqlite3")
	checker.Init(tmpDbPath, stderrLogger)

	b := usb.Init(bus...)
	defer b.Close()
	longMemoryWriter.Log("Creating core")
	c := core.New(b, longMemoryWriter, allowCancel(), reset)
	longMemoryWriter.Log("Creating HTTP server")
	s, err := server.New(c, stderrWriter, shortMemoryWriter, longMemoryWriter, version)

	if err != nil {
		stderrLogger.Fatalf("https: %s", err)
	}

	longMemoryWriter.Log("Running HTTP server")
	err = s.Run()
	if err != nil {
		stderrLogger.Fatalf("https: %s", err)
	}

	longMemoryWriter.Log("Main ended successfully")
}

func printWelcomeInfo(stderrLogger *log.Logger) {
	stderrLogger.Printf("trezord v%s is starting.", version)
	if core.IsDebugBinary() {
		stderrLogger.Print("!! DEBUG mode enabled! Please contact Trezor support in case you did not initiate this. !!")
	}
}

// Does OS allow sync canceling via our custom libusb patches?
func allowCancel() bool {
	return runtime.GOOS != "freebsd" && runtime.GOOS != "openbsd"
}

// Does OS detach kernel driver in libusb?
func detachKernelDriver() bool {
	return runtime.GOOS == "linux"
}
