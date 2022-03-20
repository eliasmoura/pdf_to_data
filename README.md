# pdf_to_data

Transform a pdf accounting document into a data that is machine parsable. Somw what inspired by [jq](https://github.com/stedolan/jq).

This is a toy project to learn how to program in the go language.

`pdf_to_data` will take a pdf file and a convert its content to a machine parsable (cli FS style) format based of a `query` like [jq](https://stedolan.github.io/jq).

## Examples
### List texts
The command bellow list all the contiguous text found in the document.

```sh
pdf_to_data -f myfile.pdf -list
```

### Query for a section
Query allow to show specific sections of the document.

```sh
pdf_to_data -f myfile.pdf -query '@"START TEXT"+1[4@#200]
```

## Query syntax
- `@` set the index for the specified:
  - `"text"` match the text.
  - `#123` match the index.
- `+1` increment the index by the specified number.
- `[2]` indicate the number of elements to be printed per line.


## References:

- [Adobe PDF Reference](https://www.adobe.com/content/dam/acom/en/devnet/pdf/pdfs/pdf_reference_archives/PDFReference.pdf)
- [ledger manual](https://www.ledger-cli.org/3.0/doc/ledger3.html)
- [jq examples](https://stedolan.github.io/jq/tutorial)
- Existing tools:
  - [Adobe converter tools](https://documentcloud.adobe.com/link/tools/?x_api_client_id=adobe_com&x_api_client_location=pdf_to_excel&group=group-convert)
  - [docparser](https://docparser.com/features)
