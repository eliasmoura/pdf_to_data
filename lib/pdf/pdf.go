package pdf

import (
	"bytes"
	"compress/zlib"
	"errors"
	"fmt"
	"io"
	"log"
	"strconv"
	"strings"
)

type obj struct {
	Type interface{}
	line int
	col  int
}
type obj_ref struct { // 5 0 R
	id     obj_int
	mod_id obj_int
}
type obj_ind struct { // 5 0 obj\ncontent\nendobj
	id     obj_int
	mod_id obj_int
	objs   []obj
}
type obj_strl struct { // (Some string)
	str        string
	to_balance int
}
type obj_strh string  // <2ca231fd1>
type obj_named string // /NAME
type obj_pair struct {
	key   obj
	value obj
}
type obj_dict []obj_pair // <</key [/value 1 2 R]>>
type obj_array []obj     // [(ds) /qq [null]]
type obj_stream struct { // stream\ncontent\nendstream
	decoded         bool
	encoded_content []byte
	decoded_content []byte
	objs            []obj
}
type obj_bool bool      // true/false
type obj_int int        // 123/-11/+23
type obj_real float64   // 01.2/-.2/+3.1
type obj_null []byte    // null
type obj_comment string // %comment
type obj_eof string     // %%EOF

type pdf struct {
	ver struct {
		major, minor int
	}
	objs []obj
}

type close_obj struct {
	obj    obj   // witch obj holds the last `count` objs
	childs []obj // num of objs that should be added to the `obj`
}

type xref_ref struct {
	n, m obj_int
	c    string
}

type obj_xref struct {
	id        obj_int
	refs      []xref_ref
	enc       obj_dict
	startxref obj_int
}

func append_pair(slice []obj_pair, data ...obj_pair) []obj_pair {
	m := len(slice)
	n := m + len(data)
	if n > cap(slice) { // if necessary, reallocate
		// allocate double what's needed, for future growth.
		newSlice := make([]obj_pair, (n+1)*2)
		copy(newSlice, slice)
		slice = newSlice
	}
	slice = slice[0:n]
	copy(slice[m:n], data)
	return slice
}

func typeStr(t obj) string {
	switch t.Type.(type) {
	case obj_ind:
		return "obj_ind"
	case obj_ref:
		return "obj_ref"
	case obj_strl:
		return "obj_strl"
	case obj_strh:
		return "obj_strh"
	case obj_named:
		return "obj_named"
	case obj_pair:
		return "obj_pair"
	case obj_dict:
		return "obj_dict"
	case obj_array:
		return "obj_array"
	case obj_stream:
		return "obj_stream"
	case obj_bool:
		return "obj_bool"
	case obj_int:
		return "obj_int"
	case obj_real:
		return "obj_real"
	case obj_null:
		return "obj_null"
	case obj_comment:
		return "obj_comment"
	case obj_eof:
		return "obj_eof"
	case obj_xref:
		return "obj_xref"
	case xref_ref:
		return "ref"
	default:
		log.Printf("Warnning: %T is not a PDF obj or not implemented\n", t.Type)
		return "Not any PDF obj"
	}
}

func AppendCloseObj(c []close_obj, data obj) []close_obj {
	m := len(c)
	if m == cap(c) { // if necessary, reallocate
		// allocate double what's needed, for future growth.
		newSlice := make([]close_obj, (m+1)*2)
		copy(newSlice, c)
		c = newSlice
	}
	c = c[0 : m+1]
	c[m] = close_obj{data, nil}
	return c
}

func Append(slice []obj, data ...obj) []obj {
	m := len(slice)
	n := m + len(data)
	if n > cap(slice) { // if necessary, reallocate
		// allocate double what's needed, for future growth.
		newSlice := make([]obj, (n+1)*2)
		copy(newSlice, slice)
		slice = newSlice
	}
	slice = slice[0:n]
	copy(slice[m:n], data)
	return slice
}

func AppendRef(slice []xref_ref, data ...xref_ref) []xref_ref {
	m := len(slice)
	n := m + len(data)
	if n > cap(slice) { // if necessary, reallocate
		// allocate double what's needed, for future growth.
		newSlice := make([]xref_ref, (n+1)*2)
		copy(newSlice, slice)
		slice = newSlice
	}
	slice = slice[0:n]
	copy(slice[m:n], data)
	return slice
}

var delimiters = []byte{'(', ')', '<', '>', '[', ']', '{', '}', '/', '%', ' ', '\n', ''}

