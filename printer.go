// Copyright 2013 The Go Authors.  All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Windows printing.
package printer

import (
	"encoding/base64"
	"fmt"
	"io/ioutil"
	"log"
	"strconv"
	"strings"
	"syscall"
	"time"
	"unsafe"

	"golang.org/x/sys/windows"
)

//go:generate go run mksyscall_windows.go -output zapi.go printer.go

type DOC_INFO_1 struct {
	DocName    *uint16
	OutputFile *uint16
	Datatype   *uint16
}

type PRINTER_INFO_5 struct {
	PrinterName              *uint16
	PortName                 *uint16
	Attributes               uint32
	DeviceNotSelectedTimeout uint32
	TransmissionRetryTimeout uint32
}

type DRIVER_INFO_8 struct {
	Version                  uint32
	Name                     *uint16
	Environment              *uint16
	DriverPath               *uint16
	DataFile                 *uint16
	ConfigFile               *uint16
	HelpFile                 *uint16
	DependentFiles           *uint16
	MonitorName              *uint16
	DefaultDataType          *uint16
	PreviousNames            *uint16
	DriverDate               syscall.Filetime
	DriverVersion            uint64
	MfgName                  *uint16
	OEMUrl                   *uint16
	HardwareID               *uint16
	Provider                 *uint16
	PrintProcessor           *uint16
	VendorSetup              *uint16
	ColorProfiles            *uint16
	InfPath                  *uint16
	PrinterDriverAttributes  uint32
	CoreDriverDependencies   *uint16
	MinInboxDriverVerDate    syscall.Filetime
	MinInboxDriverVerVersion uint32
}

type JOB_INFO_1 struct {
	JobID        uint32
	PrinterName  *uint16
	MachineName  *uint16
	UserName     *uint16
	Document     *uint16
	DataType     *uint16
	Status       *uint16
	StatusCode   uint32
	Priority     uint32
	Position     uint32
	TotalPages   uint32
	PagesPrinted uint32
	Submitted    syscall.Systemtime
}

const (
	PRINTER_ENUM_LOCAL       = 2
	PRINTER_ENUM_CONNECTIONS = 4

	PRINTER_DRIVER_XPS = 0x00000002
)

const (
	JOB_STATUS_PAUSED                  = 0x00000001 // Job is paused
	JOB_STATUS_ERROR                   = 0x00000002 // An error is associated with the job
	JOB_STATUS_DELETING                = 0x00000004 // Job is being deleted
	JOB_STATUS_SPOOLING                = 0x00000008 // Job is spooling
	JOB_STATUS_PRINTING                = 0x00000010 // Job is printing
	JOB_STATUS_OFFLINE                 = 0x00000020 // Printer is offline
	JOB_STATUS_PAPEROUT                = 0x00000040 // Printer is out of paper
	JOB_STATUS_PRINTED                 = 0x00000080 // Job has printed
	JOB_STATUS_DELETED                 = 0x00000100 // Job has been deleted
	JOB_STATUS_BLOCKED_DEVQ            = 0x00000200 // Printer driver cannot print the job
	JOB_STATUS_USER_INTERVENTION       = 0x00000400 // User action required
	JOB_STATUS_RESTART                 = 0x00000800 // Job has been restarted
	JOB_STATUS_COMPLETE                = 0x00001000 // Job has been delivered to the printer
	JOB_STATUS_RETAINED                = 0x00002000 // Job has been retained in the print queue
	JOB_STATUS_RENDERING_LOCALLY       = 0x00004000 // Job rendering locally on the client
	esc                          byte  = 0x1B
	gs                           byte  = 0x1D
	fs                           byte  = 0x1C
	QRCodeErrorCorrectionLevelL  uint8 = 48
	QRCodeErrorCorrectionLevelM  uint8 = 49
	QRCodeErrorCorrectionLevelQ  uint8 = 50
	QRCodeErrorCorrectionLevelH  uint8 = 51
)

