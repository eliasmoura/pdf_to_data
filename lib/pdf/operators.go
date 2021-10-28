package pdf

import (
	"errors"
	"fmt"
	"log"
)

type ColorSpace obj_named

var cs ColorSpace

const (
	DeviceGray  ColorSpace = "DeviceGray"
	CalGray     ColorSpace = "CalGray"
	DeviceRGB   ColorSpace = "DeviceRGB"
	CalRGB      ColorSpace = "CalRGB"
	DeviceCMYK  ColorSpace = "DeiceCMYK"
	Lab         ColorSpace = "Lab"
	ICCBAsed    ColorSpace = "ICCBAsed"
	Indexed     ColorSpace = "Indexed"
	Pattern     ColorSpace = "Pattern"
	Separation  ColorSpace = "Separation"
	DeviceN     ColorSpace = "DeviceN"
	DefaultCMYK ColorSpace = "DefaultjCMYK"
)

func (cp ColorSpace) String() string {
	return string(cp)
}

// PDF operators from the PDFReference.pdf
// Category                | operators                         | page
//  General graphics state | w, J, j, M, d, ri, i, gs          | 156
//  Special graphics state | q, Q, cm                          | 156
//  Path construction      | m, l, c, v, y, h, re              | 163
//  Path painting          | S, s, f, F, f*, B, B*, b, b*, n   | 167
//  Clipping paths         | W, W*                             | 172
//  Text objects           | BT, ET                            | 308
//  Text state             | Tc, Tw, Tz, TL, Tf, Tr, Ts        | 302
//  Text positioning       | Td, TD, Tm, T*                    | 310
//  Text showing           | Tj, TJ, ', "                      | 311
//  Type 3 fonts           | d0, d1                            | 326
//  Color                  | CS, cs, SC, SCN, sc, scn, G, g, RG, rg, K, k | 216
//  Shading patterns       | sh                                | 232
//  Inline images          | BI, ID, EI                        | 278
//  XObjects               | Do                                | 261
//  Marked content         | MP, DP, BMC, BDC, EMC             | 584
//  Compatibility          | BX, EX                            | 95

