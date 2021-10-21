package pdf

import (
	"errors"
	"fmt"
)

func handle_xref(objs []obj) (obj_xref, error) {
	var xref obj_xref
	if len(objs) < 2 {
		return xref, errors.New(fmt.Sprintf("ERROR: xref expected 2 or more elementes, found %d\n!", len(objs)))
	}

	o := objs[0]
	id, ok_id := o.Type.(obj_int)
	if !ok_id {
		return xref, errors.New(fmt.Sprintf("ERROR:%d:%d: %v should be an obj_int is %T\n", objs[0].line, objs[0].col, objs[0], objs[0]))
	}
	xref.id = id
	o = objs[1]
	o_tot := o
	tot, ok_tot := o.Type.(obj_int)
	if !ok_tot {
		return xref, errors.New(fmt.Sprintf("ERROR:%d:%d: %v should be an obj_int is %T\n", objs[1].line, objs[1].col, objs[1], objs[1]))

	}
	objs = objs[2:]

	objs, o_start := Pop(objs)
	startxref, ok := o_start.Type.(obj_int)
	if !ok {
		return xref, errors.New(fmt.Sprintf("ERROR:%d:%d expected interger, found %v\n!\n", o_start.line, o_start.col, o_start.Type))
	}
	xref.startxref = startxref

	var o_dict obj
	// discard everything until we find the dict
	for len(objs) > 0 {
		objs, o_dict = Pop(objs)
		var ok bool
		xref.enc, ok = o_dict.Type.(obj_dict)
		if ok {
			break
		}
	}
	if xref.enc == nil {
		return xref, errors.New(fmt.Sprintf("ERROR:%d:%d: %v should be an obj_dict is %T\n", o.line, o.col, o, o))
	}

	if int(tot) != len(objs)/3 {
		return xref, errors.New(fmt.Sprintf("ERROR:%d:%d: given number of xrefs `%d` doesn't match the number of xrefs found `%d`.\n", o_tot.line, o_tot.col, tot, len(objs)/3))
	}

	var i int
	for i = 0; i < len(objs); i += 3 {
		_1, ok1 := objs[i].Type.(obj_int)
		_2, ok2 := objs[i+1].Type.(obj_int)
		_3, ok3 := objs[i+2].Type.(string)
		if !ok1 || !ok2 || !ok3 {
			return xref, errors.New(fmt.Sprintf("ERROR:%d:%d: expect xref ref, found %s[%v]\n", objs[i].line, objs[i].col, typeStr(objs[i]), objs[i].Type))
		}
		xref.refs = AppendRef(xref.refs, xref_ref{_1, _2, _3})
	}

	return xref, nil
}
