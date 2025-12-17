package nie

import (
	"database/sql"
	"encoding/json"
	"reflect"
	"testing"
	"time"

	"google.golang.org/protobuf/types/known/structpb"
	"gorm.io/datatypes"
)

type entFromLabelsSlice struct {
	Labels []string
}

type entToLabelsJSON struct {
	Labels datatypes.JSON
}

type entFromMetaPB struct {
	Meta *structpb.Struct
}

type entToMetaJSON struct {
	Meta datatypes.JSON
}

type entFromMetasPB struct {
	Metas []*structpb.Struct
}

type entToMetasJSON struct {
	Metas datatypes.JSON
}

type entFromLabelsJSON struct {
	Labels datatypes.JSON
}

type entToLabelsSlice struct {
	Labels []string
}

type entFrom struct {
	CreatedAt time.Time
	DeletedAt sql.NullTime
	Labels    datatypes.JSON
	Meta      *structpb.Struct
	Metas     []*structpb.Struct
}

type entTo struct {
	CreatedAt string
	DeletedAt string
	Labels    []string
	Meta      datatypes.JSON
	Metas     datatypes.JSON
}

func TestCopier4Ent_BasicConverters(t *testing.T) {
	pb, err := structpb.NewStruct(map[string]interface{}{"k": "v", "n": float64(1)})
	if err != nil {
		t.Fatalf("NewStruct error: %v", err)
	}
	from := &entFrom{
		CreatedAt: time.Date(2025, 12, 16, 10, 11, 12, 0, time.UTC),
		DeletedAt: sql.NullTime{Time: time.Date(2025, 12, 1, 0, 0, 0, 0, time.UTC), Valid: true},
		Labels:    datatypes.JSON([]byte("[\"a\",\"b\"]")),
		Meta:      pb,
		Metas:     []*structpb.Struct{pb},
	}

	to := &entTo{}
	if err := Copier4Ent(to, from); err != nil {
		t.Fatalf("Copier4Ent error: %v", err)
	}
	if to.CreatedAt == "" {
		t.Fatalf("expected CreatedAt to be set")
	}
	if to.DeletedAt == "" {
		t.Fatalf("expected DeletedAt to be set")
	}
	if len(to.Labels) != 2 {
		t.Fatalf("expected Labels len=2, got %d", len(to.Labels))
	}
	if len(to.Meta) == 0 {
		t.Fatalf("expected Meta JSON not empty")
	}
	if len(to.Metas) == 0 {
		t.Fatalf("expected Metas JSON not empty")
	}
}

func BenchmarkCopier4Ent_WithConverters(b *testing.B) {
	pb, err := structpb.NewStruct(map[string]interface{}{"k": "v", "n": float64(1)})
	if err != nil {
		b.Fatal(err)
	}
	from := &entFrom{
		CreatedAt: time.Date(2025, 12, 16, 10, 11, 12, 0, time.UTC),
		DeletedAt: sql.NullTime{Time: time.Date(2025, 12, 1, 0, 0, 0, 0, time.UTC), Valid: true},
		Labels:    datatypes.JSON([]byte("[\"a\",\"b\"]")),
		Meta:      pb,
		Metas:     []*structpb.Struct{pb},
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		to := &entTo{}
		if err := Copier4Ent(to, from); err != nil {
			b.Fatal(err)
		}
	}
}

func TestConverter_LabelsSliceToJSON_CompatWithEncodingJSON(t *testing.T) {
	converters := GetJSONConverters()
	srcT := reflect.TypeOf([]string{})
	dstT := reflect.TypeOf(datatypes.JSON{})

	var fn func(interface{}) (interface{}, error)
	for _, c := range converters {
		if reflect.TypeOf(c.SrcType) == srcT && reflect.TypeOf(c.DstType) == dstT {
			fn = c.Fn
			break
		}
	}
	if fn == nil {
		t.Fatalf("converter []string->datatypes.JSON not found")
	}

	cases := []struct {
		name string
		in   []string
	}{
		{"nil", nil},
		{"empty", []string{}},
		{"non-empty", []string{"a", "b", "c"}},
		{"escape", []string{"a\"b", "x\\y", "\n"}},
	}

	for _, tc := range cases {
		out, err := fn(tc.in)
		if err != nil {
			t.Fatalf("%s: error: %v", tc.name, err)
		}
		got := out.(datatypes.JSON)
		gold, _ := json.Marshal(tc.in)
		if string(got) != string(gold) {
			t.Fatalf("%s: mismatch: got=%s want=%s", tc.name, string(got), string(gold))
		}
	}
}