//sys	GetDefaultPrinter(buf *uint16, bufN *uint32) (err error) = winspool.GetDefaultPrinterW
//sys	ClosePrinter(h syscall.Handle) (err error) = winspool.ClosePrinter
//sys	OpenPrinter(name *uint16, h *syscall.Handle, defaults uintptr) (err error) = winspool.OpenPrinterW
//sys	StartDocPrinter(h syscall.Handle, level uint32, docinfo *DOC_INFO_1) (err error) = winspool.StartDocPrinterW
//sys	EndDocPrinter(h syscall.Handle) (err error) = winspool.EndDocPrinter
//sys	WritePrinter(h syscall.Handle, buf *byte, bufN uint32, written *uint32) (err error) = winspool.WritePrinter
//sys	StartPagePrinter(h syscall.Handle) (err error) = winspool.StartPagePrinter
//sys	EndPagePrinter(h syscall.Handle) (err error) = winspool.EndPagePrinter
//sys	EnumPrinters(flags uint32, name *uint16, level uint32, buf *byte, bufN uint32, needed *uint32, returned *uint32) (err error) = winspool.EnumPrintersW
//sys	GetPrinterDriver(h syscall.Handle, env *uint16, level uint32, di *byte, n uint32, needed *uint32) (err error) = winspool.GetPrinterDriverW
//sys	EnumJobs(h syscall.Handle, firstJob uint32, noJobs uint32, level uint32, buf *byte, bufN uint32, bytesNeeded *uint32, jobsReturned *uint32) (err error) = winspool.EnumJobsW

func Default() (string, error) {
	b := make([]uint16, 3)
	n := uint32(len(b))
	err := GetDefaultPrinter(&b[0], &n)
	if err != nil {
		if err != syscall.ERROR_INSUFFICIENT_BUFFER {
			return "", err
		}
		b = make([]uint16, n)
		err = GetDefaultPrinter(&b[0], &n)
		if err != nil {
			return "", err
		}
	}
	return syscall.UTF16ToString(b), nil
}

// ReadNames return printer names on the system
func ReadNames() ([]string, error) {
	const flags = PRINTER_ENUM_LOCAL | PRINTER_ENUM_CONNECTIONS
	var needed, returned uint32
	buf := make([]byte, 1)
	err := EnumPrinters(flags, nil, 5, &buf[0], uint32(len(buf)), &needed, &returned)
	if err != nil {
		if err != syscall.ERROR_INSUFFICIENT_BUFFER {
			return nil, err
		}
		buf = make([]byte, needed)
		err = EnumPrinters(flags, nil, 5, &buf[0], uint32(len(buf)), &needed, &returned)
		if err != nil {
			return nil, err
		}
	}
	ps := (*[1024]PRINTER_INFO_5)(unsafe.Pointer(&buf[0]))[:returned:returned]
	names := make([]string, 0, returned)
	for _, p := range ps {
		names = append(names, windows.UTF16PtrToString(p.PrinterName))
	}
	return names, nil
}

func Open(name string) (*Printer, error) {
	var p Printer
	// TODO: implement pDefault parameter
	err := OpenPrinter(&(syscall.StringToUTF16(name))[0], &p.h, 0)
	if err != nil {
		return nil, err
	}
	return &p, nil
}

// DriverInfo stores information about printer driver.
type DriverInfo struct {
	Name        string
	Environment string
	DriverPath  string
	Attributes  uint32
}

// JobInfo stores information about a print job.
type JobInfo struct {
	JobID           uint32
	UserMachineName string
	UserName        string
	DocumentName    string
	DataType        string
	Status          string
	StatusCode      uint32
	Priority        uint32
	Position        uint32
	TotalPages      uint32
	PagesPrinted    uint32
	Submitted       time.Time
}