func get_token(txt []byte, byte ...interface{}) (string, int) {
	var pos int
	size := len(txt)
	txt = bytes.TrimSpace(txt)
	pos = size - len(txt)
	for i := range txt {
		for _, d := range delimiters {
			if txt[i] == d {
				if i > 0 && txt[i-1] == '\\' {
					break
				}
				if ((d == '<' && i+1 < len(txt) && txt[i+1] == '<') ||
					(d == '>' && i+1 < len(txt) && txt[i+1] == '>')) && i == 0 {
					// TODO(k0tto): check if there is a better way to do this
					// if there isn't, then make proper checks.
					i += 2
				}
				if i == 0 {
					return string(txt[0]), pos + 1
				}
				return string(txt[0:i]), pos + i
			}
		}
	}
	return string(txt), size
}

func read_strl(txt []byte) (string, int) {
	to_balance := 0
	for i := range txt {
		if txt[i] == ')' {
			if txt[i-1] == '\\' {
				continue
			}
			to_balance--
			if i == 0 {
				return string(txt[0]), to_balance
			}
			return string(txt[0:i]), to_balance
		} else if txt[i] == '(' {
			to_balance++
		}
	}
	return string(txt), to_balance
}

func read_strh(txt []byte) (string, error) {
	for i := range txt {
		if txt[i] == '>' {
			if i == 0 {
				return string(txt[0]), nil
			}
			return string(txt[0:i]), nil
		}
	}
	return string(txt), errors.New("EOF")
}

func read_until(txt []byte, ch string) ([]byte, error) {
	to_balance := 0
	for i := range txt {
		if txt[i] == '>' && txt[i+1] == '>' {
			to_balance--
			if to_balance > 0 {
				return nil, errors.New("Could not find the matching `>>`\n")
			}
			if i == 0 {
				return []byte{txt[0]}, nil
			}
			return txt[0:i], nil
		}
		if txt[i] == '<' && txt[i+1] == '<' {
			to_balance++
		}
	}
	return txt, errors.New("EOF")
}

func index_from_bread(lines []line, bread int) (int, error) {
	for i := range lines {
		if lines[i].end >= bread && lines[i].start <= bread {
			return i, nil
		}
	}
	return 0, errors.New(fmt.Sprintf("Read %d bytes, but last line ent at %d bytes\n", bread, lines[len(lines)-1].end))
}

func RemoveCloseObj(c []close_obj) ([]close_obj, close_obj) {
	n := len(c)
	if n > 0 {
		n--
	} else {
		log.Fatalf("ERROR: RemoveCloseObj len is 0\n")
	}
	o := c[n]
	c = c[:n]
	return c, o
}

func DecreaseCloseObj(c []close_obj, n int) []close_obj {
	c_len := len(c)
	s2 := len(c[c_len-1].childs)
	dec := s2 - n
	if dec >= 0 {
		if dec > 0 {
			c[c_len-1].childs = c[c_len-1].childs[:dec]
		} else {
			c, _ = RemoveCloseObj(c)
		}
	} else {
		log.Fatalln("ERROR: should be adding to the obj_to_close insted of going negative--")
	}
	return c
}

func Pop(objs []obj) ([]obj, obj) {
	m := len(objs)
	if m == 0 {
		log.Fatalln("ERROR: slice obj is len", m, "need to be >0")
	}
	o := objs[m-1]
	objs = objs[:m-1]
	return objs, o
}

func AppendChild(c []close_obj, o obj) []close_obj {
	if len(c) == 0 {
		c = AppendCloseObj(c, obj{nil, 0, 0})
	}
	c[len(c)-1].childs = Append(c[len(c)-1].childs, o)
	return c
}

func get_token_str(t obj, close bool) string {
	switch v := t.Type.(type) {
	case obj_ind:
		if close {
			return "endobj"
		}
		return "<int> <int> obj"
	case obj_ref:
		return "<int> <int> R"
	case obj_strl:
		if close {
			return ")"
		}
		return "("
	case obj_strh:
		if close {
			return ">"
		}
		return "<"
	case obj_named:
		return "/"
	case obj_pair:
		return ""
	case obj_dict:
		if close {
			return ">>"
		}
		return "<<"
	case obj_array:
		if close {
			return "]"
		}
		return "["
	case obj_stream:
		if close {
			return "endstream"
		}
		return "stream"
	case obj_bool:
		if bool(v) {
			return "true"
		}
		return "false"
	case obj_int:
		return "<int>"
	case obj_real:
		return "<real>"
	case obj_null:
		return "null"
	case obj_comment:
		return "%"
	case obj_xref:
		return "xref"
	case xref_ref:
		return "xref_ref"
	case obj_eof:
		return "%%EOF"
	default:
		log.Printf("Warnning: %T[%v] is not a PDF obj or not implemented\n", t.Type, v)
		return "Not any PDF obj"
	}

}

