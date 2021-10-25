package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"
	pdf_parser "pdf_to_ledger/lib/pdf"
	"pdf_to_ledger/lib/query"
	"strings"
)

func usage(progname string) {
	fmt.Printf(`%s usage:
%s -f <filepath> cmd
  -f <filepath>     Indicates where the PDF file is
  cmd               The command you want to execute
    -list           List the indexed text in the PDF file
    -format 'query' The query you want to use.
    `, progname, progname)
}

func show_elementes(e []string) {
	for i, v := range e {
		fmt.Printf("%d: %s\n", i, v)
	}
}

type Cmd string

const (
	list      Cmd = "list"
	format    Cmd = "formt"
	cmd_query Cmd = "query"
)

type Cmd_colors string

const (
	red    Cmd_colors = "\033[4;31m"
	normal Cmd_colors = "\033[0;0m"
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
		case "-list":
			cmd = list
		case "-query":
			cmd = cmd_query
			i++
			arg = os.Args[i]
		case "-format":
			cmd = format
			i++
			arg = os.Args[i]
		case "-help", "-h", "--help":
			usage(progname)
			os.Exit(0)
		default:
			os.Stderr.WriteString(progname)
			spaces := len(progname)
			for j, o := range os.Args[1:] {
				var err_b, err_e Cmd_colors
				if j == i-1 {
					err_b = red
					err_e = normal
				}
				os.Stderr.WriteString(fmt.Sprintf("%s%s%s", err_b, o, err_e))
				if j < len(os.Args[1:]) {
					os.Stderr.WriteString(" ")
				}
				spaces += len(o)
			}
			spaces -= len(os.Args[i])
			os.Stderr.WriteString(fmt.Sprintf("\n%*s%s^^^%s", spaces, "", red, normal))
			os.Stderr.WriteString(fmt.Sprintf("Unkwon option %s\n", os.Args[i]))
			usage(progname)
			os.Exit(1)
		}
	}

	if len(filepath) < 1 {
		os.Stderr.WriteString(fmt.Sprintf("Missing %s-f <filepath>%s\n", red, normal))
		usage(progname)
		os.Exit(1)
	}
	file, err := ioutil.ReadFile(filepath)
	if err != nil {
		log.Fatalln(err)
	}
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
	case cmd_query:
		q, err := query.ParseQuery(arg)
		if err != nil {
			log.Fatalln(err)
		}
		result, err := query.RunQuery(q, pdf.Text)
		for _, l := range result {
			fmt.Println(l)
		}
		if err != nil {
			log.Fatalf("Query `%s` did not find any entry\n", err)
		}
	default:
		os.Stderr.WriteString(fmt.Sprintf("Missing %s<cmd>%s\n", red, normal))
		usage(progname)
		os.Exit(1)
	}
}
