package main

import (
	"fmt"
	"io/ioutil"
	"log"
	pdf_parser "pdf_to_ledger/lib/pdf"
	"pdf_to_ledger/lib/query"
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

func TestQuery(t *testing.T) {

	filepath := "./sample/bank_account.pdf"
	log.Printf("Trying to parse: %s\n", filepath)
	file, err := ioutil.ReadFile(filepath)
	if err != nil {
		log.Fatalln(err)
	}
	pdf, err := pdf_parser.Parse(file, nil)
	if err != nil {
		fmt.Print(err)
		t.Fail()
	}
	arg := `@"SALDO INICIAL"+1[4@"SALDO FINAL"]`
	q, err := query.ParseQuery(arg)
	if err != nil {
		log.Fatalln(err)
	}
	result, err := query.RunQuery(q, pdf.Text)
	for _, l := range result {
		for el := range l {
			fmt.Print(el)
			if el < len(l)-2 {
				fmt.Print("\t")
			}
		}
		fmt.Println()
	}
	if err != nil {
		log.Fatalf("Query `%s` did not find any entry\n", err)
	}
	// pdf_parser.Print_objs(pdf)
}