type line struct {
	index, start, end int
}

func AppendLine(lines []line, start, end int) []line {
	m := len(lines)
	if m == cap(lines) {
		newlines := make([]line, m, (m+1)*2)
		copy(newlines, lines)
		lines = newlines
	}
	var index int
	if m > 0 {
		index = lines[m-1].index
	} else {
		index = 0
	}
	lines = lines[:m+1]
	lines[m] = line{index + 1, start, end}
	return lines
}

func get_endstream(txt []byte) (int, error) {
	i := 0
	for i < len(txt)-9 {
		if txt[i] == 'e' &&
			txt[i+1] == 'n' &&
			txt[i+2] == 'd' &&
			txt[i+3] == 's' &&
			txt[i+4] == 't' &&
			txt[i+5] == 'r' &&
			txt[i+6] == 'e' &&
			txt[i+7] == 'a' &&
			txt[i+8] == 'm' {
			return i, nil
		}
	}
	return 0, errors.New("Coulds not find `endstream`")
}

func Parse(doc []byte) (pdf, error) {
	var obj_to_close []close_obj
	line_index := 0
	var lines []line
	bread := 0
	var pdf pdf
	header := []byte("%PDF-") //Ex: %PDF-1.7
	var start, end int
	for end > -1 {
		var tend int
		tend = bytes.IndexByte(doc[start:], '\n')
		if tend == -1 {
			end = len(doc[start:]) + start
		} else {
			end = tend + start
		}
		lines = AppendLine(lines, start, end)
		start = end + 1
		end = tend
	}
	bread = len(header)
	if !bytes.HasPrefix(doc[:bread], header) {
		return pdf, errors.New("File is not a PDF format.")
	}
	ver_ := doc[bread:lines[0].end]
	ver := bytes.Split(ver_, []byte("."))
	if len(ver) != 2 {
		return pdf, errors.New(fmt.Sprintf("ERROR:%d:%d: Failed to parse PDF version from `%v` is not a valid version `m.n`\n", line_index+1, len(header), doc[line_index]))
	}
	i, err := strconv.ParseInt(string(ver[0]), 10, 32)
	if err != nil {
		return pdf, errors.New(fmt.Sprintf("ERROR:%d:%d: Failed to parse PDF version `%v` is not an integer\n", line_index+1, len(header), ver[0]))
	}
	pdf.ver.major = int(i)
	i, err = strconv.ParseInt(string(ver[1]), 10, 32)
	if err != nil {
		return pdf, errors.New(fmt.Sprintf("ERROR:%d:%d: Failed to parse PDF version `%v` is not an integer\n", line_index+1, len(header)+len(ver[0])+1, ver[1]))
	}
	pdf.ver.minor = int(i)
	line_index++
	bread = lines[0].end + 1

	line := doc[lines[1].start:lines[1].end]
	if '%' == line[0] &&
		line[1] > 128 &&
		line[2] > 128 &&
		line[3] > 128 {
		// "PDF has binary content."
		line_index++
		bread += len(line)
		bread++
	}

	for line_index < len(lines) {
		col := 0
		line = doc[lines[line_index].start:lines[line_index].end]
		for col < len(line) {
			token, pos := get_token(line[col:])
			before_token_len := pos - len(token)
			col += before_token_len
			// bread += before_token_len
			var objc obj
			var is_stream_encoded bool
			var stream_decoded_len int
			var closed_obj obj
			{
				switch token {
				case "%":
					// % defines a commemt and it goes to the end of the line
					if line[col+1] == '%' &&
						line[col+2] == 'E' &&
						line[col+3] == 'O' &&
						line[col+4] == 'F' {
						objc = obj{obj_eof("EOF"), line_index + 1, col + 1 + before_token_len}
						col = len(line)
						if len(obj_to_close) == 1 {
							o_xref := obj_to_close[0].obj
							xref, ok := o_xref.Type.(obj_xref)
							if ok && len(obj_to_close[0].childs) > 0 {
								var oc close_obj
								obj_to_close, oc = RemoveCloseObj(obj_to_close)
								childs := oc.childs
								childs, o_start := Pop(childs)
								startxref, ok := o_start.Type.(obj_int)
								if !ok {
									log.Printf("Wrong!\n")
								}
								xref.startxref = startxref
								childs, o_dict := Pop(childs)
								xref.enc, ok = o_dict.Type.(obj_dict)
								if !ok {
									log.Printf("Wrong2!\n")
								}
								for i := range childs {
									ref, ok := childs[i].Type.(xref_ref)
									if ok {
										xref.refs = AppendRef(xref.refs, ref)
									}
								}
								pdf.objs = Append(pdf.objs, obj{xref, o_xref.line, o_xref.col})
							} else {

								for _, o := range obj_to_close[len(obj_to_close)-1].childs {
									pdf.objs = Append(pdf.objs, o)
								}
							}
						}
					} else {
						col++
						bread++
						token = string(line[col:])
						objc = obj{obj_comment(token), line_index + 1, col + before_token_len}
						closed_obj = objc
					}
				case "(":
					//- strings []u8. Empty strings is valid:
					//  (liteal) may contem new lines,(),*,!,&,^,%,\),\\…\ddd(octal up to 3 digit)
					str := obj_strl{}
					o := obj{str, line_index + 1, col + 1 + before_token_len}

					var balance int
					col++
					token, balance = read_strl(doc[lines[line_index].start+col:])
					str.str += token
					o.Type = str
					if balance > 0 {
						log.Fatalf("ERROR:%d:%d expected token `)`, found EOF\n", o.line, o.col)
					}
					col += len(token) + 1
					if len(obj_to_close) > 0 {
						obj_to_close = AppendChild(obj_to_close, o)
					} else {
						pdf.objs = Append(pdf.objs, o)
					}
				case ")":
					objc = obj{obj_strl{}, line_index + 1, col + 1 + before_token_len}
				case "<<":
					//- <<…>> denotes a dictionary like
					//  <</Type /Example >>
					o := obj{obj_dict{}, line_index + 1, col + 1 + before_token_len}
					obj_to_close = AppendCloseObj(obj_to_close, o)
				case ">>":
					o := obj_to_close[len(obj_to_close)-1].obj
					dict, ok := o.Type.(obj_dict)
					if ok {
						var oc close_obj
						obj_to_close, oc = RemoveCloseObj(obj_to_close)
						childs := oc.childs
						if (len(childs) % 2) != 0 {
							log.Print("ERROR: dictionary is not even: ")
							log.Printf("%v\n", oc)
						}
						for i := 0; i < len(childs); i += 2 {
							dict = append_pair(dict, obj_pair{childs[i], childs[i+1]})
						}
						closed_obj = obj{dict, oc.obj.line, oc.obj.col}
					} else {
						log.Fatalf("+ERROR:%d:%d Expected `%v`, found `>>`.\n", o.line, o.col, typeStr(o))
					}
				case "<":
					//  <hexadecimal string> ex <ab901f> if missing a digit ex<ab1>, <ab10> is assumed.
					//TODO(elias): Need to figure out where this should be

					o := obj{obj_strh(""), line_index + 1, col + 1 + before_token_len}
					col++
					token, err = read_strh(doc[lines[line_index].start+col:])
					if err != nil {
						log.Fatalf("ERROR:%d:%d expected token `>`, found EOF\n", o.line, o.col)
					}
					strh := token
					size := len(strh)
					if size%2 != 0 {
						strh = strh + "0"
						size++
					}
					size = size / 2
					shex := make([]uint64, size)
					for i := range shex {
						it := i * 2
						var err error
						shex[i], err = strconv.ParseUint(strh[it:it+2], 16, 0)
						if err != nil {
							log.Println("cound not Parse value in hexadecimal string: ", strh[it:it+1])
						}
					}
					s := make([]string, len(shex))
					for i := range shex {
						s[i] = fmt.Sprintf("%c", shex[i])
					}
					fstr := strings.Join(s, "")
					o.Type = obj_strh(fstr)
					col += len(token) + 1
					closed_obj = o
				case ">":
					objc = obj{obj_strh(""), line_index + 1, col + 1 + before_token_len}
				case "/":
					//- named objects start with the prefix / with no white spaces or delimiters
					//  they are case sensitive… /Name1 /other /@this /$$ /1.2 /aa;dd_ss**a? /.notdef are valid.
					// TODO(k0tto): Need to handle the use of characters in hex as `/GF#3A`
					//  PDF>1.2 /#13asd is valid(hexadecimal of of invalid character)
					col++
					// bread++
					token, pos = get_token(line[col:])
					obj_to_close = AppendChild(obj_to_close, obj{obj_named(token), line_index + 1, col + 1 + before_token_len})
				case "R":
					//- obj ref, 1 0 R, Where `1 0` refers to an obj_ind
					childs, mod_id := Pop(obj_to_close[len(obj_to_close)-1].childs)
					childs, id := Pop(childs)
					obj_to_close[len(obj_to_close)-1].childs = childs

					id_val, ok1 := id.Type.(obj_int)
					mod_id_val, ok2 := mod_id.Type.(obj_int)
					if ok1 && ok2 {
						objc = obj{obj_ref{id_val, mod_id_val}, line_index + 1, col + 1 + before_token_len}
						closed_obj = objc
					} else {
						log.Printf("ERROR: token not an integer: id: [%T] mod: [%T]", id.Type, mod_id.Type)
					}
				case "[":
					//- [] denotes an array like [32 12.5 false (txt) /this]
					o := obj{obj_array{}, line_index + 1, col + 1 + before_token_len}
					obj_to_close = AppendCloseObj(obj_to_close, o)
				case "]":
					objc = obj{obj_array{}, line_index + 1, col + 1 + before_token_len}
					oc := obj_to_close[len(obj_to_close)-1]
					o, ok := oc.obj.Type.(obj_array)
					if ok {
						obj_to_close, oc = RemoveCloseObj(obj_to_close)
						childs := oc.childs
						for _, c := range childs {
							o = Append(o, c)
						}
						closed_obj = obj{o, oc.obj.line + 1, oc.obj.col + 1 + before_token_len}
					} else {
						log.Fatalf("ERROR: objc: %T, obj_to_close[last]: %T\n", objc.Type, o)
					}
				case "obj":
					//- any obj that may or maynot be refered by any obj_ref
					childs, mod_id := Pop(obj_to_close[len(obj_to_close)-1].childs)
					childs, id := Pop(childs)
					obj_to_close[len(obj_to_close)-1].childs = childs

					if len(childs) == 0 && obj_to_close[len(obj_to_close)-1].obj.Type == nil {
						obj_to_close, _ = RemoveCloseObj(obj_to_close)
					}
					id_val, ok1 := id.Type.(obj_int)
					mod_id_val, ok2 := mod_id.Type.(obj_int)
					if ok1 && ok2 {
						o := obj{obj_ind{id_val, mod_id_val, nil}, line_index + 1, col + 1 + before_token_len}
						obj_to_close = AppendCloseObj(obj_to_close, o)
					} else {
						log.Print("ERROR: token not an integer: 1", ok1, "2", ok2)
					}
				case "endobj":
					objc = obj{obj_ind{}, line_index + 1, col + 1 + before_token_len}
					oc := obj_to_close[len(obj_to_close)-1]
					o, ok := oc.obj.Type.(obj_ind)
					if ok {
						obj_to_close, oc = RemoveCloseObj(obj_to_close)
						childs := oc.childs
						for _, c := range childs {
							o.objs = Append(o.objs, c)
						}
						closed_obj = obj{o, oc.obj.line, oc.obj.col}
					} else {
						log.Fatalf("ERROR: objc: %T, obj_to_close[last]: %T\n", objc.Type, oc.obj.Type)
					}
					break
				case "stream":
					//- the content that will be displayed to in the page

					line_index, err = index_from_bread(lines, bread)
					o_ind := obj_to_close[len(obj_to_close)-1].obj
					_, ok := o_ind.Type.(obj_ind)
					var stream []byte
					delay_decode := false
					// end_stream, _ := get_endstream(doc[lines[line_index].start:])
					if ok {
						{
							childs := obj_to_close[len(obj_to_close)-1].childs
							o_dict := childs[len(childs)-1]
							metadata, ok := o_dict.Type.(obj_dict)
							var filter string
							var length int
							if ok {
								for i := range metadata {
									key, ok := metadata[i].key.Type.(obj_named)
									if ok && string(key) == "Filter" {
										of, ok := metadata[i].value.Type.(obj_named)
										if ok {
											filter = string(of)
											break
										}
									}
								}
								for i := range metadata {
									key, ok := metadata[i].key.Type.(obj_named)
									if ok && string(key) == "Length" {
										oi, ok := metadata[i].value.Type.(obj_int)
										if ok {
											length = int(oi)
											break
										} else {
											oi, ok := metadata[i].value.Type.(obj_ref)
											if ok {
												delay_decode = true
												length = int(oi.id)
												break
											}
										}
									}
								}
								if length > 0 {
									if !delay_decode {
										if filter == "FlateDecode" {
											fmt.Print("Decoding FlateDecode\n", len(line), "\n")
											if stream_decoded_len < length {
												stream_decoded_len += len(line)
												line_index++
												// Assume the data start in a new line.
												start := lines[line_index].start
												end := start + length
												// bbread := bread
												bread += length
												r, err := zlib.NewReader(bytes.NewReader(doc[start:end]))
												if err != nil {
													fmt.Println(string(doc[start:end]))
													return pdf, errors.New(fmt.Sprintf("failled to decode:%d:%d %v", line_index+1, col, err))
												}
												b, err := io.ReadAll(r)
												fmt.Printf("Read: %s\n", b)
												if err != nil {
													fmt.Println(string(doc[start:end]))
													return pdf, errors.New(fmt.Sprintf("failled to readall:%d:%d %v", line_index+1, col, err))
												}

												// stream = append(stream, b)
											}
										}
									}
								}
							}
						}
					}
					var o obj
					if !delay_decode {
						if len(stream) > 0 {
							o = obj{obj_stream{objs: []obj{{stream, 0, 0}}}, line_index + 1, col + 1 + before_token_len}
						} else {
							o = obj{obj_stream{}, line_index + 1, col + 1 + before_token_len}
						}
					} else {
						o = obj{obj_stream{encoded_content: stream}, line_index + 1, col + 1 + before_token_len}
					}
					obj_to_close = AppendChild(obj_to_close, o)

				case "endstream":
					//everything is proccessed in after the `stream` token.
				case "false":
					//- boolean false
					obj_to_close = AppendChild(obj_to_close, obj{obj_bool(false), line_index + 1, col + 1 + before_token_len})
				case "true":
					//- boolean true
					obj_to_close = AppendChild(obj_to_close, obj{obj_bool(true), line_index + 1, col + 1 + before_token_len})
				case "null":
					//- null obj
					obj_to_close = AppendChild(obj_to_close, obj{obj_null(nil), line_index + 1, col + 1 + before_token_len})
				case "xref":
					obj_to_close = AppendCloseObj(obj_to_close, obj{obj_xref{}, line_index + 1, col + 1 + before_token_len})
				case "trailer", "startxref":
				default:
					//- numbers 10 +12 -12 0 32.5 -.1 +21.0 4. 0.0
					//  if the interger exceeds the limit it is converted to a real(float)
					//  interger is auto converted to real when needed
					if !is_stream_encoded {
						num_int, err := strconv.ParseInt(token, 10, 0)
						var obj_num obj
						if err == nil {
							obj_num = obj{obj_int(num_int), line_index + 1, col + 1 + before_token_len}
							obj_to_close = AppendChild(obj_to_close, obj_num)
						} else {
							num_float, err := strconv.ParseFloat(token, 0)
							if err == nil {
								obj_num = obj{obj_real(num_float), line_index + 1, col + 1 + before_token_len}
								obj_to_close = AppendChild(obj_to_close, obj_num)
							} else {
								_, ok := obj_to_close[len(obj_to_close)-1].obj.Type.(obj_xref)
								if ok {
									if len(token) == 1 && (token[0] == 'f' || token[0] == 'n') {
										_, ok := obj_to_close[len(obj_to_close)-1].obj.Type.(obj_xref)
										if ok {
											childs, m_ := Pop(obj_to_close[len(obj_to_close)-1].childs)
											childs, n_ := Pop(childs)
											obj_to_close[len(obj_to_close)-1].childs = childs

											n, ok1 := n_.Type.(obj_int)
											m, ok2 := m_.Type.(obj_int)
											if ok1 && ok2 {
												o := obj{xref_ref{n, m, token}, m_.line, m_.col}
												obj_to_close = AppendChild(obj_to_close, o)
											} else {
												log.Print("ERROR: token not an integer: 1", ok1, "2", ok2)
											}
										}
									}
								} else {
									_, ok := obj_to_close[len(obj_to_close)-1].obj.Type.(obj_stream)
									if ok {
										obj_to_close = AppendChild(obj_to_close, obj{token, line_index + 1, col + 1 + before_token_len})
									} else {
										log.Printf("Can't parse token `%s` at %d:%d\n", token, line_index+1, col+1+before_token_len)
									}
								}
							}
						}
					}
				}
			}
			if objc.Type != nil {
				switch objc.Type.(type) {
				case obj_strh:
				case obj_strl:
				case obj_dict:
				case obj_ind:
				case obj_stream:
				case obj_array:
				case obj_comment:
				case obj_eof:
					closed_obj = objc
					if len(obj_to_close) == 1 {
						o := obj_to_close[0].obj
						if o.Type == nil && len(obj_to_close[0].childs) == 1 {
							pdf.objs = Append(pdf.objs, obj_to_close[0].childs[0])
							obj_to_close, _ = RemoveCloseObj(obj_to_close)
						} else {
							xref, ok := obj_to_close[0].obj.Type.(obj_xref)
							childs := obj_to_close[0].childs
							if ok && len(childs) > 0 {
								id, ok_id := childs[0].Type.(obj_int)
								if !ok_id {
									return pdf, errors.New(fmt.Sprintf("ERROR:%d:%d: %v should be an obj_int is %T\n", childs[0].line, childs[0].col, childs[0], childs[0]))
								}
								xref.id = id
								tot_line := childs[1].line
								tot_col := childs[1].col
								tot, ok_tot := childs[1].Type.(obj_int)
								if !ok_tot {
									return pdf, errors.New(fmt.Sprintf("ERROR:%d:%d: %v should be an obj_int is %T\n", childs[1].line, childs[1].col, childs[1], childs[1]))

								}
								childs = childs[2:]
								childs, o = Pop(childs)
								pos, ok_pos := o.Type.(obj_int)
								if !ok_pos {
									return pdf, errors.New(fmt.Sprintf("ERROR:%d:%d: %v should be an obj_int is %T\n", o.line, o.col, o, o))
								}
								childs, o = Pop(childs)
								dict, ok_dict := o.Type.(obj_dict)
								if !ok_dict {
									return pdf, errors.New(fmt.Sprintf("ERROR:%d:%d: %v should be an obj_dict is %T\n", o.line, o.col, o, o))
								}
								xref.enc = dict
								xref.startxref = pos
								if int(tot) != len(childs) {
									return pdf, errors.New(fmt.Sprintf("ERROR:%d:%d: given number of xrefs `%d` doesn't match the number of xrefs found `%d`.\n", tot_line, tot_col, tot, len(childs)))
								}
								for i := range childs {
									ref, ok := childs[i].Type.(xref_ref)
									if !ok {
										return pdf, errors.New(fmt.Sprintf("ERROR:%d:%d: expect xref ref, found %s[%v]\n", childs[i].line, childs[i].col, typeStr(childs[i]), childs[i].Type))
									}
									xref.refs = AppendRef(xref.refs, ref)
								}
							}
						}
					}
				case obj_ref:
				default:
					log.Printf("-Warning: %d:%d ", line_index, col)
					log.Printf("Closing %T, but it is not implemented or invalid.\n", objc.Type)
				}
			}
			col += len(token)
			if col > len(line) {
				col = len(line)
			}
			if closed_obj.Type != nil {
				if len(obj_to_close) > 0 {
					obj_to_close = AppendChild(obj_to_close, closed_obj)
				} else {
					pdf.objs = Append(pdf.objs, closed_obj)
				}
			}
		}
		bread += col + 1
		line_index++
	}
	if len(obj_to_close) > 0 {
		return pdf, errors.New(fmt.Sprintf("%%EOF found, expected token %v\n", get_token_str(obj_to_close[len(obj_to_close)-1].obj, true)))
	}
	return pdf, nil
}

