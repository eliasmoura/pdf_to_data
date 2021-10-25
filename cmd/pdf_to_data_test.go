package main

import (
	"fmt"
	"io/ioutil"
	"log"
	pdf_parser "pdf_to_data/lib/pdf"
	"pdf_to_data/lib/query"
	"testing"
)

func TestPDF(t *testing.T) {
  t.SkipNow()
	filepath := "../sample/test.pdf"
	file, err := ioutil.ReadFile(filepath)
	if err != nil {
	log.Printf("Trying to parse: %s\n", filepath)
		log.Fatalln(err)
	}
	_, err = pdf_parser.Parse(file, nil)
	if err != nil {
	log.Printf("Trying to parse: %s\n", filepath)
		log.Println(err)
		t.Fail()
	}
}

func TestQuery(t *testing.T) {
  t.SkipNow()
	filepath := "../sample/bank_account.pdf"
	file, err := ioutil.ReadFile(filepath)
	if err != nil {
	log.Printf("Trying to parse: %s\n", filepath)
		log.Fatalln(err)
	}
	pdf, err := pdf_parser.Parse(file, nil)
	if err != nil {
	log.Printf("Trying to parse: %s\n", filepath)
		fmt.Print(err)
		t.Fail()
	}
	arg := `@"SALDO INICIAL"+1[4@"SALDO FINAL"]`
	q, err := query.ParseQuery(arg)
	if err != nil {
	log.Printf("Trying to parse: %s\n", filepath)
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
	log.Printf("Trying to parse: %s\n", filepath)
		log.Fatalf("Query `%s` did not find any entry\n", err)
	}
}
