package goraptor

import (
	"errors"
	"fmt"
	"log"
	"os"
	"runtime"
	"strings"
	"testing"
	"time"
)

func TestGlobal(t *testing.T) {
	log.Print("[log levels]")
	for level, str := range LogLevels {
		log.Printf("%d - %s", level, str)
	}
	log.Print(" ")
	log.Print("[parser syntax]")
	for _, s := range ParserSyntax {
		log.Print(s)
	}
	log.Print(" ")
	log.Print("[serializer syntax]")
	for _, s := range SerializerSyntax {
		log.Print(s)
	}
	log.Print(" ")
}

func codec(s *Statement) (err error) {
	subj := s.Subject
	pred := s.Predicate
	obj := s.Object

	sbuf, err := subj.GobEncode()
	if err != nil {
		errs := fmt.Sprintf("GobEncode(%s): %s", subj, err)
		err = errors.New(errs)
		return
	}
	pbuf, err := pred.GobEncode()
	if err != nil {
		errs := fmt.Sprintf("GobEncode(%s): %s", pred, err)
		err = errors.New(errs)
		return
	}
	obuf, err := obj.GobEncode()
	if err != nil {
		errs := fmt.Sprintf("GobEncode(%s): %s", obj, err)
		err = errors.New(errs)
		return
	}

	subj2, err := TermDecode(sbuf)
	if err != nil {
		errs := fmt.Sprintf("TermDecode(%s): %s", subj, err)
		err = errors.New(errs)
		return
	}
	pred2, err := TermDecode(pbuf)
	if err != nil {
		errs := fmt.Sprintf("TermDecode(%s): %s", pred, err)
		err = errors.New(errs)
		return
	}
	obj2, err := TermDecode(obuf)
	if err != nil {
		errs := fmt.Sprintf("TermDecode(%s): %s", obj, err)
		err = errors.New(errs)
		return
	}

	if !subj.Equals(subj2) {
		errs := fmt.Sprintf("%s != %s", subj, subj2)
		err = errors.New(errs)
		return
	}
	if !pred.Equals(pred2) {
		errs := fmt.Sprintf("%s != %s", pred, pred2)
		err = errors.New(errs)
		return
	}
	if !obj.Equals(obj2) {
		errs := fmt.Sprintf("%s != %s", obj, obj2)
		err = errors.New(errs)
		return
	}

	s2 := &Statement{subj2, pred2, obj2, nil}
	if !s.Equals(s2) {
		errs := fmt.Sprintf("%s != %s", s, s2)
		err = errors.New(errs)
		return
	}

	ssbuf, err := s.GobEncode()
	if err != nil {
		errs := fmt.Sprintf("Statement.GobEncode(%s): %s", s, err)
		err = errors.New(errs)
		return
	}

	s3 := &Statement{}
	err = s3.GobDecode(ssbuf)
	if err != nil {
		errs := fmt.Sprintf("Statement.GobDecode(%s): %s", s, err)
		err = errors.New(errs)
		return
	}
	if !s.Equals(s3) {
		errs := fmt.Sprintf("%s != %s", s, s3)
		err = errors.New(errs)
		return
	}
	return
}

func TestRaptorParseFile(t *testing.T) {
	parser := NewParser("rdfxml")
	defer parser.Free()

	count := 0
	exp := 153
	out := parser.ParseFile("foaf.rdf", "")
	for {
		s, ok := <-out
		if !ok {
			break
		}
		count++
		err := codec(s)
		if err != nil {
			t.Fatal(err)
		}
	}
	if count != exp {
		t.Errorf("Expected %d statements got %d\n", count, exp)
	}
}

func TestRaptorParseUri(t *testing.T) {
	parser := NewParser("guess")
	defer parser.Free()

	count := 0
	out := parser.ParseUri("http://www.w3.org/People/Berners-Lee/card", "")
	for {
		s, ok := <-out
		if !ok {
			break
		}
		count++
		err := codec(s)
		if err != nil {
			t.Fatal(err)
		}
	}
	if count == 0 {
		t.Errorf("Expected to find some statements... maybe there is no network?")
	}
}

func TestRaptorParseBuf(t *testing.T) {
	turtle := `
@prefix ex: <http://example.org/>.

ex:foo ex:p1 [ ex:bar "hello" ].
_:b1 ex:p2 ex:foo.
`
	parser := NewParser("turtle")
	defer parser.Free()

	count := 0
	for s := range parser.Parse(strings.NewReader(turtle), "http://example.org/") {
		count++
		err := codec(s)
		if err != nil {
			t.Fatal(err)
		}
	}
	if count != 3 {
		t.Errorf("Expected to find three statements")
	}
}

func TestRaptorSerializeFile(t *testing.T) {
	parser := NewParser("rdfxml")
	defer parser.Free()

	serializer := NewSerializer("turtle")
	defer serializer.Free()

	fp, err := os.OpenFile("/dev/null", os.O_WRONLY, 0644)
	if err != nil {
		t.Fatalf("Open(/dev/null): %s", err)
	}
	defer fp.Close()

	err = serializer.SetFile(fp, "")
	if err != nil {
		t.Fatalf("SetFile(%s, \"\"): %s", fp, err)
	}

	statements := parser.ParseFile("foaf.rdf", "")
	serializer.AddN(statements)
}

func TestRaptorSerializeString(t *testing.T) {
	parser := NewParser("rdfxml")
	defer parser.Free()

	serializer := NewSerializer("turtle")
	defer serializer.Free()

	parser.SetNamespaceHandler(func(prefix, uri string) { serializer.SetNamespace(prefix, uri) })

	statements := parser.ParseFile("foaf.rdf", "")
	str, err := serializer.Serialize(statements, "")
	if err != nil {
		t.Fatalf("Serialize(): %s", err)
	}
	if len(str) == 0 {
		t.Errorf("serialize to string failed, got empty string")
	}
	//	log.Print(str)
}

func TestTiger(t *testing.T) {
	parser := NewParser("ntriples")
	ch := parser.ParseFile("TGR06001.nt", "")
	count := 0
	start := time.Now()
	for {
		s, ok := <-ch
		if !ok {
			break
		}
		_ = fmt.Sprintf("%s", s)
		count++
		if count%10000 == 0 {
			end := time.Now()
			dt := uint64(count) * 1e9 / uint64(end.Sub(start))
			log.Printf("%d statements loaded at %d tps", count, dt)
		}
	}
	end := time.Now()
	dt := uint64(count) * 1e9 / uint64(end.Sub(start))
	log.Printf("%d statements loaded at %d tps", count, dt)
}

func benchParse() {
	parser := NewParser("rdfxml")
	out := parser.ParseFile("foaf.rdf", "")
	for {
		s, ok := <-out
		if !ok {
			break
		}
		codec(s)
		_ = s
	}
	parser.Free()
}

func BenchmarkCodecMemory(b *testing.B) {
	m := new(runtime.MemStats)

	for i := 0; i < b.N; i++ {
		runtime.ReadMemStats(m)
		log.Printf("start alloc: %d total: %d heap: %d",
			m.Alloc,
			m.TotalAlloc,
			m.HeapAlloc)
		benchParse()
		Reset()
		runtime.GC()
		runtime.ReadMemStats(m)
		log.Printf("  end alloc: %d total: %d heap: %d",
			m.Alloc,
			m.TotalAlloc,
			m.HeapAlloc)

	}
}
