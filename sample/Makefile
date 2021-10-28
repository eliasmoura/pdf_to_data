
.PHONY: all

all: pdf_example.pdf

pdf_example.pdf: pdf_example.tex
	xelatex $<

clean: pdf_example.log pdf_example.pdf pdf_example.aux
	rm $^