// Jobs returns information about all print jobs on this printer
func (p *Printer) Jobs() ([]JobInfo, error) {
	var bytesNeeded, jobsReturned uint32
	buf := make([]byte, 1)
	for {
		err := EnumJobs(p.h, 0, 255, 1, &buf[0], uint32(len(buf)), &bytesNeeded, &jobsReturned)
		if err == nil {
			break
		}
		if err != syscall.ERROR_INSUFFICIENT_BUFFER {
			return nil, err
		}
		if bytesNeeded <= uint32(len(buf)) {
			return nil, err
		}
		buf = make([]byte, bytesNeeded)
	}
	if jobsReturned <= 0 {
		return nil, nil
	}
	pjs := make([]JobInfo, 0, jobsReturned)
	ji := (*[2048]JOB_INFO_1)(unsafe.Pointer(&buf[0]))[:jobsReturned:jobsReturned]
	for _, j := range ji {
		pji := JobInfo{
			JobID:        j.JobID,
			StatusCode:   j.StatusCode,
			Priority:     j.Priority,
			Position:     j.Position,
			TotalPages:   j.TotalPages,
			PagesPrinted: j.PagesPrinted,
		}
		if j.MachineName != nil {
			pji.UserMachineName = windows.UTF16PtrToString(j.MachineName)
		}
		if j.UserName != nil {
			pji.UserName = windows.UTF16PtrToString(j.UserName)
		}
		if j.Document != nil {
			pji.DocumentName = windows.UTF16PtrToString(j.Document)
		}
		if j.DataType != nil {
			pji.DataType = windows.UTF16PtrToString(j.DataType)
		}
		if j.Status != nil {
			pji.Status = windows.UTF16PtrToString(j.Status)
		}
		if strings.TrimSpace(pji.Status) == "" {
			if pji.StatusCode == 0 {
				pji.Status += "Queue Paused, "
			}
			if pji.StatusCode&JOB_STATUS_PRINTING != 0 {
				pji.Status += "Printing, "
			}
			if pji.StatusCode&JOB_STATUS_PAUSED != 0 {
				pji.Status += "Paused, "
			}
			if pji.StatusCode&JOB_STATUS_ERROR != 0 {
				pji.Status += "Error, "
			}
			if pji.StatusCode&JOB_STATUS_DELETING != 0 {
				pji.Status += "Deleting, "
			}
			if pji.StatusCode&JOB_STATUS_SPOOLING != 0 {
				pji.Status += "Spooling, "
			}
			if pji.StatusCode&JOB_STATUS_OFFLINE != 0 {
				pji.Status += "Printer Offline, "
			}
			if pji.StatusCode&JOB_STATUS_PAPEROUT != 0 {
				pji.Status += "Out of Paper, "
			}
			if pji.StatusCode&JOB_STATUS_PRINTED != 0 {
				pji.Status += "Printed, "
			}
			if pji.StatusCode&JOB_STATUS_DELETED != 0 {
				pji.Status += "Deleted, "
			}
			if pji.StatusCode&JOB_STATUS_BLOCKED_DEVQ != 0 {
				pji.Status += "Driver Error, "
			}
			if pji.StatusCode&JOB_STATUS_USER_INTERVENTION != 0 {
				pji.Status += "User Action Required, "
			}
			if pji.StatusCode&JOB_STATUS_RESTART != 0 {
				pji.Status += "Restarted, "
			}
			if pji.StatusCode&JOB_STATUS_COMPLETE != 0 {
				pji.Status += "Sent to Printer, "
			}
			if pji.StatusCode&JOB_STATUS_RETAINED != 0 {
				pji.Status += "Retained, "
			}
			if pji.StatusCode&JOB_STATUS_RENDERING_LOCALLY != 0 {
				pji.Status += "Rendering on Client, "
			}
			pji.Status = strings.TrimRight(pji.Status, ", ")
		}
		pji.Submitted = time.Date(
			int(j.Submitted.Year),
			time.Month(int(j.Submitted.Month)),
			int(j.Submitted.Day),
			int(j.Submitted.Hour),
			int(j.Submitted.Minute),
			int(j.Submitted.Second),
			int(1000*j.Submitted.Milliseconds),
			time.Local,
		).UTC()
		pjs = append(pjs, pji)
	}
	return pjs, nil
}

