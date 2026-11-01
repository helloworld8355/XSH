package pipeline

import (
	"reflect"
	"testing"
)

func TestParseBasic(t *testing.T) {
	st, err := Parse("ls -l /tmp")
	if err != nil {
		t.Fatal(err)
	}
	if len(st) != 1 || st[0].Name != "ls" || !reflect.DeepEqual(st[0].Args, []string{"-l", "/tmp"}) {
		t.Fatalf("got %+v", st)
	}
}

func TestParsePipe(t *testing.T) {
	st, err := Parse("ps | where cpu gt 5 | ft")
	if err != nil {
		t.Fatal(err)
	}
	if len(st) != 3 {
		t.Fatalf("want 3 stages got %d", len(st))
	}
	if st[0].Name != "ps" || st[1].Name != "where" || st[1].Args[0] != "cpu" || st[2].Name != "ft" {
		t.Fatalf("got %+v", st)
	}
}

func TestParseQuotes(t *testing.T) {
	st, _ := Parse(`echo "hello world"`)
	if st[0].Args[0] != "hello world" {
		t.Fatalf("got %+v", st)
	}
	st, _ = Parse(`echo 'a b'`)
	if st[0].Args[0] != "a b" {
		t.Fatalf("got %+v", st)
	}
	st, _ = Parse(`echo "a\"b"`)
	if st[0].Args[0] != `a"b` {
		t.Fatalf("got %+v", st)
	}
}

func TestParseComment(t *testing.T) {
	st, _ := Parse("ls # comment here")
	if len(st) != 1 || len(st[0].Args) != 0 {
		t.Fatalf("got %+v", st)
	}
	st, _ = Parse(`grep "#tag"`)
	if len(st) != 1 || st[0].Args[0] != "#tag" {
		t.Fatalf("got %+v", st)
	}
}

func TestParsePipeError(t *testing.T) {
	if _, err := Parse("| ls"); err == nil {
		t.Fatal("want error")
	}
	if _, err := Parse("ls |"); err == nil {
		t.Fatal("want error")
	}
}

func TestParseEmpty(t *testing.T) {
	st, err := Parse("   # only comment")
	if err != nil || st != nil {
		t.Fatalf("got %+v %v", st, err)
	}
}
