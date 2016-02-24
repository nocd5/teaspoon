package main

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/gizak/termui"
	"github.com/jacobsa/go-serial/serial"
	"github.com/jessevdk/go-flags"
)

type Options struct {
	// required
	PortName string `short:"p" long:"port" description:"Serial Port"`
	BaudRate uint   `short:"b" long:"baud" description:"Baud Rate"`
	// optional
	DataBits     uint   `long:"data" description:"Number of Data Bits" default:"8"`
	ParityMode   string `long:"parity" description:"Parity Mode. none | even | odd" default:"none"`
	StopBits     uint   `long:"stop" description:"Number of Stop Bits" default:"1"`
	ListComPorts bool   `short:"l" long:"list" description:"List COM Ports"`
	Delimiter    string `short:"d" long:"delimiter" description:"Delimiter for Received Data parsing" default:"\n"`
	LineMode     string `long:"mode" description:"LineChart Mode. braille | dot" default:"braille"`
}

var opts Options

func min(a int, b int) int {
	if a < b {
		return a
	} else {
		return b
	}
}

// FIXME:
// I don't know how to fit a Y-Axis scale without redraw.
// Thus, the func recreate LineChart instance every time.
func draw(data []float64) {
	chart := termui.NewLineChart()
	chart.Height = termui.TermHeight()
	chart.BorderLabel = opts.PortName
	chart.LineColor = termui.ColorRed | termui.AttrBold
	chart.Mode = opts.LineMode

	list := termui.NewList()
	list.Height = termui.TermHeight()
	list.BorderLabel = "Data"

	termui.Body.Rows = nil
	termui.Body.AddRows(
		termui.NewRow(
			termui.NewCol(10, 0, chart),
			termui.NewCol(2, 0, list),
		),
	)

	termui.Body.Align()

	if len(data) > 0 {
		iarea := (termui.Body.Rows[0].Cols[0].Width - 8)
		if chart.Mode == "braille" {
			iarea *= 2
		}
		trim := min(len(data), iarea)
		chart.Data = data[len(data)-trim:]
		for i := min(list.Height, len(data)) - 1; i >= 0; i-- {
			list.Items = append(list.Items, fmt.Sprint(data[len(data)-1-i]))
		}
	}

	termui.Render(termui.Body)
}

func main() {
	_, err := flags.Parse(&opts)
	if err != nil {
		os.Exit(1)
	}

	if opts.ListComPorts {
		listComPorts()
		os.Exit(0)
	}

	if opts.PortName == "" || opts.BaudRate == 0 {
		fmt.Fprintln(os.Stderr, "the required flags `/b, /baud' and `/p, /port' were not specified")
		os.Exit(1)
	}

	var parityMode serial.ParityMode
	switch opts.ParityMode {
	case "none":
		parityMode = serial.PARITY_NONE
	case "odd":
		parityMode = serial.PARITY_ODD
	case "even":
		parityMode = serial.PARITY_EVEN
	default:
		fmt.Fprintf(os.Stderr, "Invalid ParityMode: %s\n", opts.ParityMode)
		fmt.Fprintf(os.Stderr, "`--parity` should be any one of none/odd/even\n")
		os.Exit(1)
	}

	options := serial.OpenOptions{
		PortName:              opts.PortName,
		BaudRate:              opts.BaudRate,
		DataBits:              opts.DataBits,
		ParityMode:            parityMode,
		StopBits:              opts.StopBits,
		InterCharacterTimeout: 1000,
	}

	port, err := serial.Open(options)
	if err != nil {
		panic(err)
	}
	defer port.Close()

	if err := termui.Init(); err != nil {
		panic(err)
	}
	defer termui.Close()

	qreq := make(chan bool)
	qack := make(chan bool)

	termui.Handle("/sys/kbd/q", func(termui.Event) {
		qreq <- true
	})

	// Rx Handler
	data := make([]float64, 0)
	var strbuf string
	go func(_qreq chan bool) {
		buf := make([]byte, 128)
		for {
			select {
			case <-_qreq:
				qack <- true
				break
			default:
				n, err := port.Read(buf)
				if err != nil {
					panic(err)
				}
				if n > 0 {
					// Allocate extra data size.
					// Because data will be truncated at draw function.
					displen := termui.TermWidth()
					if opts.LineMode == "braille" {
						displen *= 2
					}
					strbuf = strbuf + string(buf[:n])
					strval := strings.Split(strbuf, opts.Delimiter)
					for i := 0; i < len(strval)-1; i++ {
						val, err := strconv.ParseFloat(strval[i], 64)
						if err == nil {
							if len(data) >= displen {
								data = data[1:]
							}
							data = append(data, val)
						}
					}
					draw(data)
					strbuf = strval[len(strval)-1]
				}
			}
		}
	}(qreq)

	go func() {
		<-qack
		termui.StopLoop()
	}()

	draw(data)
	termui.Loop()
}
