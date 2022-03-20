package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"
	pdf_parser "pdf_to_data/lib/pdf"
	"pdf_to_data/lib/query"
	"strings"
)

func usage(progname string) {
	fmt.Printf(`%s usage:
%s -f <filepath> cmd
  -f <filepath>     Indicates where the PDF file is
  cmd               The command you want to execute
    -list           List the indexed text in the PDF file
    -query 'query' The query you want to use.
     @ set the index for the specified:
       "text" match the text.
       #123 match the index.
       +1 increment the index by the specified number.
       [2] indicate the number of elements to be printed per line.
  EXAMPLE:
    %s -f myfile.pdf -query '@"COMPARY"[6@#100]'
      print 6 elements per line, start at the text "COMPARY" and stop at the 100th index.
    `, progname, progname, progname)
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
	var filepath []string
	i := 1
	var arg string
	var cmd Cmd
	var prev_arg string
	for ; i < len(os.Args); i++ {
		switch os.Args[i] {
		case "-f":
			i++
			filepath = append(filepath, os.Args[i])
			prev_arg = "-f"
		case "-list":
			cmd = list
			prev_arg = "-list"
		case "-query":
			cmd = cmd_query
			if len(os.Args) < 5 {
				os.Stderr.WriteString(fmt.Sprintf("ERROR missing query\n"))
				usage(progname)
				os.Exit(1)
			}
			prev_arg = "-query"
			i++
			arg = os.Args[i]
		case "-format":
			cmd = format
			i++
			arg = os.Args[i]
			prev_arg = "-format"
		case "-help", "-h", "--help":
			usage(progname)
			os.Exit(0)
		default:
			os.Stderr.WriteString(progname)
			if prev_arg == "-f" {
				filepath = append(filepath, os.Args[i])
        continue
			}
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
	for i := range filepath {
		file, err := ioutil.ReadFile(filepath[i])
		if err != nil {
			log.Fatalln(err)
		}
		pdf, err := pdf_parser.Parse(file, nil, nil)
		if err != nil {
			fmt.Print(err)
			os.Exit(1)
		}

		switch cmd {
		case list:
			for j, v := range pdf.Text {
				fmt.Printf("%4d: [%s]\n", j, v)
			}
		case cmd_query:
			q, err := query.ParseQuery(arg)
			if err != nil {
				log.Fatalln(err)
			}
			result, err := query.RunQuery(q, pdf.Text)
			for _, l := range result {
				for i, el := range l {
					fmt.Print(el)
					if i < len(l)-1 {
						fmt.Print("\t")
					}
				}
				fmt.Println()
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
}
