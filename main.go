package main

import (
	"io/ioutil"
	"local/pdf"
	"log"
)

func main() {
	file, err := ioutil.ReadFile("example2.pdf")
	if err != nil {
		log.Fatalln(err)
	}
	// pdf, _ := parse(file)
	pdf, _ := pdf.Parse(file)
	pdf.print_objs(pdf)
	// fmt.Println(pdf.objs[0])
}
