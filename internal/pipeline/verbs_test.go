package pipeline

import "testing"

func sample() []Record {
	return []Record{
		{"name": "a", "cpu": "1.5", "type": "x"},
		{"name": "b", "cpu": "10", "type": "y"},
		{"name": "c", "cpu": "3", "type": "x"},
	}
}

func TestFilter(t *testing.T) {
	out := Filter(sample(), "cpu", "gt", "2")
	if len(out) != 2 {
		t.Fatalf("got %d", len(out))
	}
	out = Filter(sample(), "type", "eq", "x")
	if len(out) != 2 {
		t.Fatalf("got %d", len(out))
	}
	out = Filter(sample(), "name", "like", "a")
	if len(out) != 1 {
		t.Fatalf("got %d", len(out))
	}
	out = Filter(sample(), "type", "in", "x,y")
	if len(out) != 3 {
		t.Fatalf("got %d", len(out))
	}
}

func TestProject(t *testing.T) {
	out := Project(sample(), []string{"name"})
	if len(out) != 3 || out[0]["cpu"] != "" {
		t.Fatalf("got %+v", out)
	}
}

func TestSort(t *testing.T) {
	in := sample()
	SortRecords(in, "cpu", true, false)
	if in[0]["name"] != "b" || in[2]["name"] != "a" {
		t.Fatalf("got %+v", in)
	}
	SortRecords(in, "name", false, false)
	if in[0]["name"] != "a" {
		t.Fatalf("got %+v", in)
	}
}

func TestMeasure(t *testing.T) {
	r := Measure(sample(), "cpu")
	if r["count"] != "3" || r["sum"] != "14.5" {
		t.Fatalf("got %+v", r)
	}
	r = Measure(sample(), "")
	if r["count"] != "3" {
		t.Fatalf("got %+v", r)
	}
}

func TestGroup(t *testing.T) {
	out := Group(sample(), "type")
	if len(out) != 2 || out[0]["value"] != "x" || out[0]["count"] != "2" {
		t.Fatalf("got %+v", out)
	}
}

func TestFirstLast(t *testing.T) {
	if len(FirstN(sample(), 2)) != 2 {
		t.Fatal("first")
	}
	if len(LastN(sample(), 1)) != 1 {
		t.Fatal("last")
	}
}