// DriverInfo returns information about printer p driver.
func (p *Printer) DriverInfo() (*DriverInfo, error) {
	var needed uint32
	b := make([]byte, 1024*10)
	for {
		err := GetPrinterDriver(p.h, nil, 8, &b[0], uint32(len(b)), &needed)
		if err == nil {
			break
		}
		if err != syscall.ERROR_INSUFFICIENT_BUFFER {
			return nil, err
		}
		if needed <= uint32(len(b)) {
			return nil, err
		}
		b = make([]byte, needed)
	}
	di := (*DRIVER_INFO_8)(unsafe.Pointer(&b[0]))
	return &DriverInfo{
		Attributes:  di.PrinterDriverAttributes,
		Name:        windows.UTF16PtrToString(di.Name),
		DriverPath:  windows.UTF16PtrToString(di.DriverPath),
		Environment: windows.UTF16PtrToString(di.Environment),
	}, nil
}

func (p *Printer) StartDocument(name, datatype string) error {
	d := DOC_INFO_1{
		DocName:    &(syscall.StringToUTF16(name))[0],
		OutputFile: nil,
		Datatype:   &(syscall.StringToUTF16(datatype))[0],
	}
	return StartDocPrinter(p.h, 1, &d)
}

// StartRawDocument calls StartDocument and passes either "RAW" or "XPS_PASS"
// as a document type, depending if printer driver is XPS-based or not.
func (p *Printer) StartRawDocument(name string) error {
	di, err := p.DriverInfo()
	if err != nil {
		return err
	}
	// See https://support.microsoft.com/en-us/help/2779300/v4-print-drivers-using-raw-mode-to-send-pcl-postscript-directly-to-the
	// for details.
	datatype := "RAW"
	if di.Attributes&PRINTER_DRIVER_XPS != 0 {
		datatype = "XPS_PASS"
	}
	return p.StartDocument(name, datatype)
}

func (p *Printer) Write(b []byte) (int, error) {
	var written uint32
	err := WritePrinter(p.h, &b[0], uint32(len(b)), &written)
	if err != nil {
		return 0, err
	}
	if p.Debug {
		p.data = append(p.data, b...)
	}
	return int(written), nil
}

func (p *Printer) EndDocument() error {
	if p.Debug {
		err := ioutil.WriteFile("file.pj", p.data, 0644)
		if err != nil {
			// handle error
		}
	}
	return EndDocPrinter(p.h)
}

func (p *Printer) StartPage() error {
	return StartPagePrinter(p.h)
}

func (p *Printer) EndPage() error {
	return EndPagePrinter(p.h)
}

func (p *Printer) Close() error {
	return ClosePrinter(p.h)
}

type Printer struct {
	h syscall.Handle
	// font metrics
	width, height uint8

	// state toggles ESC[char]
	underline  uint8
	emphasize  uint8
	upsidedown uint8
	rotate     uint8

	// state toggles GS[char]
	reverse, smooth uint8
	Debug           bool
	data            []byte
}

const (
	// ASCII DLE (DataLinkEscape)
	DLE byte = 0x10

	// ASCII EOT (EndOfTransmission)
	EOT byte = 0x04

	// ASCII GS (Group Separator)
	GS byte = 0x1D
)

// text replacement map
var textReplaceMap = map[string]string{
	// horizontal tab
	"&#9;":  "\x09",
	"&#x9;": "\x09",

	// linefeed
	"&#10;": "\n",
	"&#xA;": "\n",

	// xml stuff
	"&apos;": "'",
	"&quot;": `"`,
	"&gt;":   ">",
	"&lt;":   "<",

	// ampersand must be last to avoid double decoding
	"&amp;": "&",
}

// replace text from the above map
func textReplace(data string) string {
	for k, v := range textReplaceMap {
		data = strings.Replace(data, k, v, -1)
	}
	return data
}

// reset toggles
func (p *Printer) reset() {
	p.width = 1
	p.height = 1

	p.underline = 0
	p.emphasize = 0
	p.upsidedown = 0
	p.rotate = 0

	p.reverse = 0
	p.smooth = 0
}

// write a string to the printer
func (p *Printer) WriteString(data string) (int, error) {
	return p.Write([]byte(data))
}

