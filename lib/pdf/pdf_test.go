package pdf

import (
	"fmt"
	"log"
	"strings"
	"testing"
)

// func TestHelloName(t *testing.T) {
//   name := "Elias"
//   want := regexp.MustCompile(`\b`+name+`\b`)
//   msg, err := Hello(name)
//   if want.MatchString(msg) || err != nil {
//     t.Fail(`Hello("`+name+`") = %q, %v, want match fo %#q, nil`, msg, err, want)
//   }
// }

// func TestHelloEmpty(t *testing.T) {
//   name := ""
//   msg, err := Hello(name)
//   if msg != name || err == nil {
//     t.Fail(`Hello("`+name+`") = %q, %v, want "`+name+`", error`, msg, err)
//   }
// }

func init() {
	log.SetFlags(log.Lshortfile | log.Lmsgprefix)
}

func matchObj(o1, o2 obj) bool {
	switch v1 := o1.Type.(type) {
	case obj_ind:
		v2, ok := o2.Type.(obj_ind)
		if !ok {
			log.Printf(`%T does not match %T.`, v1, v2)
			return false
		}
		if v1.id != v2.id || v1.mod_id != v2.mod_id || len(v1.objs) != len(v2.objs) {
			log.Printf(`%T(%v)[%d] does not have the same size %T(%v)[%d].`, v1, v1, len(v1.objs), v2, v2, len(v2.objs))
			return false
		}
		for i, o := range v1.objs {
			if !matchObj(o, v2.objs[i]) {
				log.Printf(`%T[%d](%v) does not match %T[%d](%v).`, v1, i, o, v2, i, v2.objs[i])
				return false
			}
		}
	case obj_ref:
		v2, ok := o2.Type.(obj_ref)
		if !ok {
			log.Printf(`%T(%v) does not match %T(%v).`, v1, v1, v2, v2)
			return false
		}
		if v1.id != v2.id && v1.mod_id != v2.mod_id {
			log.Printf(`%T(%d,%d) does not match %T(%d,%d).`, v1, v1.id, v1.mod_id, v2, v2.id, v2.mod_id)
			return false
		}
	case obj_strl:
		v2, ok := o2.Type.(obj_strl)
		if !ok {
			log.Printf(`%T does not match %T.`, v1, v2)
			return false
		}
		if v1 != v2 {
			log.Printf(`%T(%v) does not match %T(%v).`, v1, v1, v2, v2)
			return false
		}
	case obj_strh:
		v2, ok := o2.Type.(obj_strh)
		if !ok {
			log.Printf(`%T does not match %T.`, v1, v2)
			return false
		}
		if v1 != v2 {
			log.Printf(`%T(%v) does not match %T(%v).`, v1, v1, v2, v2)
			return false
		}
	case obj_eof:
		v2, ok := o2.Type.(obj_eof)
		if !ok {
			log.Printf(`%T does not match %T.`, v1, v2)
			return false
		}
	case obj_comment:
		v2, ok := o2.Type.(obj_comment)
		if !ok {
			log.Printf(`%T does not match %T.`, v1, v2)
			return false
		}
		if v1 != v2 {
			log.Printf(`%T(%v) does not match %T(%v).`, v1, v1, v2, v2)
			return false
		}
	case obj_named:
		v2, ok := o2.Type.(obj_named)
		if !ok {
			log.Printf(`%T does not match %T.`, v1, v2)
			return false
		}
		if v1 != v2 {
			log.Printf(`%T(%v) does not match %T(%v).`, v1, v1, v2, v2)
			return false
		}
	case obj_pair:
		v2, ok := o2.Type.(obj_pair)
		if !ok {
			log.Printf(`%T does not match %T.`, v1, v2)
			return false
		}
		if !matchObj(v1.key, v2.key) || !matchObj(v1.value, v2.value) {
			log.Printf(`%T(%v=%v) does not match %T(%v=%v).`, v1, v1.key, v1.value, v2, v2.key, v2.value)
			return false
		}
	case obj_dict:
		v2, ok := o2.Type.(obj_dict)
		if !ok {
			log.Printf(`%T does not match %T.`, v1, v2)
			return false
		}
		if len(v1) != len(v2) {
			log.Printf(`%T(%v)[%d] does not have the same size %T(%v)[%d].`, v1, v1, len(v1), v2, v2, len(v2))
			return false
		}
		for i, o := range v1 {
			if !matchObj(obj{Type: o}, obj{Type: v2[i]}) {
				log.Printf(`%T[%d](%v) does not match %T[%d](%v).`, v1, i, o, v2, i, v2[i])
				return false
			}
		}
	case obj_array:
		v2, ok := o2.Type.(obj_array)
		if !ok {
			log.Printf(`%T does not match %T.`, v1, v2)
			return false
		}
		if len(v1) != len(v2) {
			log.Printf(`%T(%v)[%d] does not have the same size %T(%v)[%d].`, v1, v1, len(v1), v2, v2, len(v2))
			return false
		}
		for i, o := range v1 {
			if !matchObj(o, v2[i]) {
				log.Printf(`%T[%d](%v) does not match %T[%d](%v).`, v1, i, o, v2, i, v2[i])
				return false
			}
		}
	case obj_stream:
		v2, ok := o2.Type.(obj_stream)
		if !ok {
			log.Printf(`%T does not match %T.`, v1, v2)
			return false
		}
		if len(v1.objs) != len(v2.objs) {
			log.Printf(`%T(%v)[%d] does not have the same size %T(%v)[%d].`, v1, v1, len(v1.objs), v2, v2, len(v2.objs))
			return false
		}
		for i, o := range v1.objs {
			if !matchObj(o, v2.objs[i]) {
				log.Printf(`%T[%d](%v) does not match %T[%d](%v).`, v1, i, o, v2, i, v2.objs[i])
				return false
			}
		}
	case obj_bool:
		v2, ok := o2.Type.(obj_bool)
		if !ok || v1 != v2 {
			log.Printf(`%T(%v) does not match %T(%v).`, v1, v1, v2, v2)
			return false
		}
	case obj_int:
		v2, ok := o2.Type.(obj_int)
		if !ok && v1 != v2 {
			log.Printf(`%T(%v) does not match %T(%v).`, v1, v1, v2, v2)
			return false
		}
	case obj_real:
		v2, ok := o2.Type.(obj_real)
		if !ok && v1 != v2 {
			log.Printf(`%T(%v) does not match %T(%v).`, v1, v1, v2, v2)
			return false
		}
	case xref_ref:
		v2, ok := o2.Type.(xref_ref)
		if !ok {
			log.Printf(`%T(%v) does not match %T(%v).`, v1, v1, v2, v2)
			return false
		}
		if v1.c != v2.c && v1.m != v2.m && v1.n != v2.n {
			log.Printf(`%T(%v) does not match %T(%v).`, v1, v1, v2, v2)
			return false
		}
	case obj_xref:
		v2, ok := o2.Type.(obj_xref)
		if !ok {
			log.Printf(`%T(%v) does not match %T(%v).`, v1, v1, v2, v2)
			return false
		}
		if len(v1.refs) != len(v2.refs) {
			log.Printf(`%T(%v)[%d] does not have the same size %T(%v)[%d].`, v1, v1, len(v1.refs), v2, v2, len(v2.refs))
			return false
		}
		for i, o := range v1.refs {
			if !matchObj(obj{o, 0, 0}, obj{v2.refs[i], 0, 0}) {
				log.Printf(`%T[%d](%v) does not match %T[%d](%v).`, v1, i, o, v2, i, v2.refs[i])
				return false
			}
		}
		if !matchObj(obj{v1.enc, 0, 0}, obj{v2.enc, 0, 0}) {
			log.Printf(`%T(%v) does not match %T(%v).`, v1, v1.enc, v2, v2.enc)
			return false
		}
	case obj_null:
		v2, ok := o2.Type.(obj_null)
		if !ok {
			log.Printf(`%T(%v) does not match %T(%v).`, v1, v1, v2, v2)
			return false
		}
	default:
		log.Printf("Not any PDF obj: %T %v\n", v1, v1)
		return false
	}
	return true
}

