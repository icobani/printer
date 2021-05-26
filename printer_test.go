// Copyright 2013 The Go Authors.  All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package printer

import (
	"bytes"
	"encoding/json"
	"golang.org/x/text/encoding/charmap"
	"log"
	"os"
	"testing"
)

func TestPrinttofile(t *testing.T) {
	filerc, err := os.Open("file.pj")
	if err != nil {
		log.Fatal(err)
	}
	defer filerc.Close()
	PrintToFile(filerc)
}
func PrintToFile(filerc *os.File) error {
	name, err := Default()
	if err != nil {
		return err
	}

	p, err := Open(name)
	p.Debug = true
	if err != nil {
		return err
	}
	defer p.Close()

	err = p.StartDocument(filerc.Name(), "RAW")
	if err != nil {
		return err
	}
	defer p.EndDocument()
	err = p.StartPage()
	if err != nil {
		return err
	}

	p.Init()

	buf := new(bytes.Buffer)
	buf.ReadFrom(filerc)
	contents := buf.Bytes()

	p.Write(contents)

	err = p.EndPage()
	if err != nil {
		return err
	}
	return nil
}

func TestPrinter(t *testing.T) {
	name, err := Default()
	if err != nil {
		t.Fatalf("Default failed: %v", err)
	}

	p, err := Open(name)
	p.Debug = true
	if err != nil {
		t.Fatalf("Open failed: %v", err)
	}
	defer p.Close()

	err = p.StartDocument("my document", "RAW")
	if err != nil {
		t.Fatalf("StartDocument failed: %v", err)
	}
	defer p.EndDocument()
	err = p.StartPage()
	if err != nil {
		t.Fatalf("StartPage failed: %v", err)
	}

	text := "£"
	encoder := charmap.CodePage437.NewEncoder()
	encoded, _ := encoder.String(text)

	p.Init()
	p.SetFontSize(2, 2)
	p.SetFont("B")
	p.SetAlign("center")
	p.WriteString("** CARD PAYMENT **\n")
	p.WriteString("------------------------\n")
	p.WriteString("GETMENULINK Ref: 1544\n")
	p.WriteString("ACEPTED (Auto)\n")
	p.WriteString("------------------------\n")
	p.FormfeedN(2)
	p.SetEmphasize(1)
	p.SetReverse(1)
	p.WriteString("YUM YUM THAI\n")
	p.SetReverse(0)
	p.WriteString("Pickup\n")
	p.SetEmphasize(0)
	p.Formfeed()

	p.SetFont("A")
	p.SetAlign("left")
	p.SetFontSize(1, 1)
	p.WriteString("Date            : 25.05.2021 17:51\n")
	p.WriteString("Server          : Pit\n")
	p.WriteString("Order           : 21/34953\n")
	p.WriteString("Dispatch Time   : 18:20\n")
	p.Formfeed()

	p.SetAlign("center")
	p.SetFont("B")
	p.SetFontSize(2, 2)
	p.SetEmphasize(1)
	p.WriteString("------------------------------\n")
	p.SetUnderline(1)
	p.WriteString("Pickup Details\n")
	p.Formfeed()
	p.SetUnderline(0)
	p.WriteString("Ibrahim COBANI\n")
	p.WriteString("(532 540 1194)\n")
	p.Formfeed()
	p.WriteString("------------------------------\n")
	p.WriteString("ORDER DETAILS\n")
	p.WriteString("------------------------------\n")
	p.SetEmphasize(0)
	p.SetFont("A")
	p.SetFontSize(1, 2)
	p.SetAlign("center")
	p.WriteString("***STARTED***\n")
	p.SetAlign("left")
	p.WriteString("1x3. SA-TAY KING PRAWN\n")
	p.Formfeed()
	p.SetAlign("center")
	p.WriteString("***MAIN***\n")
	p.SetAlign("left")
	p.WriteString("1x61. Jungle Curry with  Chicken\n")
	p.WriteString("1x130. Sauted Aubergine with chilli, Onion & Peppers (V) \n")
	p.WriteString("1x141. Steamed Rice\n")
	p.Formfeed()

	p.SetFont("B")
	p.SetFontSize(2, 2)
	p.SetEmphasize(1)
	p.SetAlign("right")
	p.WriteString("------------------------------\n")
	p.WriteString("Total (4 Items)\n")
	p.WriteString("Total : " + encoded + "29\n")
	p.SetAlign("left")

	p.Formfeed()
	p.Cut()

	p.SetFontSize(2, 2)
	p.SetFont("B")
	p.SetAlign("center")
	p.WriteString("** CARD PAYMENT **\n")
	p.WriteString("------------------------\n")
	p.WriteString("GETMENULINK Ref: 1544\n")
	p.WriteString("ACEPTED (Auto)\n")
	p.WriteString("------------------------\n")
	p.FormfeedN(2)
	p.SetEmphasize(1)
	p.SetReverse(1)
	p.WriteString("YUM YUM THAI\n")
	p.SetReverse(0)
	p.SetEmphasize(0)
	p.SetFont("A")
	p.SetFontSize(1, 1)
	p.WriteString("187 STOKE NEWINGTON HIGH STREET\n")
	p.WriteString("LONDON\n")
	p.WriteString("N16 OLH\n")
	p.WriteString("0207 254 6751\n")
	p.WriteString("www.yumyumthain16.co.uk\n")
	p.WriteString("317318415\n")
	p.WriteString("\n")

	p.Formfeed()

	p.SetAlign("left")
	p.WriteString("Date            : 25.05.2021 17:51\n")
	p.WriteString("Server          : Pit\n")
	p.WriteString("Order           : 21/34953\n")

	p.SetAlign("center")
	p.SetFont("B")
	p.SetFontSize(2, 2)
	p.SetEmphasize(1)
	p.WriteString("------------------------------\n")
	p.WriteString("Dispatch Time   : 18:20\n")

	p.WriteString("------------------------------\n")
	p.SetUnderline(1)
	p.WriteString("Pickup Details\n")
	p.Formfeed()
	p.SetUnderline(0)
	p.WriteString("Ibrahim COBANI\n")
	p.WriteString("(532 540 1194)\n")
	p.Formfeed()
	p.WriteString("------------------------------\n")
	p.WriteString("ORDER DETAILS\n")
	p.WriteString("------------------------------\n")
	p.SetEmphasize(0)
	p.SetFont("A")
	p.SetFontSize(1, 2)
	p.SetAlign("right")
	p.WriteString("1x3. SA-TAY KING PRAWN              " + encoded + "10.95\n")
	p.WriteString("1x61. Jungle Curry with Chicken     " + encoded + "8.95\n")
	p.WriteString("1x141. Steamed Rice                 " + encoded + "2.75\n")
	p.WriteString("1x130. Sauted Aubergine with..      " + encoded + "7.25\n")

	p.SetFont("B")
	p.SetFontSize(2, 2)
	p.SetEmphasize(1)

	p.WriteString("------------------------------\n")
	p.SetFont("A")
	p.SetFontSize(1, 1)
	p.SetEmphasize(1)
	p.WriteString("Sub Total (4 Items)     " + encoded + "29.90\n")
	p.WriteString("Total                   " + encoded + "29.90\n")
	p.WriteString("Paid : (Cards - dineNet)" + encoded + "29.90\n")
	p.Formfeed()
	p.SetAlign("center")
	p.SetFontSize(1, 2)
	p.SetEmphasize(0)
	p.WriteString("Signature _________________________________\n")
	p.Formfeed()
	p.SetEmphasize(1)
	p.WriteString("Thank you, Please call again\n")
	p.WriteString("Yum Yum Thai Restaurants Ltd.\n")

	p.Formfeed()
	p.Cut()

	err = p.EndPage()
	if err != nil {
		t.Fatalf("EndPage failed: %v", err)
	}

}