// init/reset printer settings
func (p *Printer) Init() {
	p.reset()
	p.WriteString("\x1B@")
}

// end output
func (p *Printer) End() {
	p.WriteString("\xFA")
}

// send cut
func (p *Printer) Cut() {
	p.WriteString("\x1DVA0")
}

// send cut minus one point (partial cut)
func (p *Printer) CutPartial() {
	p.Write([]byte{GS, 0x56, 1})
}

// send cash
func (p *Printer) Cash() {
	p.WriteString("\x1B\x70\x00\x0A\xFF")
}

// send linefeed
func (p *Printer) Linefeed() {
	p.WriteString("\n")
}

// send N formfeeds
func (p *Printer) FormfeedN(n int) {
	p.WriteString(fmt.Sprintf("\x1Bd%c", n))
}

// send formfeed
func (p *Printer) Formfeed() {
	p.FormfeedN(1)
}

// set font
func (p *Printer) SetFont(font string) {
	f := 0

	switch font {
	case "A":
		f = 0
	case "B":
		f = 1
	case "C":
		f = 2
	default:
		log.Fatalf("Invalid font: '%s', defaulting to 'A'", font)
		f = 0
	}

	p.WriteString(fmt.Sprintf("\x1BM%c", f))
}

func (p *Printer) SendFontSize() {
	p.WriteString(fmt.Sprintf("\x1D!%c", ((p.width-1)<<4)|(p.height-1)))
}

// set font size
func (p *Printer) SetFontSize(width, height uint8) {
	if width > 0 && height > 0 && width <= 8 && height <= 8 {
		p.width = width
		p.height = height
		p.SendFontSize()
	} else {
		log.Fatalf("Invalid font size passed: %d x %d", width, height)
	}
}

// send underline
func (p *Printer) SendUnderline() {
	p.WriteString(fmt.Sprintf("\x1B-%c", p.underline))
}

// send emphasize / doublestrike
func (p *Printer) SendEmphasize() {
	p.WriteString(fmt.Sprintf("\x1BG%c", p.emphasize))
}

// send upsidedown
func (p *Printer) SendUpsidedown() {
	p.WriteString(fmt.Sprintf("\x1B{%c", p.upsidedown))
}

// send rotate
func (p *Printer) SendRotate() {
	p.WriteString(fmt.Sprintf("\x1BR%c", p.rotate))
}

// send reverse
func (p *Printer) SendReverse() {
	p.WriteString(fmt.Sprintf("\x1DB%c", p.reverse))
}

// send smooth
func (p *Printer) SendSmooth() {
	p.WriteString(fmt.Sprintf("\x1Db%c", p.smooth))
}

// send move x
func (p *Printer) SendMoveX(x uint16) {
	p.WriteString(string([]byte{0x1b, 0x24, byte(x % 256), byte(x / 256)}))
}

// send move y
func (p *Printer) SendMoveY(y uint16) {
	p.WriteString(string([]byte{0x1d, 0x24, byte(y % 256), byte(y / 256)}))
}

// set underline
func (p *Printer) SetUnderline(v uint8) {
	p.underline = v
	p.SendUnderline()
}

// set emphasize
func (p *Printer) SetEmphasize(u uint8) {
	p.emphasize = u
	p.SendEmphasize()
}

// set upsidedown
func (p *Printer) SetUpsidedown(v uint8) {
	p.upsidedown = v
	p.SendUpsidedown()
}

// set rotate
func (p *Printer) SetRotate(v uint8) {
	p.rotate = v
	p.SendRotate()
}

// set reverse
func (p *Printer) SetReverse(v uint8) {
	p.reverse = v
	p.SendReverse()
}

// set smooth
func (p *Printer) SetSmooth(v uint8) {
	p.smooth = v
	p.SendSmooth()
}

// pulse (open the drawer)
func (p *Printer) Pulse() {
	// with t=2 -- meaning 2*2msec
	p.WriteString("\x1Bp\x02")
}

