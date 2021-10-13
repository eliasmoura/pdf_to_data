package pdf

import (
	"errors"
	"fmt"
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
type obj_stream []obj    // stream\ncontent\nendstream
type obj_bool bool       // true/false
type obj_int int         // 123/-11/+23
type obj_real float64    // 01.2/-.2/+3.1
type obj_null []byte     // null
type obj_comment string  // %comment
type obj_eof string      // %%EOF

type pdf struct {
	lines []string
	line  int
	col   int
	ver   string
	objs  []obj
}

type close_obj struct {
	obj    obj   // witch obj holds the last `count` objs
	childs []obj // num of objs that should be added to the `obj`
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

var delimiters = []byte{'(', ')', '<', '>', '[', ']', '{', '}', '/', '%', ' ', '\n', ''}

func get_token(txt string, token ...interface{}) (string, int) {
	var pos int
	size := len(txt)
	txt = strings.TrimSpace(txt)
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
				return txt[0:i], pos + i
			}
		}
	}
	return txt, size + 1
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

func Get_pdf_obj(lines []string) (obj, []string, error) {
	var obj_to_close []close_obj
	is_str := false
	l := lines
	var line_index int
	var line string
	for line_index, line = range l {
		var col int
		bt_size := len(line)
		line = strings.TrimLeft(line, " \n\t")
		at_pos := bt_size - len(line)
		line = strings.TrimLeft(line, " \n\t")
		for col < len(line) {
			token, pos := get_token(line[col:])
			befor_token_len := pos - len(token)
			var objc obj
			if !is_str {
				switch token {
				case "%":
					// % defines a commemt and it goes to the end of the line
					col += pos
					if line[col:] == "%EOF" {
						objc = obj{obj_comment(line[col+1:]), line_index + 1, col + 1 + befor_token_len + at_pos}
						col = len(line)
					} else {
						objc = obj{obj_comment(line[col:]), line_index + 1, col + 1 + befor_token_len + at_pos}
						col = len(line)
					}
				case "(":
					//- strings []u8. Empty strings is valid:
					//  (liteal) may contem new lines,(),*,!,&,^,%,\),\\…\ddd(octal up to 3 digit)
					is_str = true
					o := obj{obj_strl{}, line_index + 1, col + 1 + befor_token_len + at_pos}
					obj_to_close = AppendCloseObj(obj_to_close, o)
				case ")":
					objc = obj{obj_strl{}, line_index + 1, col + 1 + befor_token_len + at_pos}
				case "<<":
					//- <<…>> denotes a dictionary like
					//  <</Type /Example >>
					o := obj{obj_dict{}, line_index + 1, col + 1 + befor_token_len + at_pos}
					obj_to_close = AppendCloseObj(obj_to_close, o)
				case ">>":
					objc = obj{obj_dict{}, line_index + 1, col + 1 + befor_token_len + at_pos}
				case "<":
					//  <hexadecimal string> ex <ab901f> if missing a digit ex<ab1>, <ab10> is assumed.
					o := obj{obj_strh(""), line_index + 1, col + 1 + befor_token_len + at_pos}
					obj_to_close = AppendCloseObj(obj_to_close, o)
					is_str = true
				case ">":
					objc = obj{obj_strh(""), line_index + 1, col + 1 + befor_token_len + at_pos}
				case "/":
					//- named objects start with the prefix / with no white spaces or delimiters
					//  they are case sensitive… /Name1 /other /@this /$$ /1.2 /aa;dd_ss**a? /.notdef are valid.
					// TODO(k0tto): Need to handle the use of characters in hex as `/GF#3A`
					//  PDF>1.2 /#13asd is valid(hexadecimal of of invalid character)
					token, pos2 := get_token(line[col+pos:])
					obj_to_close = AppendChild(obj_to_close, obj{obj_named(token), line_index + 1, col + 1 + befor_token_len + at_pos})
					pos += pos2
				case "R":
					//- obj ref, 1 0 R, Where `1 0` refers to an obj_ind
					childs, mod_id := Pop(obj_to_close[len(obj_to_close)-1].childs)
					childs, id := Pop(childs)
					obj_to_close[len(obj_to_close)-1].childs = childs

					id_val, ok1 := id.Type.(obj_int)
					mod_id_val, ok2 := mod_id.Type.(obj_int)
					if ok1 && ok2 {
						objc = obj{obj_ref{id_val, mod_id_val}, line_index + 1, col + 1 + befor_token_len + at_pos}
					} else {
						log.Printf("ERROR: token not an integer: id: [%T] mod: [%T]", id.Type, mod_id.Type)
					}
				case "[":
					//- [] denotes an array like [32 12.5 false (txt) /this]
					o := obj{obj_array{}, line_index + 1, col + 1 + befor_token_len + at_pos}
					obj_to_close = AppendCloseObj(obj_to_close, o)
				case "]":
					objc = obj{obj_array{}, line_index + 1, col + 1 + befor_token_len + at_pos}
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
						o := obj{obj_ind{id_val, mod_id_val, nil}, line_index + 1, col + 1 + befor_token_len + at_pos}
						obj_to_close = AppendCloseObj(obj_to_close, o)
					} else {
						log.Print("ERROR: token not an integer: 1", ok1, "2", ok2)
					}
				case "endobj":
					objc = obj{obj_ind{}, line_index + 1, col + 1 + befor_token_len + at_pos}
					break
				case "stream":
					//- the content that will be displayed to in the page
					o := obj{obj_stream{}, line_index + 1, col + 1 + befor_token_len + at_pos}
					obj_to_close = AppendCloseObj(obj_to_close, o)
					break
				case "endstream":
					objc = obj{obj_stream{}, line_index + 1, col + 1 + befor_token_len + at_pos}
					break
				case "false":
					//- boolean false
					obj_to_close = AppendChild(obj_to_close, obj{obj_bool(false), line_index + 1, col + 1 + befor_token_len + at_pos})
				case "true":
					//- boolean true
					obj_to_close = AppendChild(obj_to_close, obj{obj_bool(true), line_index + 1, col + 1 + befor_token_len + at_pos})
				case "null":
					//- null obj
					obj_to_close = AppendChild(obj_to_close, obj{obj_null(nil), line_index + 1, col + 1 + befor_token_len + at_pos})
				default:
					//- numbers 10 +12 -12 0 32.5 -.1 +21.0 4. 0.0
					//  if the interger exceeds the limit it is converted to a real(float)
					//  interger is auto converted to real when needed
					num_int, err := strconv.ParseInt(token, 10, 0)
					var obj_num obj
					if err != nil {
						num_float, err := strconv.ParseFloat(token, 0)
						if err == nil {
							obj_num = obj{obj_real(num_float), line_index + 1, col + 1 + befor_token_len + at_pos}
							obj_to_close = AppendChild(obj_to_close, obj_num)
						}
					} else {
						obj_num = obj{obj_int(num_int), line_index + 1, col + 1 + befor_token_len + at_pos}
						obj_to_close = AppendChild(obj_to_close, obj_num)
					}
				}
			} else {
				pdf_str := obj_to_close[len(obj_to_close)-1].obj
				switch str := pdf_str.Type.(type) {
				case obj_strl:
					if token == "(" {
						str.to_balance++
					}
					if token != ")" {
						str.str += line[col:col+befor_token_len] + token
						obj_to_close[len(obj_to_close)-1].obj = obj{str, pdf_str.line, pdf_str.col}
					} else {
						if str.to_balance == 0 {
							objc = obj{str, pdf_str.line + 1, pdf_str.col + 1 + befor_token_len + at_pos}
						} else {
							str.str += line[col:col+befor_token_len] + token
							obj_to_close[len(obj_to_close)-1].obj = obj{str, pdf_str.line, pdf_str.col}
							str.to_balance--
						}
					}

				case obj_strh:
					if token != ">" {
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
						str += obj_strh(fstr)
						obj_to_close[len(obj_to_close)-1].obj = obj{str, pdf_str.line, pdf_str.col}
					} else {
						objc = obj{str, pdf_str.line + 1, pdf_str.col + 1 + befor_token_len + at_pos}
					}
				default:
					log.Fatalf("is_str= %v, but obj is: %T", is_str, str)
				}
			}

			closed := false
			var closed_obj obj
			if objc.Type != nil {
				var oc close_obj
				switch objc.Type.(type) {
				case obj_strh:
					obj_to_close, oc = RemoveCloseObj(obj_to_close)
					_, ok := oc.obj.Type.(obj_strh)
					if ok {
						closed = true
						closed_obj = oc.obj
						is_str = false
					} else {
						log.Fatalf("ERROR: objc: %T, obj_to_close[last]: %T\n", objc.Type, oc.obj.Type)
					}
				case obj_strl:
					obj_to_close, oc = RemoveCloseObj(obj_to_close)
					_, ok := oc.obj.Type.(obj_strl)
					if ok {
						is_str = false
						closed = true
						closed_obj = oc.obj
					} else {
						log.Fatalf("ERROR: objc: %T, obj_to_close[last]: %T\n", objc.Type, oc.obj.Type)
					}
				case obj_dict:
					obj_to_close, oc = RemoveCloseObj(obj_to_close)
					o, ok := oc.obj.Type.(obj_dict)
					if ok {
						childs := oc.childs
						if (len(childs) % 2) != 0 {
							log.Print("ERROR: dictionary is not even: ")
							log.Printf("%v\n", oc)
						}
						for i := 0; i < len(childs); i += 2 {
							o = append_pair(o, obj_pair{childs[i], childs[i+1]})
						}
						closed = true
						closed_obj = obj{o, oc.obj.line, oc.obj.col}
					} else {
						log.Fatalf("+ERROR: objc: %T, obj_to_close[last]: %T\n", objc.Type, oc.obj.Type)
					}
				case obj_ind:
					obj_to_close, oc = RemoveCloseObj(obj_to_close)
					o, ok := oc.obj.Type.(obj_ind)
					if ok {
						childs := oc.childs
						for _, c := range childs {
							o.objs = Append(o.objs, c)
						}
						closed = true
						closed_obj = obj{o, oc.obj.line, oc.obj.col}
					} else {
						log.Fatalf("ERROR: objc: %T, obj_to_close[last]: %T\n", objc.Type, oc.obj.Type)
					}
				case obj_stream:
					obj_to_close, oc = RemoveCloseObj(obj_to_close)
					o, ok := oc.obj.Type.(obj_stream)
					if ok {
						childs := oc.childs
						for _, c := range childs {
							o = Append(o, c)
						}
						closed = true
						closed_obj = oc.obj
					} else {
						log.Fatalf("ERROR: objc: %T, obj_to_close[last]: %T\n", objc.Type, oc.obj.Type)
					}
				case obj_array:
					obj_to_close, oc = RemoveCloseObj(obj_to_close)
					o, ok := oc.obj.Type.(obj_array)
					if ok {
						childs := oc.childs
						for _, c := range childs {
							o = Append(o, c)
						}
						closed = true
						closed_obj = obj{o, oc.obj.line + 1, oc.obj.col + 1 + befor_token_len + at_pos}
					} else {
						log.Fatalf("ERROR: objc: %T, obj_to_close[last]: %T\n", objc.Type, o)
					}
				case obj_comment:
					closed = true
					closed_obj = objc
				case obj_ref:
					closed = true
					closed_obj = objc
				default:
					log.Printf("-Warning: %d:%d ", line_index, col)
					log.Printf("Closing %T, but it is not implemented or invalid.\n", objc.Type)
				}
			}
			col += pos
			if col > len(line) {
				col = len(line)
			}
			if closed {
				if len(obj_to_close) > 0 {
					obj_to_close = AppendChild(obj_to_close, closed_obj)
				} else {
					l[line_index] = line[col:]
					return closed_obj, l[line_index:], nil
				}
			}
		}
	}
	o := obj_to_close[0].obj
	if len(obj_to_close) == 1 {
		if o.Type == nil && len(obj_to_close[0].childs) == 1 {
			return obj_to_close[0].childs[0], l[line_index:], nil
		}
	}
	if len(obj_to_close) > 1 {
		log.Fatalf("There are too many objcs left to close. len: %v", len(obj_to_close))
		return o, nil, errors.New("failed…")
	}
	return o, nil, errors.New("failed…")
}

