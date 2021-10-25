package query

import (
	"log"
	"testing"
)

func init() {
	log.SetFlags(log.Lshortfile)
}

func TestTokens(t *testing.T) {
	str := `[4 @"END"]`
	tokens, err := get_tokens(str)
	if err != nil {
		log.Println(err)
		t.Fail()
	}
	if len(tokens) != 7 {
		log.Printf("Length mismatch, expected %d, got %d\n", 9, len(tokens))
		t.Fail()
	}
	if tokens[2] != "@" {
		log.Printf("Expected `%s`, found `%s`\n", "@", tokens[2])
		t.Fail()
	}
	if tokens[0] != "[" {
		log.Printf("Expected `%s`, found `%s`\n", "[", tokens[0])
		t.Fail()
	}
	if tokens[4] != "END" {
		log.Printf("Expected `%s`, found `%s`\n", "END", tokens[4])
		t.Fail()
	}
}

func TestIndex(t *testing.T) {
	str := `#2` // zero indexed
	txt := []string{"num 0", "num 1", "num 3"}

	query, err := ParseQuery(str)
	if err != nil {
		log.Println(err)
		t.Fail()
	}
	if len(query) != 1 {
		log.Printf("Length mismatch, expected %d, got %d\n", 1, len(query))
		t.Fail()
	}
	result, err := RunQuery(query, txt)
	if err != nil {
		log.Printf("Query `%s` did not find the entry %s\n", str, txt[2])
		t.Fail()
	}
	if result[0][0] != txt[2] {
		log.Printf("got `%s`, expected `%s`\n", result[0], txt[2])
		t.Fail()
	}
}

func TestStartIndex(t *testing.T) {
	str := `@"START TEXT 1"[1]`
	txt := []string{"Should skip 0", "START TEXT 1", "Should Print this 2", "And this 3"}

	query, err := ParseQuery(str)
	if err != nil {
		log.Println(err)
		t.Fail()
	}
	if len(query) != 4 {
		log.Printf("Length mismatch, expected %d, got %d\n", 4, len(query))
		t.Fail()
	}
	result, err := RunQuery(query, txt)
	if err != nil {
		log.Printf("Query `%s` did not find the entry %s\n%s\n", str, txt[2], result)
		t.Fail()
	}
	if result[0][0] != txt[2] {
		log.Printf("got `%s`, expected `%s`\n", result[0][0], txt[2])
		t.Fail()
	}
	if result[1][0] != txt[3] {
		log.Printf("got `%s`, expected `%s`\n", result[0][0], txt[3])
		t.Fail()
	}
}

func TestEndIndex(t *testing.T) {
	str := `[1@"START TEXT 1"]`
	txt := []string{"Should skip 0", "START TEXT 1", "Should Print this 3", "And this 4"}

	query, err := ParseQuery(str)
	if err != nil {
		log.Println(err)
		t.Fail()
	}
	if len(query) != 3 {
		log.Printf("Length mismatch, expected %d, got %d\n", 3, len(query))
		t.Fail()
	}
	result, err := RunQuery(query, txt)
	if err != nil {
		log.Printf("Query `%s` did not find the entry %s\n%s\n", str, txt[2], result)
		t.Fail()
	}
	if len(result) == 1 && result[0][0] != txt[0] {
		log.Printf("got `%s`, expected `%s`\n", result[0], txt[0])
		t.Fail()
	}
}