// set alignment
func (p *Printer) SetAlign(align string) {
	a := 0
	switch align {
	case "left":
		a = 0
	case "center":
		a = 1
	case "right":
		a = 2
	default:
		log.Fatalf("Invalid alignment: %s", align)
	}
	p.WriteString(fmt.Sprintf("\x1Ba%c", a))
}

// set language -- ESC R
func (p *Printer) SetLang(lang string) {
	l := 0

	switch lang {
	case "en":
		l = 0
	case "fr":
		l = 1
	case "de":
		l = 2
	case "uk":
		l = 3
	case "da":
		l = 4
	case "sv":
		l = 5
	case "it":
		l = 6
	case "es":
		l = 7
	case "ja":
		l = 8
	case "no":
		l = 9
	default:
		log.Fatalf("Invalid language: %s", lang)
	}
	p.WriteString(fmt.Sprintf("\x1BR%c", l))
}

// do a block of text
func (p *Printer) Text(params map[string]string, data string) {

	// send alignment to printer
	if align, ok := params["align"]; ok {
		p.SetAlign(align)
	}

	// set lang
	if lang, ok := params["lang"]; ok {
		p.SetLang(lang)
	}

	// set smooth
	if smooth, ok := params["smooth"]; ok && (smooth == "true" || smooth == "1") {
		p.SetSmooth(1)
	}

	// set emphasize
	if em, ok := params["em"]; ok && (em == "true" || em == "1") {
		p.SetEmphasize(1)
	}

	// set underline
	if ul, ok := params["ul"]; ok && (ul == "true" || ul == "1") {
		p.SetUnderline(1)
	}

	// set reverse
	if reverse, ok := params["reverse"]; ok && (reverse == "true" || reverse == "1") {
		p.SetReverse(1)
	}

	// set rotate
	if rotate, ok := params["rotate"]; ok && (rotate == "true" || rotate == "1") {
		p.SetRotate(1)
	}

	// set font
	if font, ok := params["font"]; ok {
		p.SetFont(strings.ToUpper(font[5:6]))
	}

	// do dw (double font width)
	if dw, ok := params["dw"]; ok && (dw == "true" || dw == "1") {
		p.SetFontSize(2, p.height)
	}

	// do dh (double font height)
	if dh, ok := params["dh"]; ok && (dh == "true" || dh == "1") {
		p.SetFontSize(p.width, 2)
	}

	// do font width
	if width, ok := params["width"]; ok {
		if i, err := strconv.Atoi(width); err == nil {
			p.SetFontSize(uint8(i), p.height)
		} else {
			log.Fatalf("Invalid font width: %s", width)
		}
	}

	// do font height
	if height, ok := params["height"]; ok {
		if i, err := strconv.Atoi(height); err == nil {
			p.SetFontSize(p.width, uint8(i))
		} else {
			log.Fatalf("Invalid font height: %s", height)
		}
	}

	// do y positioning
	if x, ok := params["x"]; ok {
		if i, err := strconv.Atoi(x); err == nil {
			p.SendMoveX(uint16(i))
		} else {
			log.Fatalf("Invalid x param %s", x)
		}
	}

	// do y positioning
	if y, ok := params["y"]; ok {
		if i, err := strconv.Atoi(y); err == nil {
			p.SendMoveY(uint16(i))
		} else {
			log.Fatalf("Invalid y param %s", y)
		}
	}

	// do text replace, then write data
	data = textReplace(data)
	if len(data) > 0 {
		p.WriteString(data)
	}
}

// feed the printer
func (p *Printer) Feed(params map[string]string) {
	// handle lines (form feed X lines)
	if l, ok := params["line"]; ok {
		if i, err := strconv.Atoi(l); err == nil {
			p.FormfeedN(i)
		} else {
			log.Fatalf("Invalid line number %s", l)
		}
	}

	// handle units (dots)
	if u, ok := params["unit"]; ok {
		if i, err := strconv.Atoi(u); err == nil {
			p.SendMoveY(uint16(i))
		} else {
			log.Fatalf("Invalid unit number %s", u)
		}
	}

	// send linefeed
	p.Linefeed()

	// reset variables
	p.reset()

	// reset printer
	p.SendEmphasize()
	p.SendRotate()
	p.SendSmooth()
	p.SendReverse()
	p.SendUnderline()
	p.SendUpsidedown()
	p.SendFontSize()
	p.SendUnderline()
}