func TestLineComment(t *testing.T) {
	str := `%PDF-1.4
%Some comment`
	txt := strings.Split(str, "\n")
	c_comment := obj{Type: obj_comment(txt[1][1:])}
	log.SetPrefix("TestLineComment: ")
	pdf, err := Parse([]byte(str))
	if err != nil {
		log.Printf(`Failed to parse valid pdf %T object. %v`, c_comment.Type, err)
		t.Fail()
	}
	if !matchObj(c_comment, pdf.objs[0]) {
		t.Fail()
	}
}

func TestEOF(t *testing.T) {
	str := `%PDF-1.4
%%EOF`
	txt := strings.Split(str, "\n")
	c_eof := obj{Type: obj_eof(txt[1][2:])}
	log.SetPrefix("TestEOF: ")
	pdf, err := Parse([]byte(str))
	if err != nil {
		log.Printf(`Failed to parse valid pdf %T object. %v`, c_eof.Type, err)
		t.Fail()
	}
	if !matchObj(c_eof, pdf.objs[0]) {
		t.Fail()
	}
}

func TestInt(t *testing.T) {
	str := `%PDF-1.4
10
%%EOF`
	c_int := obj{Type: obj_int(10)}
	log.SetPrefix("TestInt: ")
	pdf, err := Parse([]byte(str))
	if err != nil {
		log.Printf(`Failed to parse valid pdf %T object. %v\n`, c_int, err)
		t.Fail()
	}
	if !matchObj(c_int, pdf.objs[0]) {
		t.Fail()
	}
}