func TestReadNames(t *testing.T) {
	names, err := ReadNames()
	if err != nil {
		t.Fatalf("ReadNames failed: %v", err)
	}
	name, err := Default()
	if err != nil {
		t.Fatalf("Default failed: %v", err)
	}
	// make sure default printer is listed
	for _, v := range names {
		if v == name {
			return
		}
	}
	t.Fatalf("Default printed %q is not listed amongst printers returned by ReadNames %q", name, names)
}

func TestDriverInfo(t *testing.T) {
	name, err := Default()
	if err != nil {
		t.Fatalf("Default failed: %v", err)
	}

	p, err := Open(name)
	if err != nil {
		t.Fatalf("Open failed: %v", err)
	}
	defer p.Close()

	di, err := p.DriverInfo()
	if err != nil {
		t.Fatalf("DriverInfo failed: %v", err)
	}
	t.Logf("%+v", di)
}

func TestJobs(t *testing.T) {
	names, err := ReadNames()
	if err != nil {
		t.Fatalf("ReadNames failed: %v", err)
	}
	for _, name := range names {
		t.Log("Printer Name:", name)
		p, err := Open(name)
		if err != nil {
			t.Fatalf("Open failed: %v", err)
		}
		defer p.Close()

		pj, err := p.Jobs()
		if err != nil {
			t.Fatalf("Jobs failed: %v", err)
		}
		if len(pj) > 0 {
			t.Log("Print Jobs:", len(pj))
			for _, j := range pj {
				b, err := json.MarshalIndent(j, "", "   ")
				if err == nil && len(b) > 0 {
					t.Log(string(b))
				}
			}
		}
	}
}