// feed and cut based on parameters
func (p *Printer) FeedAndCut(params map[string]string) {
	if t, ok := params["type"]; ok && t == "feed" {
		p.Formfeed()
	}

	p.Cut()
}

// Barcode sends a barcode to the printer.
func (p *Printer) Barcode(barcode string, format int) {
	code := ""
	switch format {
	case 0:
		code = "\x00"
	case 1:
		code = "\x01"
	case 2:
		code = "\x02"
	case 3:
		code = "\x03"
	case 4:
		code = "\x04"
	case 73:
		code = "\x49"
	}

	// reset settings
	p.reset()

	// set align
	p.SetAlign("center")

	// write barcode
	if format > 69 {
		p.WriteString(fmt.Sprintf("\x1dk"+code+"%v%v", len(barcode), barcode))
	} else if format < 69 {
		p.WriteString(fmt.Sprintf("\x1dk"+code+"%v\x00", barcode))
	}
	p.WriteString(fmt.Sprintf("%v", barcode))
}

// used to send graphics headers
func (p *Printer) gSend(m byte, fn byte, data []byte) {
	l := len(data) + 2

	p.WriteString("\x1b(L")
	p.Write([]byte{byte(l % 256), byte(l / 256), m, fn})
	p.Write(data)
}

// write an image
func (p *Printer) Image(params map[string]string, data string) {
	// send alignment to printer
	if align, ok := params["align"]; ok {
		p.SetAlign(align)
	}

	// get width
	wstr, ok := params["width"]
	if !ok {
		log.Fatal("No width specified on image")
	}

	// get height
	hstr, ok := params["height"]
	if !ok {
		log.Fatal("No height specified on image")
	}

	// convert width
	width, err := strconv.Atoi(wstr)
	if err != nil {
		log.Fatalf("Invalid image width %s", wstr)
	}

	// convert height
	height, err := strconv.Atoi(hstr)
	if err != nil {
		log.Fatalf("Invalid image height %s", hstr)
	}

	// decode data frome b64 string
	dec, err := base64.StdEncoding.DecodeString(data)
	if err != nil {
		log.Fatal(err)
	}

	log.Printf("Image len:%d w: %d h: %d\n", len(dec), width, height)

	// $imgHeader = self::dataHeader(array($img -> getWidth(), $img -> getHeight()), true);
	// $tone = '0';
	// $colors = '1';
	// $xm = (($size & self::IMG_DOUBLE_WIDTH) == self::IMG_DOUBLE_WIDTH) ? chr(2) : chr(1);
	// $ym = (($size & self::IMG_DOUBLE_HEIGHT) == self::IMG_DOUBLE_HEIGHT) ? chr(2) : chr(1);
	//
	// $header = $tone . $xm . $ym . $colors . $imgHeader;
	// $this -> graphicsSendData('0', 'p', $header . $img -> toRasterFormat());
	// $this -> graphicsSendData('0', '2');

	header := []byte{
		byte('0'), 0x01, 0x01, byte('1'),
	}

	a := append(header, dec...)

	p.gSend(byte('0'), byte('p'), a)
	p.gSend(byte('0'), byte('2'), []byte{})

}

// write a "node" to the printer
func (p *Printer) WriteNode(name string, params map[string]string, data string) {
	cstr := ""
	if data != "" {
		str := data[:]
		if len(data) > 40 {
			str = fmt.Sprintf("%s ...", data[0:40])
		}
		cstr = fmt.Sprintf(" => '%s'", str)
	}
	log.Printf("WriteString: %s => %+v%s\n", name, params, cstr)

	switch name {
	case "text":
		p.Text(params, data)
	case "feed":
		p.Feed(params)
	case "cut":
		p.FeedAndCut(params)
	case "pulse":
		p.Pulse()
	case "image":
		p.Image(params, data)
	}
}