func TestReal(t *testing.T) {
	str := `%PDF-1.4
10.5
%%EOF`
	c_real := obj{Type: obj_real(10.5)}
	log.SetPrefix("TestReal: ")
	pdf, err := Parse([]byte(str))
	if err != nil {
		log.Printf(`Failed to parse valid pdf %T object. %v\n`, c_real, err)
		t.Fail()
	}
	if !matchObj(c_real, pdf.objs[0]) {
		t.Fail()
	}
}

func TestArray(t *testing.T) {
	str := `%PDF-1.4
[0 1 2]
%%EOF`
	c_array := obj{obj_array{obj{obj_int(0), 1, 2},
		obj{obj_int(1), 1, 4}, obj{obj_int(2), 1, 6}}, 1, 1}
	log.SetPrefix("TestArray: ")
	pdf, err := Parse([]byte(str))
	if err != nil {
		log.Printf(`Failed to parse valid pdf %T object. %v\n`, c_array.Type, err)
		t.Fail()
	}
	if !matchObj(c_array, pdf.objs[0]) {
		t.Fail()
	}
}

func TestNamedObj(t *testing.T) {
	str := `%PDF-1.4
/myName
%%EOF`
	c_named := obj{obj_named("myName"), 1, 1}
	log.SetPrefix("TestNamedObj: ")
	pdf, err := Parse([]byte(str))
	if err != nil {
		log.Printf(`Failed to parse valid pdf %T object. %v\n`, c_named, err)
		t.Fail()
	}
	if !matchObj(c_named, pdf.objs[0]) {
		t.Fail()
	}
}

func TestDict(t *testing.T) {
	str := `%PDF-1.4
<</Myname /k0tto /Age 2>>
%%EOF`
	c_dict := obj{
		obj_dict{
			obj_pair{obj{obj_named("Myname"), 1, 3}, obj{obj_named("k0tto"), 1, 11}},
			obj_pair{obj{obj_named("Age"), 1, 18}, obj{obj_int(2), 1, 23}},
		}, 1, 1}
	log.SetPrefix("TestDict: ")
	pdf, err := Parse([]byte(str))
	if err != nil {
		log.Printf(`Failed to parse valid pdf %T object. %v\n`, c_dict.Type, err)
		t.Fail()
	}
	if !matchObj(c_dict, pdf.objs[0]) {
		t.Fail()
	}
}

func TestRef(t *testing.T) {
	str := `%PDF-1.4
0 1 R
%%EOF`
	c_ref := obj{obj_ref{obj_int(0),
		obj_int(1)}, 1, 1}
	log.SetPrefix("TestRef: ")
	pdf, err := Parse([]byte(str))
	if err != nil {
		log.Printf(`Failed to parse valid pdf %T object. %v\n`, c_ref.Type, err)
		t.Fail()
	}
	if !matchObj(c_ref, pdf.objs[0]) {
		t.Fail()
	}
}

func TestIndEmpty(t *testing.T) {
	str := `%PDF-1.4
0 1 obj
endobj
%%EOF`
	c_obj := obj{obj_ind{obj_int(0),
		obj_int(1), nil}, 1, 1}
	log.SetPrefix("TestIndEmpty: ")
	pdf, err := Parse([]byte(str))
	if err != nil {
		log.Printf(`Failed to parse valid pdf %T object. %v\n`, c_obj.Type, err)
		t.Fail()
	}
	if !matchObj(c_obj, pdf.objs[0]) {
		t.Fail()
	}
}

