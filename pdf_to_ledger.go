package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"
	pdf_parser "pdf_to_ledger/lib/pdf"
	"strconv"
	"strings"
)

func usage(progname string) {
	fmt.Printf(`%s usage:
    %s file.pdf [output.txt]
    `, progname, progname)
}

func show_elementes(e []string) {
	for i, v := range e {
		fmt.Printf("%d: %s\n", i, v)
	}
}

type Cmd string

const (
	list   Cmd = "list"
	format Cmd = "formt"
)

func main() {
	progname := os.Args[0]
	progname_ := strings.Split(progname, "/")
	progname = progname_[len(progname_)-1]
	var filepath string
	i := 1
	var arg string
	var cmd Cmd
	for ; i < len(os.Args); i++ {
		switch os.Args[i] {
		case "-f":
			i++
			filepath = os.Args[i]
		case "-e":
			i++
			arg = os.Args[i]
		case "-list":
			cmd = list
		case "-format":
			cmd = format
		case "-help", "-h", "--help":
			usage(progname)
			os.Exit(0)
		default:
			os.Stderr.WriteString(progname)
			spaces := len(progname)
			for j, o := range os.Args[1:] {
			var err_b, err_e string
				if j == i-1 {
					err_b = "\033[4;31m"
					err_e = "\033[0;0m"
				}
				os.Stderr.WriteString(fmt.Sprintf("%s%s%s", err_b, o, err_e))
				if j < len(os.Args[1:]) {
					os.Stderr.WriteString(" ")
				}
				spaces += len(o)
			}
      spaces -= len(os.Args[i])
			os.Stderr.WriteString(fmt.Sprintf("\n%*s%s^^^%s", spaces, "", "\033[1;31m","\033[0;0m"))
			os.Stderr.WriteString(fmt.Sprintf("Unkwon option %s\n", os.Args[i]))
			usage(progname)
			os.Exit(1)
		}
	}

	if len(filepath) < 1 {
		usage(progname)
		filepath = "bank_recipe.pdf"
	}
	file, err := ioutil.ReadFile(filepath)
	if err != nil {
		log.Fatalln(err)
	}
	seq := strings.Split(arg, ",")
	pdf, err := pdf_parser.Parse(file, nil)
	if err != nil {
		fmt.Print(err)
		os.Exit(1)
	}

	switch cmd {
	case list:
		for j, v := range pdf.Text {
			fmt.Printf("%4d: %s\n", j, v)
		}
	case format:
		for j, v := range seq {
			i, err := strconv.ParseInt(v, 10, 32)
			if err != nil {
				log.Println(err)
				log.Fatalln("Failed to parse eletemt, nedd to be an interger.|")
			}
			fmt.Print(pdf.Text[i])
			if j < len(seq)-1 {
				fmt.Print(" ")
			}
		}
		fmt.Println()
	}
}
