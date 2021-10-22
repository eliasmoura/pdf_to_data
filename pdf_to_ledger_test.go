package main

import (
	"fmt"
	"io/ioutil"
	"log"
	pdf_parser "pdf_to_ledger/lib/pdf"
	"testing"
)

func TestPDF(t *testing.T) {

	filepath := "./sample/FT A 752253767.pdf"
	log.Printf("Trying to parse: %s\n", filepath)
	file, err := ioutil.ReadFile(filepath)
	if err != nil {
		log.Fatalln(err)
	}
	_, err = pdf_parser.Parse(file, nil)
	if err != nil {
		fmt.Print(err)
		t.Fail()
	}
	// pdf_parser.Print_objs(pdf)
}