func TestConverter_JSONNullToLabelsSlice_ReturnsNil(t *testing.T) {
	from := &entFromLabelsJSON{Labels: datatypes.JSON([]byte("null"))}
	to := &entToLabelsSlice{}
	if err := Copier4Ent(to, from); err != nil {
		t.Fatalf("Copier4Ent error: %v", err)
	}
	if to.Labels != nil {
		t.Fatalf("expected Labels to be nil, got len=%d", len(to.Labels))
	}
}

func TestConverter_EmptyObjectToLabelsSlice_ReturnsNil(t *testing.T) {
	from := &entFromLabelsJSON{Labels: datatypes.JSON([]byte("{}"))}
	to := &entToLabelsSlice{}
	if err := Copier4Ent(to, from); err != nil {
		t.Fatalf("Copier4Ent error: %v", err)
	}
	if to.Labels != nil {
		t.Fatalf("expected Labels to be nil, got len=%d", len(to.Labels))
	}
}

func TestConverter_JSONNullToStructPBSlice_ReturnsNil(t *testing.T) {
	from := &entToMetasJSON{Metas: datatypes.JSON([]byte("null"))}
	to := &entFromMetasPB{}
	if err := Copier4Ent(to, from); err != nil {
		t.Fatalf("Copier4Ent error: %v", err)
	}
	if to.Metas != nil {
		t.Fatalf("expected Metas to be nil, got len=%d", len(to.Metas))
	}
}

func TestConverter_EmptyObjectToStructPBSlice_ReturnsNil(t *testing.T) {
	from := &entToMetasJSON{Metas: datatypes.JSON([]byte("{}"))}
	to := &entFromMetasPB{}
	if err := Copier4Ent(to, from); err != nil {
		t.Fatalf("Copier4Ent error: %v", err)
	}
	if to.Metas != nil {
		t.Fatalf("expected Metas to be nil, got len=%d", len(to.Metas))
	}
}

func BenchmarkCopier4Ent_LabelsSliceToJSON(b *testing.B) {
	from := &entFromLabelsSlice{Labels: []string{"a", "b", "c", "d", "e"}}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		to := &entToLabelsJSON{}
		if err := Copier4Ent(to, from); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkCopier4Ent_MetaStructPBToJSON(b *testing.B) {
	pb, err := structpb.NewStruct(map[string]interface{}{"k": "v", "n": float64(1)})
	if err != nil {
		b.Fatal(err)
	}
	from := &entFromMetaPB{Meta: pb}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		to := &entToMetaJSON{}
		if err := Copier4Ent(to, from); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkCopier4Ent_MetasSliceStructPBToJSON(b *testing.B) {
	pb, err := structpb.NewStruct(map[string]interface{}{"k": "v", "n": float64(1)})
	if err != nil {
		b.Fatal(err)
	}
	from := &entFromMetasPB{Metas: []*structpb.Struct{pb, pb, pb}}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		to := &entToMetasJSON{}
		if err := Copier4Ent(to, from); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkCopier4Ent_LabelsJSONToSlice(b *testing.B) {
	from := &entFromLabelsJSON{Labels: datatypes.JSON([]byte("[\"a\",\"b\",\"c\",\"d\",\"e\"]"))}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		to := &entToLabelsSlice{}
		if err := Copier4Ent(to, from); err != nil {
			b.Fatal(err)
		}
	}
}
