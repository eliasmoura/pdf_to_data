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
	filepath := "../sample/pdf_example.pdf"
	file, err := ioutil.ReadFile(filepath)
	if err != nil {
		log.Printf("Trying to parse: %s\n", filepath)
		log.Fatalln(err)
	}
	_, err = pdf_parser.Parse(file, nil, nil)
	if err != nil {
		log.Printf("Trying to parse: %s\n", filepath)
		log.Println(err)
		t.Fail()
	}
}

func TestQuery(t *testing.T) {
	filepath := "../sample/pdf_example.pdf"
	file, err := ioutil.ReadFile(filepath)
	if err != nil {
		log.Printf("Trying to parse: %s\n", filepath)
		log.Fatalln(err)
	}
	pdf, err := pdf_parser.Parse(file, nil, nil)
	if err != nil {
		log.Printf("Trying to parse: %s\n", filepath)
		fmt.Print(err)
		t.Fail()
	}
	arg := `@"START"+1[3@"END"]`
	q, err := query.ParseQuery(arg)
	if err != nil {
		log.Printf("Trying to parse: %s\n", filepath)
		log.Fatalln(err)
	}
	result, err := query.RunQuery(q, pdf.Text)
	if err != nil {
		log.Printf("Trying to parse: %s\n", filepath)
		log.Printf("Query `%s` did not find any entry\n", err)
		t.FailNow()
	}
	if result[0][1] != "Some Stuff" || result[2][0] != "10-3" {
		log.Printf("Result did not return rexpected data `%s`.\n", "Some Stuff")
		t.FailNow()
	}
}

func TestERR(t *testing.T) {
	filepath := "../sample/pdf_example.pdf"
	file, err := ioutil.ReadFile(filepath)
	if err != nil {
		log.Printf("Trying to parse: %s\n", filepath)
		log.Fatalln(err)
	}
	pdf, err := pdf_parser.Parse(file, nil, nil)
	if err != nil {
		log.Printf("Trying to parse: %s\n", filepath)
		fmt.Print(err)
		t.Fail()
	}
	arg := `@"START"+1[3@"END"]`
	q, err := query.ParseQuery(arg)
	if err != nil {
		log.Printf("Trying to parse: %s\n", filepath)
		log.Fatalln(err)
	}
	result, err := query.RunQuery(q, pdf.Text)
	if err != nil {
		log.Printf("Trying to parse: %s\n", filepath)
		log.Printf("Query `%s` did not find any entry\n", err)
		t.FailNow()
	}
	if result[0][1] != "Some Stuff" || result[2][0] != "10-3" {
		log.Printf("Result did not return rexpected data.\n")
		t.FailNow()
	}
}
