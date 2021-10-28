package query

import (
	"errors"
	"fmt"
	"log"
	"strconv"
	"strings"
)

type op interface{}
type index uint64

type op_print int
type op_printindex int
type op_incdataindex int
type op_setdataindex int
type op_setdataindex_fromstr string

// type op_
type op_condition_str string
type op_condition_index int
type op_condition_eof bool
type op_jump struct {
	Condition interface{}
	Label     op_label
}
type op_stop_atstr string
type op_stopatdataindex int
type op_label int

func append_op(o []op, _o op) []op {
	l := len(o)
	if l >= cap(o) {
		newstr := make([]op, (l+1)*2)
		copy(newstr, o)
		o = newstr
	}
	o = o[:l+1]
	o[l] = _o
	return o
}

func append_str_d(str [][]string, other_strs []string) [][]string {
	l := len(str)
	if l == cap(str) {
		newstr := make([][]string, (l+1)*2)
		copy(newstr, str)
		str = newstr
	}
	str = str[0 : l+1]
	str[l] = other_strs
	return str
}

func append_str(str, other_strs []string) []string {
	l := len(str)
	n := l + len(other_strs)
	if n >= cap(str) {
		newstr := make([]string, (n+1)*2)
		copy(newstr, str)
		str = newstr
	}
	str = str[0:n]
	copy(str[l:], other_strs)
	return str
}

func get_tokens(str string) ([]string, error) {
	var result []string
	var i int
	for ; i < len(str); i++ {
		switch str[i] {
		case '#':
			var index int
			for j := range str[i+1:] {
				if str[i+j+1] >= '0' && str[i+j+1] <= '9' {
					index++
				}
			}
			if index == 0 {
				return result, errors.New(fmt.Sprintf("ERROR:%d no string index passed for %s\n", i, string(str[i])))
			}
			result = append_str(result, []string{str[i : i+1], str[i+1 : i+index+1]})
			i += index + 1
		case '"':
			index := strings.IndexByte(str[i+1:], '"')
			if index == -1 {
				return result, errors.New(fmt.Sprintf("ERROR:%d could not find the END STRING token %s\n", i, string(str[i])))
			}
			result = append_str(result, []string{str[i : i+1], str[i+1 : i+index+1], str[i+index+1 : i+index+2]})
			i += index + 1
		case ' ', '\t', '\n', '\r':
		case '[':
			result = append_str(result, []string{str[i : i+1]})
		case ']':
			result = append_str(result, []string{str[i : i+1]})
		case '@':
			result = append_str(result, []string{str[i : i+1]})
		case '$':
			result = append_str(result, []string{str[i : i+1]})
		case '{':
			result = append_str(result, []string{str[i : i+1]})
		case '}':
			result = append_str(result, []string{str[i : i+1]})
		case '|':
			result = append_str(result, []string{str[i : i+1]})
		case ',':
		case '-':
		case '+':
			var num int
			result = append_str(result, []string{str[i : i+1]})
			i++
			for j := range str[i:] {
				if str[i+j] < '0' || str[i+j] > '9' {
					break
				}
				num++
			}
			if num > 0 {
				result = append_str(result, []string{str[i : i+num]})
				i += num - 1
				continue
			}
			return result, errors.New(fmt.Sprintf("ERROR:%d failed parse token `%s`\n", i, string(str[i])))
		default:
			var num int
			for j := range str[i:] {
				if str[i+j] < '0' || str[i+j] > '9' {
					break
				}
				num++
			}
			if num > 0 {
				result = append_str(result, []string{str[i : i+num]})
				i += num - 1
				continue
			}
			return result, errors.New(fmt.Sprintf("ERROR:%d failed parse token `%s`\n", i, string(str[i])))
		}
	}

	return result, nil
}

func pop(slice []string) ([]string, string) {
	if len(slice) == 0 {
		log.Fatalln("Can't remove elements from a 0 length slice")
		return slice, ""
	}
	el := slice[len(slice)-1]
	slice = slice[:len(slice)-1]
	return slice, el
}

func RunQuery(ops []op, data []string) ([][]string, error) {
	var iq int
	var data_index int
	var result [][]string
	for iq < len(ops) {
		switch op := ops[iq].(type) {
		case op_label:
		case op_print:
			var line []string
			for i := op; i > 0 && data_index < len(data); i-- {
				line = append(line, data[data_index])
				data_index++
			}
			result = append_str_d(result, line)
		case op_printindex:
			result = append_str_d(result, []string{data[op]})
		case op_setdataindex:
			data_index = int(op)
		case op_incdataindex:
			data_index++
		case op_setdataindex_fromstr:
			found := false
			var _index int
			for i := data_index; i < len(data); i++ {
				if string(op) == data[i] {
					found = true
					_index = i
					break
				}
			}
			if found {
				data_index = _index + 1
			}
		case op_jump:
			switch val := op.Condition.(type) {
			case op_condition_str:
				if data_index == len(data) {
					return result, errors.New("ERROR: EOF at op_jump")
				}
				if data_index < len(data) && string(val) != data[data_index] {
					iq = int(op.Label)
				}
			case op_condition_index:
				if int(val) != data_index {
					iq = int(op.Label)
				}
			case op_condition_eof:
				if len(data) != data_index {
					iq = int(op.Label)
				}
			default:
				return result, errors.New(fmt.Sprintf("Error: invalid jump condition: %v\n", op.Condition))
			}
		default:
			log.Println("Not implemented!")
			return result, errors.New("Not implemented!")
		}
		iq++
	}
	return result, nil
}