func TestStrl(t *testing.T) {
	str := `%PDF-1.4
(My cool string)
%%EOF`
	txt := strings.Split(str, "\n")
	t_txt := strings.Trim(txt[1], "()")
	c_strl := obj{obj_strl{t_txt, 0}, 1, 1}
	log.SetPrefix("TestStrl: ")
	pdf, err := Parse([]byte(str))
	if err != nil {
		log.Printf(`Failed to parse valid pdf %T object. %v\n`, c_strl.Type, err)
		t.Fail()
	}
	if !matchObj(c_strl, pdf.objs[0]) {
		t.Fail()
	}
}

func TestStrh(t *testing.T) {
	log.SetPrefix("TestStrh: ")
	str := `%PDF-1.4
<My cool string>
%%EOF`

  h_str:= ""
  index_l := strings.IndexByte(str, '<')+1
  index_r := strings.IndexByte(str, '>')
  for _, c := range str[index_l:index_r] {
		h_str += fmt.Sprintf("%x", c)
	}
  lstr := str[:index_l]
  lstr += h_str
  lstr += str[index_r:]

  c_strh := obj{obj_strh(str[index_l:index_r]), 1, 1}
	pdf, err := Parse([]byte(lstr))
	if err != nil {
		log.Printf(`Failed to parse valid pdf %T object. %v\n`, c_strh.Type, err)
		t.Fail()
	}
	if !matchObj(c_strh, pdf.objs[0]) {
		t.Fail()
	}
}

func TestBool(t *testing.T) {
	str_true := `%PDF-1.4
true
%%EOF`
	c_true := obj{obj_bool(true), 1, 1}
	log.SetPrefix("TestBool: ")
	pdf, err := Parse([]byte(str_true))
	if err != nil {
		log.Printf(`Failed to parse valid pdf %T object. %v\n`, c_true.Type, err)
		t.Fail()
	}
	if !matchObj(c_true, pdf.objs[0]) {
		t.Fail()
	}

	str_false := `%PDF-1.4
false
%%EOF`
	c_false := obj{obj_bool(false), 1, 1}
	pdf2, err := Parse([]byte(str_false))
	if err != nil {
		log.Printf(`Failed to parse valid pdf %T object. %v\n`, c_false.Type, err)
		t.Fail()
	}
	if !matchObj(c_false, pdf2.objs[0]) {
		t.Fail()
	}
}

func TestNull(t *testing.T) {
	str := `%PDF-1.4
null
%%EOF`
	c_null := obj{obj_null(nil), 1, 1}
	log.SetPrefix("TestNull: ")
	pdf, err := Parse([]byte(str))
	if err != nil {
		log.Printf(`Failed to parse valid pdf %T object. %v\n`, c_null.Type, err)
		t.Fail()
	}
	if !matchObj(c_null, pdf.objs[0]) {
		t.Fail()
	}
}

// type obj_stream []obj    // stream endstream
func TestStreamEmpty(t *testing.T) {
	str := `%PDF-1.4
1 0 obj
<</Length 0>>
stream
endstream
endobj
%%EOF`
	c_stream := obj{obj_ind{obj_int(1), obj_int(0),
    []obj{{obj_dict{obj_pair{obj{obj_named("Length"),0,0},obj{obj_int(0),0,0}}},0,0},
      {obj_stream{}, 1, 1},
  }},0,0}
	log.SetPrefix("TestStreamEmpty: ")
	pdf, err := Parse([]byte(str))
	if err != nil {
		log.Printf(`Failed to parse valid pdf %T object. %v\n`, c_stream.Type, err)
		t.Fail()
	}
	if !matchObj(c_stream, pdf.objs[0]) {
		t.Fail()
	}
}
// NOTE)k0tto): obj_pair isn't a pdf object. It is used internaly.
// It should just work™.
// func TestPair(t *testing.T) {
//   log.SetPrefix("TestPair: ")
//   str := `<My cool string>`

//   h_str := "<"
//   t_str := strings.Trim(str, "<>")
//   for _, c := range t_str {
//     h_str += fmt.Sprintf("%x", c)
//   }
//   h_str += ">"

//   txt := strings.Split(h_str, "\n")
//   c_strh := obj{obj_strh(t_str), 1, 1}
//   o, _, err := Get_pdf_obj(txt)
//   if err != nil {
//     log.Printf(`Failed to parse valid pdf %T object. %v\n`, c_strh.Type, err)
//     t.Fail()
//   }
//   if !matchObj(c_strh, o) {
//     t.Fail()
//   }
// }