type p struct {
	obj obj
	n   int
}

func Appendp(slice []p, data ...p) []p {
	m := len(slice)
	n := m + len(data)
	if n > cap(slice) { // if necessary, reallocate
		// allocate double what's needed, for future growth.
		newSlice := make([]p, (n+1)*2)
		copy(newSlice, slice)
		slice = newSlice
	}
	slice = slice[0:n]
	copy(slice[m:n], data)
	return slice
}

func Print_objs(pdf pdf) {
	var indent int
	to_close := make([]obj, len(pdf.objs))
	to_close_ := make([]p, len(to_close))
	ri := len(pdf.objs) - 1
	for i := 0; i < len(pdf.objs); i++ {
		to_close[i] = pdf.objs[ri]
		ri--
	}
	for len(to_close) > 0 {
		for i := 0; i < indent; i++ {
			fmt.Print("  ")
		}
		var o obj
		to_close, o = Pop(to_close)
		switch t := o.Type.(type) {
		case obj_ind:
			fmt.Println("\n", t.id, t.mod_id, "obj")
			if len(t.objs) > 0 {
				ri := len(t.objs) - 1
				to_close_ = Appendp(to_close_, p{obj{obj_ind{}, 0, 0}, len(t.objs)})
				for ; ri >= 0; ri-- {
					to_close = Append(to_close, t.objs[ri])
				}
			}
			continue
		case obj_array:
			fmt.Print(get_token_str(obj{t, 0, 0}, false))
			if len(t) > 0 {
				ri := len(t) - 1
				to_close_ = Appendp(to_close_, p{obj{obj_array{}, 0, 0}, len(t)})
				for ; ri >= 0; ri-- {
					to_close = Append(to_close, t[ri])
				}
			}
			continue
		case obj_dict:
			fmt.Print(get_token_str(obj{t, 0, 0}, false))
			if len(t) > 0 {
				ri := len(t) - 1
				to_close_ = Appendp(to_close_, p{obj{t, 0, 0}, len(t)})
				for ; ri >= 0; ri-- {
					to_close = Append(to_close, obj{Type: t[ri]})
				}
			}
			continue
		case obj_pair:
			to_close = Append(to_close, obj{Type: t.value.Type, line: t.value.line, col: t.value.col})
			to_close = Append(to_close, obj{Type: t.key.Type, line: t.key.line, col: t.key.col})
			to_close_[len(to_close_)-1].n += 1
		case obj_ref:
			fmt.Print(" ", t.id, t.mod_id, " R")
			to_close_[len(to_close_)-1].n--
		case obj_strl:
			fmt.Printf("(%s)", t.str)
			to_close_[len(to_close_)-1].n--
		case obj_strh:
			fmt.Printf("<%s>", t)
			to_close_[len(to_close_)-1].n--
		case obj_named:
			fmt.Printf("/%s", t)
			to_close_[len(to_close_)-1].n--
		case obj_stream:
			fmt.Print("\nstream\n")
			ri := len(t.objs) - 1
			if ri >= 0 {
				to_close_ = Appendp(to_close_, p{obj{t, 0, 0}, len(t.objs)})
				for ; ri >= 0; ri-- {
					to_close = Append(to_close, t.objs[ri])
				}
			}
			continue
		case obj_bool:
			fmt.Printf(" %v", t)
			to_close_[len(to_close_)-1].n--
		case obj_int:
			fmt.Printf(" %v", t)
			to_close_[len(to_close_)-1].n--
		case obj_real:
			fmt.Printf(" %v", t)
			to_close_[len(to_close_)-1].n--
		case obj_null:
			fmt.Printf(" null")
			to_close_[len(to_close_)-1].n--
		case obj_comment:
			fmt.Printf("%%%v", t)
			to_close_[len(to_close_)-1].n--
		case obj_eof:
			fmt.Printf("%%EOF")
			to_close_[len(to_close_)-1].n--
		case obj_xref:
			fmt.Print("\nxref\n")
			fmt.Printf("%d %d\n", int(t.id), len(t.refs))
			tot := len(t.refs) + 1 // + obj_dict
			to_close_ = Appendp(to_close_, p{obj{t, 0, 0}, tot})
			if len(t.enc) > 0 {
				to_close = Append(to_close, obj{Type: t.enc})
			}
			if len(t.refs) > 0 {
				ri := len(t.refs) - 1
				for ; ri >= 0; ri-- {
					to_close = Append(to_close, obj{Type: t.refs[ri]})
				}
			}
			continue
		case xref_ref:
			fmt.Printf("%d %d %s\n", t.n, t.m, t.c)
			to_close_[len(to_close_)-1].n--
		default:
		}
		if to_close_[len(to_close_)-1].n == 0 {
			fmt.Printf("%s", get_token_str(to_close_[len(to_close_)-1].obj, true))
			to_close_ = to_close_[:len(to_close_)-1]
		}
	}
}