func ParseQuery(txt string) ([]op, error) {
	var exec []op
	tokens, err := get_tokens(txt)
	if err != nil {
		return exec, errors.New("Failed to parse the query.")
	}
	var i int
	var cond_stack []string
	for i < len(tokens) {
		switch tokens[i] {
		case "#":
			i++
			s_val := tokens[i]
			_index, err := strconv.ParseUint(s_val, 10, 32)
			index := int(_index)
			if err != nil {
				return exec, errors.New(fmt.Sprintf("Failed to parse index: %s\n", err))
			}
			exec = append_op(exec, op_printindex(index))
			i++
		case "+":
			i++
			s_val := tokens[i]
			_index, err := strconv.ParseUint(s_val, 10, 32)
			index := int(_index)
			if err != nil {
				return exec, errors.New(fmt.Sprintf("Failed to parse index: %s\n", err))
			}
			exec = append_op(exec, op_incdataindex(index))
			i++
		case "\"":
			i++
			cond_stack = append(cond_stack, tokens[i])
			i += 2
		case "@":
			i++
			if tokens[i] == "\"" {
				i++
				exec = append(exec, op_setdataindex_fromstr(tokens[i]))
				i += 2
			} else if tokens[i] == "#" {
				i++
				s_val := tokens[i]
				_index, err := strconv.ParseUint(s_val, 10, 32)
				index := int(_index)
				if err != nil {
					return exec, errors.New(fmt.Sprintf("Failed to parse index: %s\n", err))
				}
				exec = append_op(exec, op_setdataindex(index))
				i++
			}
		case "$":
			i++
			if tokens[i] == "\"" {
				i++
				exec = append(exec, op_stop_atstr(tokens[i]))
				i += 2
			} else if tokens[i] == "#" {
				i++
				s_val := tokens[i]
				_index, err := strconv.ParseUint(s_val, 10, 32)
				index := int(_index)
				if err != nil {
					return exec, errors.New(fmt.Sprintf("Failed to parse index: %s\n", err))
				}
				exec = append(exec, op_stopatdataindex(index))
				i++
			}
		case "{":
			log.Printf("%s not implemented!", tokens[i])
			return exec, errors.New(fmt.Sprintf("%s not implemented!\n", tokens[i]))
		case "[":
			exec = append_op(exec, op_label(len(exec)))
			i++
			s_val := tokens[i]
			_count, err := strconv.ParseUint(s_val, 10, 32)
			count := int(_count)
			if err != nil {
				return exec, errors.New(fmt.Sprintf("Expected number of elements to print in a line, got %s\n%\n", tokens[i], err))
			}
			exec = append_op(exec, op_print(count))
			i++
		case "]":
			var l op_label
			found_label := false
			for i := len(exec) - 1; i >= 0; i-- {
				_index, ok := exec[i].(op_label)
				if ok {
					found_label = true
					l = _index
				}
			}
			var jump op_jump
			_, ok := exec[len(exec)-1].(op_print)
			if ok && found_label {
				jump = op_jump{Label: l}
				jump.Condition = op_condition_eof(true)
				exec = append_op(exec, jump)
				i++
				continue
			}
			if !ok && found_label {
				_exec := exec[len(exec)-1]
				exec = exec[:len(exec)-1]
				jump = op_jump{Label: l}
				switch val := _exec.(type) {
				case op_setdataindex_fromstr:
					jump.Condition = op_condition_str(val)
				case op_setdataindex:
					jump.Condition = op_condition_index(val)
				default:
					return exec, errors.New(fmt.Sprintf("Error: something wrong!!\n"))
				}
				exec = append_op(exec, jump)
				i++
			} else {
				return exec, errors.New(fmt.Sprintf("Error: `]` jump something wrong!!\n"))
			}
		default:
			s_val := tokens[i]
			_num, err := strconv.ParseInt(s_val, 10, 32)
			num := int(_num)
			if err != nil {
				return exec, errors.New(fmt.Sprintf("Failed to parse token, %s: %s", s_val, err))
			}
			found_label := false
			for i := len(exec) - 1; i >= 0; i-- {
				_, ok := exec[i].(op_label)
				if ok {
					found_label = true
				}
			}
			if found_label {
				exec = append_op(exec, op_print(num))
				// exec[l].data = num
			} else {
				return exec, errors.New(fmt.Sprintf("Invalid token %s\n", s_val))
			}
			i++
		}
	}
	return exec, nil
}