func TestInd_WithDict(t *testing.T) {
	log.SetPrefix("TestInd_WithDict: ")
	str := `%PDF-1.4
0 1 obj
<</Myname /k0tto /Age 2>>
endobj
%%EOF`
	c_dict := obj{
		obj_dict{
			obj_pair{obj{obj_named("Myname"), 2, 5}, obj{obj_named("k0tto"), 2, 13}},
			obj_pair{obj{obj_named("Age"), 2, 20}, obj{obj_int(2), 2, 25}},
		}, 2, 3}
	c_obj := obj{obj_ind{obj_int(0),
		obj_int(1), []obj{c_dict}}, 1, 1}
	pdf, err := Parse([]byte(str))
	if err != nil {
		log.Printf(`Failed to parse valid pdf %T object. %v\n`, c_obj.Type, err)
		t.Fail()
	}
	if !matchObj(c_obj, pdf.objs[0]) {
		t.Fail()
	}
}

func TestInd_WithComplDict(t *testing.T) {
	log.SetPrefix("TestInd_WithDict: ")
	str := `%PDF-1.4
4 0 obj
<<  /Type /Page
/Parent 3 0 R
/MediaBox [0 0 612 792]
/Contents 5 0 R
/Resources << /ProcSet 6 0 R
/Font << /F1 7 0 R >>
>>
>>
endobj
%%EOF`
	cdict := obj_dict{
		obj_pair{obj{obj_named("Type"), 3, 5}, obj{obj_named("Page"), 3, 11}},
		obj_pair{obj{obj_named("Parent"), 4, 1}, obj{obj_ref{obj_int(3), obj_int(0)}, 4, 14}},
		obj_pair{obj{obj_named("MediaBox"), 5, 1},
			obj{obj_array{
				obj{obj_int(0), 0, 0},
				obj{obj_int(0), 0, 0},
				obj{obj_int(612), 0, 0},
				obj{obj_int(792), 0, 0}}, 0, 0}},
		obj_pair{obj{obj_named("Contents"), 0, 0}, obj{obj_ref{obj_int(5), obj_int(0)}, 0, 0}},
		obj_pair{obj{obj_named("Resources"), 0, 0},
			obj{obj_dict{obj_pair{obj{obj_named("ProcSet"), 0, 0},
				obj{obj_ref{obj_int(6), obj_int(0)}, 0, 0}},
				obj_pair{obj{obj_named("Font"), 0, 0},
					obj{obj_dict{
						obj_pair{obj{obj_named("F1"), 0, 0}, obj{obj_ref{7, 0}, 0, 0}}}, 8, 7}},
			}, 7, 12},
		},
	}
	c_obj := obj{obj_ind{
		obj_int(4), obj_int(0),
		[]obj{{cdict, 0, 0}}},
		0, 0}
	pdf, err := Parse([]byte(str))
	if err != nil {
		log.Printf(`Failed to parse valid pdf %T object. %v\n`, c_obj.Type, err)
		t.Fail()
	}
	if !matchObj(c_obj, pdf.objs[0]) {
		t.Fail()
	}
}

func TestXref_simple(t *testing.T) {
	log.SetPrefix("TestXref_simple: ")
	str := `%PDF-1.4
xref
0 8
0000000000 65535 f
0000000009 00000 n
0000000074 00000 n
0000000120 00000 n
0000000179 00000 n
0000000364 00000 n
0000000466 00000 n
0000000496 00000 n
trailer
<< /Size 8
/Root 1 0 R
>>
startxref
625
%%EOF`
	refs := []xref_ref{
		{0, 65535, "f"},
		{9, 0, "n"},
		{74, 0, "n"},
		{120, 0, "n"},
		{179, 0, "n"},
		{364, 0, "n"},
		{466, 0, "n"},
		{496, 0, "n"}}
	dict := obj_dict{
		obj_pair{obj{obj_named("Size"), 0, 0}, obj{obj_int(8), 0, 0}},
		obj_pair{obj{obj_named("Root"), 0, 0}, obj{obj_ref{1, 0}, 0, 0}},
	}
	c_obj := obj{obj_xref{
		obj_int(0), refs,
		dict, obj_int(625)},
		0, 0}
	pdf, err := Parse([]byte(str))
	if err != nil {
		log.Printf(`Failed to parse valid pdf %T object. %v\n`, c_obj.Type, err)
		t.Fail()
	}
	if !matchObj(c_obj, pdf.objs[0]) {
		t.Fail()
	}
}