func handle_operator(objs []obj, operator string, color_space obj_dict) ([]obj, error) {
	switch operator {
	//  General graphics state | w, J, j, M, d, ri, i, gs          | 156
	case "w", "J", "j", "M", "i":
		return handle_seq_num(objs, 1, operator)
	case "d":
		objs, _ = Pop(objs)
		objs, _ = Pop(objs)
		return objs, nil
	case "ri":
		objs, _ = Pop(objs)
		return objs, nil
	case "gs":
		objs, _ = Pop(objs)
		return objs, nil

		//  Special graphics state | q, Q, cm                          | 156
	case "q", "Q":
		return objs, nil
	case "cm":
		return handle_seq_num(objs, 6, operator)

	//  Path construction      | m, l, c, v, y, h, re              | 163
	case "m", "l":
		return handle_seq_num(objs, 2, operator)
	case "c":
		return handle_seq_num(objs, 6, operator)
	case "v", "y", "re":
		return handle_seq_num(objs, 4, operator)
	case "h":
		return objs, nil

		//  Path painting          | S, s, f, F, f*, B, B*, b, b*, n   | 167
	case "S", "s", "f", "F", "f*", "B", "B*", "b", "b*", "n":
		return objs, nil

		//  Clipping paths         | W, W*                             | 172
	case "W", "W*":
		return objs, nil

		//  Text objects           | BT, ET                            | 308
	case "BT", "ET":
		return objs, nil

		//  Text state             | Tc, Tw, Tz, TL, Tf, Tr, Ts        | 302
	case "Tc", "Tw", "Tz", "TL", "Tr", "Ts":
		return handle_seq_num(objs, 1, operator)
	case "Tf":
		objs, _ = Pop(objs)
		objs, _ = Pop(objs)
		return objs, nil

	//  Text positioning       | Td, TD, Tm, T*                    | 310
	case "T*":
		return objs, nil
	case "Td", "TD":
		return handle_seq_num(objs, 2, operator)
	case "Tm":
		return handle_seq_num(objs, 6, operator)

	//  Text showing           | Tj, TJ, ', "                      | 311
	case "Tj":
		return objs, nil
	case "TJ":
		var o obj
		objs, o = Pop(objs)
		result := obj_str("")
		array, ok := o.Type.(obj_array)
		if ok {
			for _, o := range array {
				switch val := o.Type.(type) {
				case obj_strl:
					// objs = Append(objs, o)
					result += obj_str(val)
				case obj_strh:
					result += obj_str(val)
				case obj_real:
					if int(val) < -200 && int(val) > -450 {
						result += " "
					} else {
						if int(val) < -500 {
							objs = append(objs, obj{result, 0, 0})
							result = ""
						}
					}
				case obj_int:
					if int(val) < -200 && int(val) > -450 {
						result += " "
					} else {
						if int(val) < -500 {
							objs = append(objs, obj{result, 0, 0})
							result = ""
						}
					}
				}
			}
			objs = append(objs, obj{result, 0, 0})
		}
		return objs, nil
	case "'":
		strl, ok := objs[len(objs)-1].Type.(obj_strl)
		if ok {
			strl = "\n" + strl
			objs[len(objs)-1].Type = strl
		}
		strh, ok := objs[len(objs)-1].Type.(obj_strh)
		if ok {
			strh = "\n" + strh
			objs[len(objs)-1].Type = strh
		}
		return objs, nil
	case "\"":
		var o obj
		objs, o = Pop(objs)
		strl, ok := o.Type.(obj_strl)
		if ok {
			strl = "\n" + strl
			o.Type = strl
		}
		strh, ok := o.Type.(obj_strh)
		if ok {
			strh = "\n" + strh
			o.Type = strh
		}
		var err error
		objs, err = handle_seq_num(objs, 2, operator)
		return objs, err

	//  Type 3 fonts           | d0, d1                            | 326
	case "d0":
		return handle_seq_num(objs, 2, operator)
	case "d1":
		return handle_seq_num(objs, 6, operator)

	//  Color                  | CS, cs, SC, SCN, sc, scn, G, g, RG, rg, K, k |4.21 |216
	case "CS", "cs":
		var o obj
		objs, o = Pop(objs)
		color, ok := o.Type.(obj_named)
		if ok {
			var err error
			cs, err = get_color_space(color, color_space)
			if err != nil {
				fmt.Println(err)
				return objs, err
			}
		}
		return objs, nil
		// NOTE(elias): need to keep track of the ColorSpace, since the amount os operands used
		// byt the operator depends in things like the the current color space
	case "SC", "sc":
		switch cs {
		case DeviceGray, CalGray, Indexed:
			objs, _ = Pop(objs)
			return objs, nil
		case DeviceRGB, CalRGB, Lab:
			return handle_seq_num(objs, 3, operator)
		case DeviceCMYK:
			return handle_seq_num(objs, 4, operator)
		}
	case "SCN", "scn":
		switch cs {
		case DeviceGray, CalGray, Indexed:
			objs, _ = Pop(objs)
			return objs, nil
		case DeviceRGB, CalRGB, Lab:
			return handle_seq_num(objs, 3, operator)
		case DeviceCMYK:
			return handle_seq_num(objs, 4, operator)
		case Pattern:
			objs, _ = Pop(objs)
			return handle_seq_num(objs, 4, operator)
		}
	case "RG", "rg":
		cs = DeviceRGB
		return handle_seq_num(objs, 3, operator)
	case "K", "k":
		cs = DeviceCMYK
		return handle_seq_num(objs, 4, operator)

	case "g", "G":
		objs, _ = Pop(objs)
		return objs, nil

		//  Shading patterns       | sh                                | 232
	case "sh":
		objs, _ = Pop(objs)

		//  Inline images          | BI, ID, EI                        | 278
	case "BI", "ID", "EI":
		return objs, nil
	//  XObjects               | Do                                | 261
	case "Do":
		objs, _ := Pop(objs)
		return objs, nil

		//  Marked content         | MP, DP, BMC, BDC, EMC             | 584
	case "MP", "BMC":
		objs, _ = Pop(objs)
		return objs, nil
	case "DP", "BDC":
		objs, _ = Pop(objs)
		objs, _ = Pop(objs)
		return objs, nil
	case "EMC":
		return objs, nil

		//  Compatibility          | BX, EX                            | 95
	case "BX", "EX":
		return objs, nil
	}
	return objs, errors.New(fmt.Sprintf("Could not match operator `%s`\n", operator))
}

