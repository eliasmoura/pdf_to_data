package pdf

import (
	"errors"
	"fmt"
)

// PDF operators
// Category                     | operators                         | table  | page
// XXXX: General graphics state | w, J, j, M, d, ri, i, gs          | 4.7    | 156
// XXXX: Special graphics state | q, Q, cm                          | 4.7    | 156
// TODO: Path construction      | m, l, c, v, y, h, re              | 4.9    | 163
// TODO: Path painting          | S, s, f, F, f*, B, B*, b, b*, n   | 4.10   | 167
// TODO: Clipping paths         | W, W*                             | 4.11   | 172
// TODO: Text objects           | BT, ET                            | 5.4    | 308
// TODO: Text state             | Tc, Tw, Tz, TL, Tf, Tr, Ts        | 5.2    | 302
// TODO: Text positioning       | Td, TD, Tm, T*                    | 5.5    | 310
// TODO: Text showing           | Tj, TJ, ', "                      | 5.6    | 311
// TODO: Type 3 fonts           | d0, d1                            | 5.10   | 326
//   XX: Color                  | CS, cs, SC, SCN, sc, scn, G, g, RG, rg, K, k |4.21 |216
// TODO: Shading patterns       | sh                                | 4.24   | 232
// XXXX: Inline images          | BI, ID, EI                        | 4.38   | 278
// TODO: XObjects               | Do                                | 4.34   | 261
// TODO: Marked content         | MP, DP, BMC, BDC, EMC             | 9.8    | 584
// TODO: Compatibility          | BX, EX                            | 3.20   | 95

func handle_operator(objs []obj, operator string) ([]obj, error) {
	switch operator {
	// XXXX: General graphics state | w, J, j, M, d, ri, i, gs          | 4.7    | 156
	case "w", "J", "j", "M", "d", "ri", "i", "gs":
		return handle_seq_num(objs, 1, operator)
		// XXXX: Special graphics state | q, Q, cm                          | 4.7    | 156
	case "q", "Q":
	case "cm":
		return handle_seq_num(objs, 6, operator)
		// TODO: Path construction      | m, l, c, v, y, h, re              | 4.9    | 163
	case "m", "l", "c", "v", "y", "h", "re":
		// TODO: Path painting          | S, s, f, F, f*, B, B*, b, b*, n   | 4.10   | 167
	case "S", "s", "f", "F", "f*", "B", "B*", "b", "b*", "n":
		return objs, nil
		// TODO: Clipping paths         | W, W*                             | 4.11   | 172
	case "W", "W*":
		// TODO: Text objects           | BT, ET                            | 5.4    | 308
	case "BT", "ET":
		// TODO: Text state             | Tc, Tw, Tz, TL, Tf, Tr, Ts        | 5.2    | 302
	case "Tc", "Tw", "Tz", "TL", "Tf", "Tr", "Ts":
		// TODO: Text positioning       | Td, TD, Tm, T*                    | 5.5    | 310
	case "Td", "TD", "Tm", "T*":
		// TODO: Text showing           | Tj, TJ, ', "                      | 5.6    | 311
	case "Tj", "TJ", "'", "\"":
		// TODO: Type 3 fonts           | d0, d1                            | 5.10   | 326
	case "d0", "d1":
		//   XX: Color                  | CS, cs, SC, SCN, sc, scn, G, g, RG, rg, K, k |4.21 |216
	case "CS", "cs", "SC", "SCN", "sc", "scn", "G", "g", "RG", "rg", "K", "k":
		// TODO: Shading patterns       | sh                                | 4.24   | 232
	case "sh":
		// XXXX: Inline images          | BI, ID, EI                        | 4.38   | 278
	case "BI", "ID", "EI":
		return objs, nil
	// TODO: XObjects               | Do                                | 4.34   | 261
	case "Do":
	// TODO: Marked content         | MP, DP, BMC, BDC, EMC             | 9.8    | 584
	case "MP", "DP", "BMC", "BDC", "EMC":
		// TODO: Compatibility          | BX, EX                            | 3.20   | 95
	case "BX", "EX":
	}
	return objs, errors.New(fmt.Sprintf("Could not match operator `%s`\n", operator))
}

// XXXX: Special graphics state | q, Q, cm                          | 4.7    | 156
func handle_seq_num(objs []obj, total_count int, operator string) ([]obj, error) {
	count := 0 // cm is a 6 element obj
	for len(objs) > 0 {
		objs, o := Pop(objs)
		switch o.Type.(type) {
		case obj_int, obj_real:
			count++
		default:
			return objs, errors.New(fmt.Sprintf("ERROR:%d:%d %s expected a number found %s\n", o.line, o.col, operator, typeStr(o)))
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
