package main

import (
	"io/ioutil"
	pdf_parser "local/pdf"
	"log"
	"strings"
  "os"
  "fmt"
)

func usage(progname string) {
  fmt.Printf(`%s usage:
    %s file.pdf [output.txt]
    `, progname, progname)
}

func main() {
  progname := os.Args[0]
  progname_ := strings.Split(progname, "/")
  progname = progname_[len(progname_)-1]
  var filepath string
  if len(os.Args[1:]) < 1 {
    usage(progname)
    filepath = "bank_recipe.pdf"
  } else {
  filepath = os.Args[1]
  }
	file, err := ioutil.ReadFile(filepath)
	if err != nil {
		log.Fatalln(err)
	}
	pdf, err := pdf_parser.Parse(file)
  if err != nil {
    fmt.Print(err)
    os.Exit(1)
  }
	pdf_parser.Print_objs(pdf)
}
