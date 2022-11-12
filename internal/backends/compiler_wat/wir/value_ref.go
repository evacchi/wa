// 版权 @2022 凹语言 作者。保留所有权利。

package wir

import (
	"strconv"

	"github.com/wa-lang/wa/internal/backends/compiler_wat/wir/wat"
	"github.com/wa-lang/wa/internal/logger"
)

/**************************************
Ref:
**************************************/
type Ref struct {
	Base ValueType
	Struct
}

func NewRef(base ValueType) Ref {
	var v Ref
	v.Base = base
	var m []Field
	m = append(m, NewField("block", NewBlock(base)))
	m = append(m, NewField("data", NewPointer(base)))
	v.Struct = NewStruct(base.Name()+".$$ref", m)
	return v
}
func (t Ref) Name() string         { return t.Base.Name() + ".$$ref" }
func (t Ref) size() int            { return 8 }
func (t Ref) align() int           { return 4 }
func (t Ref) onFree() int          { return t.Struct.onFree() }
func (t Ref) Raw() []wat.ValueType { return t.Struct.Raw() }
func (t Ref) Equal(u ValueType) bool {
	if ut, ok := u.(Ref); ok {
		return t.Base.Equal(ut.Base)
	}
	return false
}

func (t Ref) emitHeapAlloc() (insts []wat.Inst) {
	//insts = append(insts, wat.NewBlank())
	//insts = append(insts, wat.NewComment("Ref.emitHeapAlloc start"))

	insts = append(insts, NewBlock(t.Base).emitHeapAlloc(NewConst("1", U32{}))...)
	insts = append(insts, wat.NewInstCall("$wa.RT.DupWatStack"))
	insts = append(insts, NewConst("16", U32{}).EmitPush()...)
	insts = append(insts, wat.NewInstAdd(wat.U32{}))

	//insts = append(insts, wat.NewComment("Ref.emitHeapAlloc end"))
	//insts = append(insts, wat.NewBlank())

	return
}

func (t Ref) emitStackAlloc() (insts []wat.Inst) {
	//insts = append(insts, wat.NewBlank())
	//insts = append(insts, wat.NewComment("Ref.emitStackAlloc start"))

	logger.Fatal("Todo")

	insts = append(insts, NewConst("0", U32{}).EmitPush()...)
	insts = append(insts, NewConst(strconv.Itoa(t.Base.size()), U32{}).EmitPush()...)
	insts = append(insts, wat.NewInstCall("$waStackAlloc"))

	//insts = append(insts, wat.NewComment("Ref.emitStackAlloc end"))
	//insts = append(insts, wat.NewBlank())
	return
}

func (t Ref) emitLoadFromAddr(addr Value, offset int) []wat.Inst {
	return t.Struct.emitLoadFromAddr(addr, offset)
}

/**************************************
aRef:
**************************************/
type aRef struct {
	aValue
	underlying aStruct
}

func newValueRef(name string, kind ValueKind, base_type ValueType) *aRef {
	var v aRef
	ref_type := NewRef(base_type)
	v.aValue = aValue{name: name, kind: kind, typ: ref_type}
	v.underlying = *newValueStruct(name, kind, ref_type.Struct)
	return &v
}

func (v *aRef) raw() []wat.Value        { return v.underlying.raw() }
func (v *aRef) EmitInit() []wat.Inst    { return v.underlying.EmitInit() }
func (v *aRef) EmitPush() []wat.Inst    { return v.underlying.EmitPush() }
func (v *aRef) EmitPop() []wat.Inst     { return v.underlying.EmitPop() }
func (v *aRef) EmitRelease() []wat.Inst { return v.underlying.EmitRelease() }

func (v *aRef) emitStoreToAddr(addr Value, offset int) []wat.Inst {
	return v.underlying.emitStoreToAddr(addr, offset)
}

func (v *aRef) emitGetValue() []wat.Inst {
	t := v.Type().(Ref).Base
	return t.emitLoadFromAddr(v.underlying.Extract("data"), 0)
}

func (v *aRef) emitSetValue(d Value) []wat.Inst {
	if !d.Type().Equal(v.Type().(Ref).Base) {
		logger.Fatal("Type not match")
		return nil
	}
	return d.emitStoreToAddr(v.underlying.Extract("data"), 0)
}
