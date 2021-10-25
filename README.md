# pdf_to_data

Transform a pdf accounting document into a data that is machine parsable. Somw what inspired by [jq](https://github.com/stedolan/jq).

This is a toy project to learn how to program in the go language.

`pdf_to_data` will take a pdf file and a convert its content to a machine parsable (cli FS style) format based of a `query` like [jq](https://stedolan.github.io/jq).

## References:

- [Adobe PDF Reference](https://www.adobe.com/content/dam/acom/en/devnet/pdf/pdfs/pdf_reference_archives/PDFReference.pdf)
- [ledger manual](https://www.ledger-cli.org/3.0/doc/ledger3.html)
- [jq examples](https://stedolan.github.io/jq/tutorial)
- Existing tools:
  - [Adobe converter tools](https://documentcloud.adobe.com/link/tools/?x_api_client_id=adobe_com&x_api_client_location=pdf_to_excel&group=group-convert)
  - [docparser](https://docparser.com/features)
