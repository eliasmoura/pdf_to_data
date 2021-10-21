package pdf_to_ledger

import (
	"fmt"
	"io/ioutil"
	"log"
	pdf_parser "pdf_to_ledger/lib/pdf"
	"testing"
)

func TestPDF(t *testing.T) {

	filepath := "./sample/dnsimple_recipe.pdf"
	log.Printf("Trying to parse: %s\n", filepath)
	file, err := ioutil.ReadFile(filepath)
	if err != nil {
		log.Fatalln(err)
	}
	pdf, err := pdf_parser.Parse(file)
	if err != nil {
		fmt.Print(err)
		t.Fail()
	}
	pdf_parser.Print_objs(pdf)
}