func get_color_space(color obj_named, color_space obj_dict) (ColorSpace, error) {
	var result ColorSpace
	var err error
	switch ColorSpace(color) {
	case DeviceGray:
		result = DeviceGray
	case CalGray:
		result = CalGray
	case DeviceRGB:
		result = DeviceRGB
	case CalRGB:
		result = CalRGB
	case DeviceCMYK:
		result = DeviceCMYK
	case Lab:
		result = Lab
	case ICCBAsed:
		result = ICCBAsed
	case Indexed:
		result = Indexed
	case Pattern:
		result = Pattern
	case Separation:
		result = Separation
	case DeviceN:
		result = DeviceN
	case DefaultCMYK:
		result = DefaultCMYK
	default:
		_cs, ok := color_space["ColorSpace"].Type.(obj_dict)
		if ok {
			var c obj_named
			c, ok = _cs[color].Type.(obj_named)
			if ok {
				result = ColorSpace(c)
				break
			}
			// TODO: Not properly implemented
			//just fail for now
			// var r obj_ref
			// r, ok = _cs[color].Type.(obj_ref)
			// if ok {
			// // o_ind, err := get_obj_by_id(r.id)
			// // if err == nil {
			// //   ind, ok := o_ind.Type.(obj_ind)
			// //   if ok {

			// //   }
			// // }
			// }
		}
		err = errors.New(fmt.Sprintf("Could not get the ColorSpace: %s", color))
	}
	return result, err
}

// XXXX: Special graphics state | q, Q, cm                          | 156
func handle_seq_num(objs []obj, total_count int, operator string) ([]obj, error) {
	count := 0 // cm is a 6 element obj
	for len(objs) > 0 {
		var o obj
		objs, o = Pop(objs)
		switch o.Type.(type) {
		case obj_int, obj_real:
			count++
		default:
			log.Printf("ERROR:%d:%d operator `%s` expected a %s found %s\n", o.line, o.col, operator, "number?", typeStr(o))
			return objs, nil
		}
		if count == total_count {
			return objs, nil
		}
	}
	return objs, nil
}

// TODO: Color                  | CS, cs, SC, SCN, sc, scn, G, g, RG, rg, K, k |4.21 |216
func handle_op_SC(objs []obj) ([]obj, error) {
	count := 0 // SC is a 3 element obj : 245 2 0 SC
	for len(objs) > 0 {
		objs, o := Pop(objs)
		switch o.Type.(type) {
		case obj_int, obj_real:
			count++
		default:
			return objs, errors.New(fmt.Sprintf("ERROR:%d:%d expected a number found %s\n", o.line, o.col, typeStr(o)))
		}
		if count == 3 {
			return objs, nil
		}
	}
	return objs, errors.New(fmt.Sprintf("ERROR: SC expected 3 numbers found, %d\n", count))
}

func handle_op_RG(objs []obj) ([]obj, error) {
	count := 0 // RG is a 3 element obj : 245 2 0 SC
	for len(objs) > 0 {
		objs, o := Pop(objs)
		switch o.Type.(type) {
		case obj_int, obj_real:
			count++
		default:
			return objs, errors.New(fmt.Sprintf("ERROR:%d:%d expected a number found %s\n", o.line, o.col, typeStr(o)))
		}
		if count == 6 {
			return objs, nil
		}
	}
	return objs, errors.New(fmt.Sprintf("ERROR: SC expected 3 numbers found, %d\n", count))
}