func Parse(txt []byte) (pdf, error) {
	pdf := pdf{lines: strings.Split(string(txt), "\n")}
	header := "%PDF-" //Ex: %PDF-1.7
	if strings.HasPrefix(pdf.lines[0], header) {
		log.Fatalln("File is not a PDF format.")
	}
	if '%' == pdf.lines[1][0] &&
		pdf.lines[1][1] > 128 &&
		pdf.lines[1][2] > 128 &&
		pdf.lines[1][3] > 128 {
		// log.Println("PDF has binary content.")
		pdf.line = 2
	} else {
		pdf.line = 1
	}
	it := pdf.lines
	for len(it) > 0 {
		var o obj
		var err error
		o, it, err = Get_pdf_obj(it)
		if err != nil {
			pdf.objs = Append(pdf.objs, o)
		}
	}
	return pdf, nil
}

func print_objs(pdf pdf) {
	var indent int
	to_close := make([]obj, len(pdf.objs))
	ri := len(pdf.objs)
	for i := 0; i < len(pdf.objs); i++ {
		to_close[i] = pdf.objs[ri]
		ri--
	}
	for len(to_close) > 0 {
		for i := 0; i < indent; i++ {
			fmt.Print("  ")
		}
		switch t := to_close[len(to_close)-1].Type.(type) {
		case obj_ind:
			fmt.Println(t.id, t.mod_id, "obj")
			if len(t.objs) > 0 {
				to_close = Append(to_close, t.objs...)
			}
		case obj_array:
			fmt.Printf("[")
			if len(t) > 0 {
				ri := len(t)
				for ; ri >= 0; ri-- {
					to_close = Append(to_close, obj{Type: t[ri]})
				}
			}
		case obj_dict:
			fmt.Printf("<<")
			if len(t) > 0 {
				ri := len(t)
				for ; ri >= 0; ri-- {
					to_close = Append(to_close, obj{Type: t[ri]})
				}
			}
		case obj_pair:
			to_close = Append(to_close, obj{Type: t.value})
		}
	}
}
