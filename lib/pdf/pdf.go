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
	id       obj_int
	mod_id   obj_int
	metadata obj_dict
	stream   obj_stream
	objs     []obj
}
type obj_str string
type obj_strl string  // (Some string)
type obj_strh string  // <2ca231fd1>
type obj_named string // /NAME
type obj_pair struct {
	key   obj
	value obj
}
type obj_dict map[obj_named]obj // <</key [/value 1 2 R]>>
type obj_array []obj            // [(ds) /qq [null]]
type obj_stream struct {        // stream\ncontent\nendstream
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

type obj_bi struct {
	dict obj_dict
	data []byte
}

type pdf struct {
	ver struct {
		major, minor int
	}
	cs          ColorSpace
	color_space obj_dict
	objs        []obj
	Text        []string
	Resources   []obj_resources
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

type obj_codechar uint32
type obj_bfchar map[obj_codechar]obj_codechar
type obj_bfrange struct {
	start, end    obj_codechar
	dest_codechar obj_codechar
	dest_array    []obj_codechar
}
type obj_codespace struct {
	codespacerange [2]obj_codechar
	bfranges       []obj_bfrange // either srcCode destCode or sdcCode srcCode destCode
	bfchars        obj_bfchar    // either srcCode destCode or sdcCode srcCode destCode
}
type obj_resources struct {
	CIDSystemInfo obj_dict
	CMapName      obj_named
	CMapType      obj_int
	CodeSpace     obj_codespace
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
	txt = bytes.TrimLeft(txt, " \t\r")
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
	to_balance := 1
	for i := range txt {
		if txt[i] == ')' {
			if i-1 >= 0 && txt[i-1] == '\\' {
				if i-2 >= 0 && txt[i-2] == '\\' {
				} else {
					continue
				}
			}
			to_balance--
			if i == 0 {
				return string(txt[0]), to_balance
			}
			return string(txt[0:i]), to_balance
		} else if txt[i] == '(' && i-1 >= 0 && txt[i-1] != '\\' {
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
	return len(lines), errors.New(fmt.Sprintf("Read %d bytes, but last line ent at %d bytes\n", bread, lines[len(lines)-1].end))
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
		i++
	}
	return 0, errors.New("Coulds not find `endstream`")
}

func read_until_EI(txt []byte) (int, error) {
	i := 0
	for i < len(txt)-3 {
		if txt[i] == 'E' &&
			txt[i+1] == 'I' &&
			(txt[i+2] == ' ' || txt[i+2] == '\n' || txt[i-2] == '\t' || txt[i-2] == '\r') {
			return i, nil
		}
		i++
	}
	return 0, errors.New("Coulds not find `endstream`")
}

func AppendText(text []string, str string) []string {
	m := len(text)
	if m == cap(text) {
		newlines := make([]string, m, (m+1)*2)
		copy(newlines, text)
		text = newlines
	}
	text = text[:m+1]
	text[m] = str
	return text
}

func Parse(doc []byte, color_spacce obj_dict, resources []obj_resources) (pdf, error) {
	var obj_to_close []close_obj
	line_index := 0
	var lines []line
	bread := 0
	var result pdf
	result.color_space = color_spacce
	var start, end int
	fontfile := make([]struct {
		id       obj_int
		mod_id   obj_int
		metadata map[obj_named]obj
	},
		0, 10)
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

	var to_parse []obj_int // objs that has streams to be parsed.
	dict_begin := false    // for CID resources dict begin
	for line_index < len(lines) {
		col := 0
		line := doc[lines[line_index].start:lines[line_index].end]
		for col < len(line) {
			token, pos := get_token(line[col:])
			before_token_len := pos - len(token)
			col += before_token_len
			if len(token) == 0 {
				continue
			}
			var objc obj
			var closed_obj obj
			{
				switch token {
				case "%":
					// % defines a commemt and it goes to the end of the line
					header := []byte("%PDF-") //Ex: %PDF-1.7
					if line_index == 0 && bytes.HasPrefix(doc[:len(header)], header) {
						bread = len(header)
						ver_ := bytes.TrimSpace(doc[bread:lines[line_index].end])
						ver := bytes.Split(ver_, []byte("."))
						if len(ver) != 2 {
							return result, errors.New(fmt.Sprintf("ERROR:%d:%d: Failed to parse PDF version from `%v` is not a valid version `m.n`\n", line_index+1, len(header), doc[line_index]))
						}
						i, err := strconv.ParseInt(string(ver[0]), 10, 32)
						if err != nil {
							return result, errors.New(fmt.Sprintf("ERROR:%d:%d: Failed to parse PDF version `%v` is not an integer\n", line_index+1, len(header), ver[0]))
						}
						result.ver.major = int(i)
						i, err = strconv.ParseInt(string(ver[1]), 10, 32)
						if err != nil {
							return result, errors.New(fmt.Sprintf("ERROR:%d:%d: Failed to parse PDF version `%v` is not an integer\n", line_index+1, len(header)+len(ver[0])+1, ver[1]))
						}
						result.ver.minor = int(i)
						bread = lines[0].end + 1

						line := doc[lines[line_index+1].start:lines[line_index+1].end]
						if '%' == line[0] &&
							line[1] > 128 &&
							line[2] > 128 &&
							line[3] > 128 {
							// "PDF has binary content."
							bread += len(line)
							bread++
							line_index++
						}
						col = len(doc[lines[0].start:lines[0].end])
						continue
					}

					if line[col+1] == '%' &&
						line[col+2] == 'E' &&
						line[col+3] == 'O' &&
						line[col+4] == 'F' {
						col++
						bread++
						token = string(line[col:])
						objc = obj{obj_eof("EOF"), line_index + 1, col + 1 + before_token_len}
						closed_obj = objc
						col = len(line)

						if len(obj_to_close) == 1 {
							o_xref := obj_to_close[0].obj
							_, ok := o_xref.Type.(obj_xref)
							if ok && len(obj_to_close[0].childs) > 0 {
								var oc close_obj
								obj_to_close, oc = RemoveCloseObj(obj_to_close)
								var err error
								o_xref.Type, err = handle_xref(oc.childs)
								result.objs = append(result.objs, o_xref)
								if !ok {
									return result, errors.New(fmt.Sprintf("ERROR: expected interger, found %s\n!", err))
								}

							} else {

								for _, o := range obj_to_close[len(obj_to_close)-1].childs {
									result.objs = Append(result.objs, o)
								}
								obj_to_close, _ = RemoveCloseObj(obj_to_close)
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
					str := obj_strl("")
					o := obj{str, line_index + 1, col + 1 + before_token_len}

					var balance int
					col++
					token, balance = read_strl(doc[lines[line_index].start+col:])
					token_ := strings.ReplaceAll(token, "\\(", "(")
					token_ = strings.ReplaceAll(token_, "\\)", ")")
					token_ = strings.ReplaceAll(token_, "\\\\", "\\")
					str = obj_strl(token_)
					o.Type = str
					if balance > 0 {
						return result, errors.New(fmt.Sprintf("ERROR:%d:%d expected token `)`, found EOF\n", o.line, o.col))
					}
					col++
					if len(obj_to_close) > 0 {
						obj_to_close = AppendChild(obj_to_close, o)
					} else {
						result.objs = Append(result.objs, o)
					}
				case ")":
					objc = obj{obj_strl(""), line_index + 1, col + 1 + before_token_len}
				case "<<":
					//- <<…>> denotes a dictionary like
					//  <</Type /Example >>
					o := obj{obj_dict{}, line_index + 1, col + 1 + before_token_len}
					obj_to_close = AppendCloseObj(obj_to_close, o)
				case ">>":
					o := obj_to_close[len(obj_to_close)-1].obj
					dict, ok := o.Type.(obj_dict)
					if !ok {
						log.Fatalf("+ERROR:%d:%d Expected `%v`, found `>>`.\n", o.line, o.col, typeStr(o))
					}
					var oc close_obj
					obj_to_close, oc = RemoveCloseObj(obj_to_close)
					childs := oc.childs
					if (len(childs) % 2) != 0 {
						log.Print("ERROR: dictionary is not even: ")
						log.Printf("%v\n", oc)
					}
					is_font_metadata := false
					is_color_space := false
					for i := 0; i < len(childs); i += 2 {
						o_key := childs[i]
						key, ok := o_key.Type.(obj_named)
						if ok {
							if key == "ColorSpace" {
								is_color_space = true
							}
							if bytes.HasPrefix([]byte(key), []byte("FontFile")) {
								ref, ok := childs[i+1].Type.(obj_ref)
								if ok {
									is_font_metadata = true
									fontfile = fontfile[:len(fontfile)+1]
									fontfile[len(fontfile)-1].id = ref.id
									fontfile[len(fontfile)-1].mod_id = ref.mod_id
								}
							}
							dict[key] = childs[i+1]
						} else {
							log.Printf("ERROR:%d:%d key `%v` should be obj_named???\n", o_key.line, o_key.col, o_key)
						}
					}
					if is_font_metadata {
						fontfile[len(fontfile)-1].metadata = dict
					}
					if is_color_space {
						result.color_space = dict
					}
					closed_obj = obj{dict, oc.obj.line, oc.obj.col}
				case "<":
					//  <hexadecimal string> ex <ab901f> if missing a digit ex<ab1>, <ab10> is assumed.
					o := obj{obj_strh(""), line_index + 1, col + 1 + before_token_len}
					col++
					var err error
					token, err = read_strh(doc[lines[line_index].start+col:])
					if err != nil {
						log.Println(string(doc))
						log.Fatalf("ERROR:%d:%d expected token `>`, found EOF\n", o.line, o.col)
					}
					var oj obj
					if len(obj_to_close) > 0 {
						oj = obj_to_close[len(obj_to_close)-1].obj
					}
					switch oj.Type {
					case "beginbfchar", "beginbfrange", "begincodespacerange":
						val, err := strconv.ParseUint(token, 16, 32)
						if err != nil {
							log.Printf("ERRO:%d:%d Cound not Parse `%s` in hexadecimal codepoint.\n", line_index+1, col+1, token)
						}
						obj_to_close = AppendChild(obj_to_close, obj{obj_codechar(val), line_index + 1, col + 1})
					default:
						strh := token
						size := len(strh)
						if len(resources) == 0 {
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
									log.Printf("ERRO:%d:%d Cound not Parse `%s` in hexadecimal string: %s\n", line_index+1, col+1+i, strh[it:it+1], strh)
								}
							}
							s := make([]string, len(shex))
							for i := range shex {
								s[i] = fmt.Sprintf("%c", shex[i])
							}
							fstr := strings.Join(s, "")
							o.Type = obj_strh(fstr)
							closed_obj = o
						} else { // use the resources for the encoding
							// NOTE(elias): assuming character has 16bits
							size = size / 4
							shex := make([]int64, 0, size)
							for it := 0; it < len(strh); it += 4 {
								char, err := strconv.ParseInt(strh[it:it+4], 16, 32)
								found := false
								for _, res := range resources {
									if c, ok := res.CodeSpace.bfchars[obj_codechar(char)]; ok {
										char = int64(c)
										found = true
										break
									}
								}
								if !found {
									for _, res := range resources {
										for i := range res.CodeSpace.bfranges {
											if int64(res.CodeSpace.bfranges[i].start) <= char && char <= int64(res.CodeSpace.bfranges[i].end) {
												if len(res.CodeSpace.bfranges[i].dest_array) > 0 {
													fmt.Println("So… this PDF is using bgrangs array. I don't know how to handle that yet.")
													char += int64(res.CodeSpace.bfranges[i].dest_array[0])
												} else {
													char += int64(res.CodeSpace.bfranges[i].dest_codechar)
												}
												found = true
												break
											}
										}
									}
								}
								if char&0xffff0000 == 0 {
									shex = append(shex, char)
								} else {
									shex = append(shex, char>>16)
									shex = append(shex, char&0x0000ffff)
								}
								if err != nil {
									log.Printf("ERRO:%d:%d Cound not Parse `%s` in hexadecimal string: %s\n", line_index+1, col+1+it, strh[it:it+1], strh)
								}
							}
							s := make([]string, len(shex))
							for i := range shex {
								s[i] = string(rune(shex[i]))
							}
							fstr := strings.Join(s, "")
							o.Type = obj_strh(fstr)
							closed_obj = o
						}
					}
					col++
				case ">":
					objc = obj{obj_strh(""), line_index + 1, col + 1 + before_token_len}
				case "/":
					//- named objects start with the prefix / with no white spaces or delimiters
					//  they are case sensitive… /Name1 /other /@this /$$ /1.2 /aa;dd_ss**a? /.notdef are valid.
					// TODO(k0tto): Need to handle the use of characters in hex as `/GF#3A`
					//  PDF>1.2 /#13asd is valid(hexadecimal of invalid character)
					col++
					token, pos = get_token(line[col:])
					obj_to_close = AppendChild(obj_to_close, obj{obj_named(token), line_index + 1, col + 1 + before_token_len})
				case "R":
					childs, mod_id := Pop(obj_to_close[len(obj_to_close)-1].childs)
					childs, id := Pop(childs)
					obj_to_close[len(obj_to_close)-1].childs = childs

					id_val, ok1 := id.Type.(obj_int)
					mod_id_val, ok2 := mod_id.Type.(obj_int)
					if !ok1 || !ok2 {
						log.Printf("ERROR: token not an integer: id: [%T] mod: [%T]", id.Type, mod_id.Type)
					}
					objc = obj{obj_ref{id_val, mod_id_val}, line_index + 1, col + 1 + before_token_len}
					closed_obj = objc
				case "[":
					//- [] denotes an array like [32 12.5 false (txt) /this]
					o := obj{obj_array{}, line_index + 1, col + 1 + before_token_len}
					obj_to_close = AppendCloseObj(obj_to_close, o)
				case "]":
					objc = obj{obj_array{}, line_index + 1, col + 1 + before_token_len}
					oc := obj_to_close[len(obj_to_close)-1]
					o, ok := oc.obj.Type.(obj_array)
					if !ok {
						log.Fatalf("ERROR: objc: %T, obj_to_close[last]: %T\n", objc.Type, oc.obj.Type)
					}
					obj_to_close, oc = RemoveCloseObj(obj_to_close)
					childs := oc.childs
					for _, c := range childs {
						o = Append(o, c)
					}
					closed_obj = obj{o, oc.obj.line + 1, oc.obj.col + 1 + before_token_len}
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
					if !ok1 || !ok2 {
						log.Print("ERROR: token not an integer: 1", ok1, "2", ok2)
					}
					o := obj{obj_ind{id: id_val, mod_id: mod_id_val, objs: nil}, line_index + 1, col + 1 + before_token_len}
					obj_to_close = AppendCloseObj(obj_to_close, o)
				case "endobj":
					objc = obj{obj_ind{}, line_index + 1, col + 1 + before_token_len}
					oc := obj_to_close[len(obj_to_close)-1]
					ind, ok := oc.obj.Type.(obj_ind)
					if !ok {
						log.Fatalf("ERROR:%d:%d found `endobj`, expected %s\n", line_index+1, col+1, typeStr(oc.obj))
					}
					obj_to_close, oc = RemoveCloseObj(obj_to_close)
					childs := oc.childs
					// Assuming the first dict is a dictionary with metadata
					// TODO(elias): take a look and make sure this code does what the comment says.
					for _, c := range childs {
						switch t := c.Type.(type) {
						case obj_dict:
							ind.metadata = t
							oc.obj.Type = ind
						case obj_stream:
							ind.stream = t
						default:
							ind.objs = Append(ind.objs, c)
						}
					}
					closed_obj = obj{ind, oc.obj.line, oc.obj.col}
					break
				case "stream":
					//- the content that will be displayed to in the page

					o_ind := obj_to_close[len(obj_to_close)-1].obj
					ind, ok_ind := o_ind.Type.(obj_ind)
					var stream_decoded []byte
					// Assume the stream data start in a new line.
					line_index++
					end_stream, _ := get_endstream(doc[lines[line_index].start:])
					var stream_encoded []byte
					if doc[lines[line_index].start+end_stream-1] == '\n' {
						end_stream--
					}
					if ok_ind {
						if end_stream > 0 {
							childs := obj_to_close[len(obj_to_close)-1].childs
							o_dict := childs[len(childs)-1]
							metadata, ok := o_dict.Type.(obj_dict)
							_, ok_stype := metadata[obj_named("Subtype")].Type.(obj_named)
							//NOTE(elias): assuming that the content stream metadata
							// does not contain Type or Subtype fields.
							if ok && !ok_stype {
								o_filter := metadata[obj_named("Filter")]
								if o_filter.Type != nil {
									stream_encoded = doc[lines[line_index].start : lines[line_index].start+end_stream]
								} else {
									stream_decoded = doc[lines[line_index].start : lines[line_index].start+end_stream]
								}
								to_parse = append(to_parse, obj_int(len(result.objs))) // index of the stream I need to decode.
							}
						}
					}
					o := obj{obj_stream{encoded_content: stream_encoded, decoded_content: stream_decoded, objs: nil}, line_index + 1, col + 1 + before_token_len}

					var err error
					line_index, err = index_from_bread(lines, lines[line_index].start+end_stream)
					if err != nil {
						log.Println(err)
					}
					obj_to_close = AppendChild(obj_to_close, o)
					// NOTE(elias): start using the metadata/stream fields in the struct
					if ok_ind {
						ind.stream = obj_stream{encoded_content: stream_encoded, decoded_content: stream_decoded, objs: nil}
					}

				case "endstream":
					//everything is proccessed in after the `stream` token.
				case "def":
					cspacerange, ok := obj_to_close[len(obj_to_close)-1].obj.Type.(obj_resources)
					if !ok {
						_str := fmt.Sprintf("Expected %s, found `def`\n", typeStr(obj_to_close[len(obj_to_close)-1].obj))
						log.Printf(_str)
						return result, errors.New(_str)
					}

					childs, o_value := Pop(obj_to_close[len(obj_to_close)-1].childs)
					childs, o_key := Pop(childs)
					obj_to_close[len(obj_to_close)-1].childs = childs
					key, ok := o_key.Type.(obj_named)
					if !ok {
						log.Print("ERROR:def token not an named")
					}
					switch key {
					case "CIDSystemInfo":
						dict, ok := o_value.Type.(obj_dict)
						if !ok {
							log.Print("ERROR:def dict token not a dict: ", o_value)
							return result, errors.New("ERROR:def dict token not a dict: ")
						}
						cspacerange.CIDSystemInfo = dict
					case "CMapName":
						str, ok := o_value.Type.(obj_named)
						if !ok {
							log.Print("ERROR:def dict token not a obj_named")
							return result, errors.New("ERROR:def dict token not a obj_named")
						}
						cspacerange.CMapName = str
					case "CMapType":
						i, ok := o_value.Type.(obj_int)
						if !ok {
							log.Print("ERROR:def dict token not a obj_int")
							return result, errors.New("ERROR:def dict token not a obj_int")
						}
						cspacerange.CMapType = i
					default:
						log.Printf("ERROR:def token unkown %s\n", key)
						return result, errors.New("ERROR:def token unkown")
					}
				case "pop":
					//TODO(elias): find out what this should be doing exactly.
					childs := obj_to_close[len(obj_to_close)-1].childs
					childs, o_defineresource := Pop(childs)
					childs, o_cmap := Pop(childs)
					childs, o_current := Pop(childs)
					childs, o_cmapname := Pop(childs)
					if defineresource, ok := o_defineresource.Type.(string); !ok || defineresource != "defineresource" {
						log.Printf("ERROR:pop expected defineresource, found %v\n", o_defineresource)
						return result, errors.New("ERROR:def token unkown")
					}
					if cmap, ok := o_cmap.Type.(obj_named); !ok || cmap != "CMap" {
						log.Printf("ERROR:pop expected CMap, found %v\n", o_cmap)
						return result, errors.New("ERROR:def token unkown")
					}
					if currentdict, ok := o_current.Type.(string); !ok || currentdict != "currentdict" {
						log.Printf("ERROR:pop expected currentdict, found %v\n", o_current)
						return result, errors.New("ERROR:def token unkown")
					}
					if cmapname, ok := o_cmapname.Type.(string); !ok || cmapname != "CMapName" {
						log.Printf("ERROR:pop not defineresource %v\n", o_defineresource)
						return result, errors.New("ERROR:def token unkown")
					}
					obj_to_close[len(obj_to_close)-1].childs = childs
				case "beginbfchar", "beginbfrange", "begincodespacerange":
					obj_to_close[len(obj_to_close)-1].childs, _ = Pop(obj_to_close[len(obj_to_close)-1].childs)
					obj_to_close = AppendCloseObj(obj_to_close, obj{token, line_index + 1, col + 1})
				case "begin":
					childs, o_res := Pop(obj_to_close[len(obj_to_close)-1].childs)
					key, ok := o_res.Type.(string)
					if !ok {
						log.Print("ERROR:begin token not an named")
						log.Printf("%v", o_res)
					}
					switch key {
					case "dict":
						childs, o_value := Pop(obj_to_close[len(obj_to_close)-1].childs)
						obj_to_close[len(obj_to_close)-1].childs = childs
						_, ok := o_value.Type.(obj_int) // there is no use for this for now?
						if ok {
							log.Print("ERROR:begin dict dict token not an obj_int")
							return result, errors.New("ERROR:def dict token not an obj_int")
						}
						dict_begin = true
					case "findresource":
						childs, o_named2 := Pop(childs)
						childs, o_named1 := Pop(childs)
						obj_to_close[len(obj_to_close)-1].childs = childs
						_, ok := o_named2.Type.(obj_named) // there is no use for this for now?
						if !ok {
							log.Print("ERROR:begin findresource token not a obj_named")
							return result, errors.New("ERROR:dbegin findresource token not a obj_named")
						}
						_, ok = o_named1.Type.(obj_named) // there is no use for this for now?
						if !ok {
							log.Print("ERROR:begin findresource  token not a obj_named")
							return result, errors.New("ERROR:begin findresource token not a obj_named")
						}
						obj_to_close = AppendCloseObj(obj_to_close, obj{obj_resources{}, line_index + 1, col + 1})
					default:
						log.Printf("ERROR:begin token unkown %s\n", key)
						return result, errors.New("ERROR:begin findresource token not a obj_named")
					}
				case "endcmap", "begincmap":
				case "end":
					if !dict_begin {
						var oc close_obj
						obj_to_close, oc = RemoveCloseObj(obj_to_close)
						resource, ok := oc.obj.Type.(obj_resources)
						if !ok {
							log.Panicln("Not REsources!")
						}
						result.Resources = append(result.Resources, resource)
					}
					dict_begin = false
				case "endbfrange":
					endbfchar, ok := obj_to_close[len(obj_to_close)-1].obj.Type.(string)
					if !ok || endbfchar != "beginbfrange" {
						_str := fmt.Sprintf("Expected %s(%v), found `endbfrange`\n", typeStr(obj_to_close[len(obj_to_close)-1].obj), obj_to_close[len(obj_to_close)-1].obj)
						log.Printf(_str)
						return result, errors.New(_str)
					}
					var oc close_obj
					obj_to_close, oc = RemoveCloseObj(obj_to_close)

					childs := oc.childs
					if len(childs)%3 != 0 {
						_str := fmt.Sprintf("bfchar should olnly contain a three pdf obj of char codepoints\n%v\n", childs)
						log.Printf(_str)
						return result, errors.New(_str)
					}
					bfranges := make([]obj_bfrange, len(childs)/3)

					for len(childs) > 0 {
						var o_bfchar1, o_bfchar2, o_bfchar3 obj
						childs, o_bfchar3 = Pop(childs)
						childs, o_bfchar2 = Pop(childs)
						childs, o_bfchar1 = Pop(childs)

						bfchar1, ok1 := o_bfchar1.Type.(obj_codechar)
						bfchar2, ok2 := o_bfchar2.Type.(obj_codechar)
						if !ok1 || !ok2 {
							log.Panicln("ERROR: token not an codechar: 1", ok1, o_bfchar1, o_bfchar2)
						}
						bfranges = append(bfranges, obj_bfrange{start: bfchar1, end: bfchar2})
						switch v := o_bfchar3.Type.(type) {
						case obj_codechar:
							bfranges[len(bfranges)-1].dest_codechar = v
						case obj_array:
							for i := range v {
								if codechar, ok := v[i].Type.(obj_codechar); ok {
									bfranges[len(bfranges)-1].dest_array = append(bfranges[len(bfranges)-1].dest_array, codechar)
								}
							}
						default:
							log.Panicf("Failend to match the obj for %v, in the bfrange\n", o_bfchar3.Type)
						}
					}
				case "endbfchar":
					endbfchar, ok := obj_to_close[len(obj_to_close)-1].obj.Type.(string)
					if !ok || endbfchar != "beginbfchar" {
						_str := fmt.Sprintf("Expected %s(%v), found `endbfchar`\n", typeStr(obj_to_close[len(obj_to_close)-1].obj), obj_to_close[len(obj_to_close)-1].obj)
						log.Printf(_str)
						return result, errors.New(_str)
					}

					var oc close_obj
					obj_to_close, oc = RemoveCloseObj(obj_to_close)
					childs := oc.childs
					if len(childs)%2 != 0 {
						_str := fmt.Sprintf("bfchar should olnly contain a key value sequence of char codepoints\n%v\n", childs)
						log.Printf(_str)
						return result, errors.New(_str)
					}

					bfchars := make(obj_bfchar, len(childs)/2)
					for len(childs) > 0 {
						var o_bfchar1, o_bfchar2 obj
						childs, o_bfchar2 = Pop(childs)
						childs, o_bfchar1 = Pop(childs)
						bfchar1, ok1 := o_bfchar1.Type.(obj_codechar)
						bfchar2, ok2 := o_bfchar2.Type.(obj_codechar)
						if !ok1 || !ok2 {
							log.Print("ERROR: token not an codechar: 1", ok1, o_bfchar1, o_bfchar2)
						}
						bfchars[bfchar1] = bfchar2
					}

					cspacerange, ok := obj_to_close[len(obj_to_close)-1].obj.Type.(obj_resources)
					if !ok {
						_str := fmt.Sprintf("Expected %s, found `endconge`\n", typeStr(obj_to_close[len(obj_to_close)-1].obj))
						log.Printf(_str)
						return result, errors.New(_str)
					}
					cspacerange.CodeSpace.bfchars = bfchars
					obj_to_close[len(obj_to_close)-1].obj.Type = cspacerange
				case "endcodespacerange":
					endcoderange, ok := obj_to_close[len(obj_to_close)-1].obj.Type.(string)
					if !ok || endcoderange != "begincodespacerange" {
						_str := fmt.Sprintf("Expected %s(%v), found `endcodespacerange`\n", typeStr(obj_to_close[len(obj_to_close)-1].obj), obj_to_close[len(obj_to_close)-1].obj)
						log.Printf(_str)
						return result, errors.New(_str)
					}
					var oc close_obj
					obj_to_close, oc = RemoveCloseObj(obj_to_close)

					childs, o_crange2 := Pop(oc.childs)
					childs, o_crange1 := Pop(childs)
					oc.childs = childs

					crange1, ok1 := o_crange1.Type.(obj_codechar)
					crange2, ok2 := o_crange2.Type.(obj_codechar)
					if !ok1 || !ok2 {
						log.Println("ERROR: token not an named: 1", ok1)
						log.Println(o_crange1)
						log.Println(o_crange2)
						return result, errors.New("ERROR: token not an named: 1")
					}

					cspacerange, ok := obj_to_close[len(obj_to_close)-1].obj.Type.(obj_resources)
					if !ok {
						_str := fmt.Sprintf("Expected %s, found `endcoderange`\n", typeStr(obj_to_close[len(obj_to_close)-1].obj))
						log.Printf(_str)
						return result, errors.New(_str)
					}
					cspacerange.CodeSpace.codespacerange[0] = crange1
					cspacerange.CodeSpace.codespacerange[1] = crange2
					obj_to_close[len(obj_to_close)-1].obj.Type = cspacerange
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
					// PDF operators
					// for more information look at lib/pdf/operator.go
					//NOTE(elias): The tokens below doesn't seems to be necessary when this program
					// doesn't care to display anything.
				case "w", "J", "j", "M", "d", "ri", "i", "gs",
					"q", "Q", "cm",
					"Do",
					"MP", "DP", "BMC", "BDC", "EMC",
					"BX", "EX",
					"m", "l", "c", "v", "y", "h", "re",
					"S", "s", "F", "f*", "B", "B*", "b", "b*",
					"W", "W*",
					"BT", "ET",
					"Tc", "Tw", "Tz", "TL", "Tf", "Tr", "Ts",
					"Td", "TD", "Tm", "T*",
					"Tj", "TJ", "'", "\"",
					"d0", "d1",
					"CS", "cs", "SC", "SCN", "sc", "scn", "G", "g", "RG", "rg", "K", "k",
					"sh":
					if len(obj_to_close) > 0 {
						childs := obj_to_close[len(obj_to_close)-1].childs
						var err error
						childs, err = handle_operator(childs, token, result.color_space)
						obj_to_close[len(obj_to_close)-1].childs = childs
						if err != nil {
							return result, err
						}
					}
				case "f", "n":
					_, ok := obj_to_close[len(obj_to_close)-1].obj.Type.(obj_xref)
					if ok {
						obj_to_close = AppendChild(obj_to_close, obj{token, line_index + 1, col + 1 + before_token_len})
						break
					}
				case "BI":
					obj_to_close = AppendCloseObj(obj_to_close, obj{obj_bi{}, 0, 0})
				case "ID":
					// NOTE(elias): inline iamge. It is analogus to an obj_stream with BI ID EI.
					// This will most likely be a source o problems later…
					if len(obj_to_close) > 0 {
						_, ok := obj_to_close[len(obj_to_close)-1].obj.Type.(obj_bi)
						if ok {
							ei, _ := read_until_EI(doc[lines[line_index].start+col:])
							var err error
							line_index, err = index_from_bread(lines, lines[line_index].start+ei)
							if err != nil {
								log.Println(err)
								return result, err
							}
						}
					}
					col = len(line)
					obj_to_close, _ = RemoveCloseObj(obj_to_close)
					continue
				case "EI":
				default:
					//- numbers 10 +12 -12 0 32.5 -.1 +21.0 4. 0.0
					//  if the interger exceeds the limit it is converted to a real(float)
					//  interger is auto converted to real when needed
					{
						num_int, err := strconv.ParseInt(token, 10, 0)
						if err == nil {
							obj_num := obj{obj_int(num_int), line_index + 1, col + 1 + before_token_len}
							obj_to_close = AppendChild(obj_to_close, obj_num)
							break
						}

						if err != nil {
							num_float, err := strconv.ParseFloat(token, 0)
							if err == nil {
								obj_num := obj{obj_real(num_float), line_index + 1, col + 1 + before_token_len}
								obj_to_close = AppendChild(obj_to_close, obj_num)
								break
							}

							if err != nil { // should only be used by xref? last character n and f
								// TODO(elias): check why some stream get here.
								// NOTE(elias): when there is a resource stream, those streing show up.
								// add everything and check latter what it is.
								// case "beginbfchar", "beginbfrange", "begincoderange", "findresource", "CMapName", "currentdict", "defineresource", "dict":
								obj_to_close = AppendChild(obj_to_close, obj{token, line_index + 1, col + 1})
							}
						}
					}
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
					result.objs = Append(result.objs, closed_obj)
				}
			}
		}
		bread += col + 1
		line_index++
	}
	if len(obj_to_close) > 0 {
		if result.ver.major != 0 {
			return result, errors.New(fmt.Sprintf("%%%%EOF found, expected token %v\n", get_token_str(obj_to_close[len(obj_to_close)-1].obj, true)))
		} else {
			result.objs = append(result.objs, obj_to_close[0].childs...)
		}
	}

	//add a metadata to the FontFile obj
	for _, f := range fontfile {
		for i := range result.objs {
			ind, ok := result.objs[i].Type.(obj_ind)
			if ok && f.id == ind.id {
				for key, val := range f.metadata {
					ind.metadata[key] = val
				}
				result.objs[i].Type = ind
			}
		}
	}

	//find resources
	if len(to_parse) > 0 {
		var index int
		for index < len(to_parse) {
			i := to_parse[index]
			o_ind := result.objs[i]
			ind, ok := o_ind.Type.(obj_ind)
			if ok {
				dict := ind.metadata
				ref, ok_length := dict["Length"].Type.(obj_ref)
				var length int
				if !ok_length {
					length = len(ind.stream.encoded_content)
					if length == 0 {
						length = len(ind.stream.decoded_content)
					}
					ok_length = true
				} else {
					o_ind_length, err := get_obj_by_id(result.objs, ref.id)
					if err != nil {
						log.Fatalf("failed to get id for delayed stream decode length id: %v\n", ind)
					}
					ind_length, _ := o_ind_length.Type.(obj_ind)
					length_, ok_ := ind_length.objs[len(ind_length.objs)-1].Type.(obj_int)
					ok_length = ok_
					length = int(length_)
				}
				var Type string
				t, _ok := ind.metadata["Type"].Type.(obj_named)
				if _ok {
					Type = string(t)
				}
				if ok_length && len(ind.stream.decoded_content) == 0 {
					if len(ind.stream.encoded_content) > 0 && string(t) != "Metadata" {
						str := ind.stream.encoded_content
						if int(length) != len(str) {
							log.Fatalln("failed to get id for delayed stream decode; length mismatch")
						}
						r, err := zlib.NewReader(bytes.NewReader(str))
						if err != nil {
							return result, errors.New(fmt.Sprintf("failled to decode stream of obj %d:%d %v", ind.id, ind.mod_id, err))
						}
						ind.stream.decoded_content, err = io.ReadAll(r)
						if err != nil {
							return result, errors.New(fmt.Sprintf("failled to readall:%d: %v", line_index+1, err))
						}
					}
					result.objs[i].Type = ind
				}
				{
					if len(Type) < 0 || (Type != "FontDescriptor" && Type != "Metadata" && Type != "XRef" && !strings.HasPrefix(Type, "FontFile")) {
						_pdf, err := Parse(ind.stream.decoded_content, result.color_space, result.Resources)

						if err != nil && err.Error() != "SKIP" {
							log.Printf("%s", err)
							return result, err
						}
						ind.stream.objs = _pdf.objs
						result.objs[i].Type = ind
						if len(_pdf.Resources) > 0 {
							for _, r := range _pdf.Resources {
								result.Resources = append(result.Resources, r)
							}
							newindex := make([]obj_int, len(to_parse)-1)
							copy(newindex, to_parse[:index])
							copy(newindex[index:], to_parse[index+1:])
							to_parse = newindex
							index = 0
							continue
						}
					}
				}
			}
			index++
		}
		for _, o := range result.objs {
			if ind, ok := o.Type.(obj_ind); ok {
				for _, _o := range ind.stream.objs {
					switch t := _o.Type.(type) {
					case obj_str:
						result.Text = AppendText(result.Text, strings.TrimSpace(string(t)))
					case obj_strl:
						result.Text = AppendText(result.Text, strings.TrimSpace(string(t)))
					case obj_strh:
						result.Text = AppendText(result.Text, strings.TrimSpace(string(t)))
					case obj_int, obj_real:
					default:
					}
				}
			}
		}
	}

	return result, nil
}

func get_obj(objs []obj, o string) (obj, error) {
	for _, _o := range objs {
		if typeStr(_o) == o {
			return _o, nil
		}
	}
	return obj{}, errors.New("Could not find obj")
}
func get_obj_by_id(objs []obj, id obj_int) (obj, error) {
	var o obj
	for _, _o := range objs {
		ind, ok := _o.Type.(obj_ind)
		if ok && ind.id == id {
			return _o, nil
		}
	}
	return o, errors.New("Could not find obj")
}

func init() {
	log.SetFlags(log.Lshortfile | log.Lmsgprefix)
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
			fmt.Println(len(t.stream.decoded_content))
			if len(t.objs) > 0 && len(t.stream.decoded_content) > 0 {
				ri := len(t.objs) - 1
				fmt.Println("Has COntent")
				fmt.Println(t.stream.decoded_content)
				to_close_ = Appendp(to_close_, p{obj{obj_ind{}, 0, 0}, len(t.stream.objs)})
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
				// ri := len(t) - 1
				to_close_ = Appendp(to_close_, p{obj{t, 0, 0}, len(t)})
				for key, val := range t {
					to_close = Append(to_close, obj{Type: obj_named(key)})
					to_close = Append(to_close, obj{Type: val})
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
			fmt.Printf("(%.10s)", string(t))
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
			// fmt.Printf("%%%v", t)
			// to_close_[len(to_close_)-1].n--
		case obj_eof:
			// fmt.Printf("%s\n", "%%EOF")
			to_close_[len(to_close_)-1].n--
		case obj_xref:
			// fmt.Print("xref\n")
			// fmt.Printf("%d %d\n", int(t.id), len(t.refs))
			// tot := len(t.refs) + 1 // + obj_dict
			// to_close_ = Appendp(to_close_, p{obj{t, 0, 0}, tot})
			// if len(t.enc) > 0 {
			// 	to_close = Append(to_close, obj{Type: t.enc})
			// }
			// if len(t.refs) > 0 {
			// 	ri := len(t.refs) - 1
			// 	for ; ri >= 0; ri-- {
			// 		to_close = Append(to_close, obj{Type: t.refs[ri]})
			// 	}
			// }
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
