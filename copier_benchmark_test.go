package nie

import (
	"testing"

	"google.golang.org/protobuf/types/known/structpb"
)

type bffPlainFrom struct {
	ID   int64
	Name string
	Tags []string
}

type bffPlainTo struct {
	ID   int64
	Name string
	Tags []string
}

type bffNestedFrom struct {
	Meta bffNestedMetaFrom
}

type bffNestedMetaFrom struct {
	PB *structpb.Struct
}

type bffNestedTo struct {
	Meta bffNestedMetaTo
}

type bffNestedMetaTo struct {
	PB *bffMeta
}

type bffMeta struct {
	A string `json:"a"`
	B int64  `json:"b"`
}

func TestCopier4Bff_SkipWhenNoStructPB(t *testing.T) {
	from := &bffPlainFrom{ID: 1, Name: "n", Tags: []string{"a", "b"}}
	to := &bffPlainTo{}
	if err := Copier4Bff(to, from); err != nil {
		t.Fatalf("Copier4Bff error: %v", err)
	}
	if to.ID != from.ID || to.Name != from.Name || len(to.Tags) != len(from.Tags) {
		t.Fatalf("unexpected copy result: %+v", *to)
	}
}

func TestCopier4Bff_NestedStructPBConverted(t *testing.T) {
	pb, err := structpb.NewStruct(map[string]interface{}{"a": "x", "b": float64(2)})
	if err != nil {
		t.Fatalf("NewStruct error: %v", err)
	}

	from := &bffNestedFrom{Meta: bffNestedMetaFrom{PB: pb}}
	to := &bffNestedTo{Meta: bffNestedMetaTo{PB: &bffMeta{}}}
	if err := Copier4Bff(to, from); err != nil {
		t.Fatalf("Copier4Bff error: %v", err)
	}
	if to.Meta.PB == nil || to.Meta.PB.A != "x" || to.Meta.PB.B != 2 {
		t.Fatalf("unexpected conversion result: %+v", to.Meta.PB)
	}
}

func BenchmarkCopier4Bff_NoStructPB(b *testing.B) {
	from := &bffPlainFrom{ID: 1, Name: "n", Tags: []string{"a", "b", "c"}}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		to := &bffPlainTo{}
		if err := Copier4Bff(to, from); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkCopier4Bff_NestedStructPB(b *testing.B) {
	pb, err := structpb.NewStruct(map[string]interface{}{"a": "x", "b": float64(2)})
	if err != nil {
		b.Fatal(err)
	}
	from := &bffNestedFrom{Meta: bffNestedMetaFrom{PB: pb}}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		to := &bffNestedTo{Meta: bffNestedMetaTo{PB: &bffMeta{}}}
		if err := Copier4Bff(to, from); err != nil {
			b.Fatal(err)
		}
	}
}
