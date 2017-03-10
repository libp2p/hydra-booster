package main

import (
	"fmt"
	"strings"

	human "gx/ipfs/QmPSBJL4momYnE7DcUyk2DVhD6rH488ZmHBGLbxNdhU44K/go-humanize"
	ma "gx/ipfs/QmSWLfmj5frN9xVLMMN846dMDriy5wN5jeghUm7aTW3DAG/go-multiaddr"
	net "gx/ipfs/QmVtMT3fD7DzQNW7hdm6Xe6KPstzcggrhNpeVZ4422UpKK/go-libp2p-net"
)

const (
	QClrLine = "\033[K"
	QReset   = "\033[2J"
)

/*
Move the cursor up N lines:
  \033[<N>A
- Move the cursor down N lines:
  \033[<N>B
- Move the cursor forward N columns:
  \033[<N>C
- Move the cursor backward N columns:
  \033[<N>D
*/

const (
	Clear = 0
)

const (
	Black = 30 + iota
	Red
	Green
	Yellow
	Blue
	Magenta
	Cyan
	LightGray
)

const (
	LightBlue = 94
)

func color(color int, s string) string {
	return fmt.Sprintf("\x1b[%dm%s\x1b[0m", color, s)
}

const width = 25

func padPrint(line int, label, value string) {
	putMessage(line, fmt.Sprintf("%s%s%s", label, strings.Repeat(" ", width-len(label)), value))
}

func putMessage(line int, mes string) {
	fmt.Printf("\033[%d;0H%s%s", line, QClrLine, mes)
}

func printDataSharedLine(line int, bup int, totup int64, rateup float64) {
	pad := "            "
	a := fmt.Sprintf("%d            ", bup)[:12]
	b := (human.Bytes(uint64(totup)) + pad)[:12]
	c := (human.Bytes(uint64(rateup)) + "/s" + pad)[:12]

	padPrint(line, "", a+b+c)
}

type GooeyApp struct {
	Title      string
	DataFields []*DataLine
	Log        *Log
}

func (a *GooeyApp) Print() {
	fmt.Println(QReset)
	putMessage(1, a.Title)
	for _, dl := range a.DataFields {
		dl.Print()
	}
	a.Log.Print()
}

func (a *GooeyApp) NewDataLine(line int, label, defval string) *DataLine {
	dl := &DataLine{
		Default: defval,
		Label:   label,
		Line:    line,
	}
	a.DataFields = append(a.DataFields, dl)

	return dl
}

type DataLine struct {
	Label   string
	Line    int
	Default string
	LastVal string
}

func (dl *DataLine) SetVal(s string) {
	dl.LastVal = s
	dl.Print()
}
func (dl *DataLine) Print() {
	s := dl.Default
	if dl.LastVal != "" {
		s = dl.LastVal
	}

	padPrint(dl.Line, dl.Label, s)
}

type Log struct {
	Size      int
	StartLine int
	Messages  []string
	beg       int
	end       int
}

func NewLog(line, size int) *Log {
	return &Log{
		Size:      size,
		StartLine: line,
		Messages:  make([]string, size),
		end:       -1,
	}
}

func (l *Log) Add(m string) {
	l.end = (l.end + 1) % l.Size
	if l.Messages[l.end] != "" {
		l.beg++
	}
	l.Messages[l.end] = m
}

func (l *Log) Print() {
	for i := 0; i < l.Size; i++ {
		putMessage(l.StartLine+i, l.Messages[(l.beg+i)%l.Size])
	}
}

type LogNotifee struct {
	addMes chan<- string
}

func (ln *LogNotifee) Listen(net.Network, ma.Multiaddr)      {}
func (ln *LogNotifee) ListenClose(net.Network, ma.Multiaddr) {}
func (ln *LogNotifee) Connected(_ net.Network, c net.Conn) {
	ln.addMes <- fmt.Sprintf("New connection from %s", c.RemotePeer().Pretty())
}

func (ln *LogNotifee) Disconnected(_ net.Network, c net.Conn) {
	ln.addMes <- fmt.Sprintf("Lost connection to %s", c.RemotePeer().Pretty())
}

func (ln *LogNotifee) OpenedStream(net.Network, net.Stream) {}
func (ln *LogNotifee) ClosedStream(net.Network, net.Stream) {}
