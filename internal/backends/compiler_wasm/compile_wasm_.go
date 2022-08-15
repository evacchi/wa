// 版权 @2021 凹语言 作者。保留所有权利。

package compiler_wasm

import (
	"bytes"

	"github.com/wa-lang/wa/internal/3rdparty/wasm/encoding"
	"github.com/wa-lang/wa/internal/3rdparty/wasm/instruction"
	"github.com/wa-lang/wa/internal/3rdparty/wasm/module"
	"github.com/wa-lang/wa/internal/3rdparty/wasm/types"
	"github.com/wa-lang/wa/internal/ast"
	"github.com/wa-lang/wa/internal/loader"
)

type Compiler struct {
	stages []func() error

	prog *loader.Program

	globalNames []string
	globalNodes map[string]*ast.Ident

	module   *module.Module
	mainBody *module.CodeEntry

	builtinFdWriteIndex uint32
	builtinMainIndex    uint32
}

func New() *Compiler {
	p := new(Compiler)
	p.stages = []func() error{
		p.reset,
		p.initModule,
		p.emitGlobals,
		p.emitMainBody,
	}
	return p
}

func (p *Compiler) Compile(prog *loader.Program) (output string, err error) {
	p.prog = prog

	for _, stage := range p.stages {
		if err := stage(); err != nil {
			return "", err
		}
	}

	var buf bytes.Buffer
	if err := encoding.WriteModule(&buf, p.module); err != nil {
		return "", err
	}

	return buf.String(), nil
}

func (p *Compiler) reset() error {
	p.globalNames = []string{}
	p.globalNodes = make(map[string]*ast.Ident)
	return nil
}

func (p *Compiler) initModule() error {
	p.module = &module.Module{
		Names: module.NameSection{
			Module: "walang",
		},
		Memory: module.MemorySection{
			Memorys: []module.Memory{
				{InitPages: 1, MaxPages: 1},
			},
		},
		Export: module.ExportSection{
			Exports: []module.Export{
				{
					Name: "memory",
					Descriptor: module.ExportDescriptor{
						Type:  module.MemoryExportType,
						Index: 0,
					},
				},
			},
		},
	}

	// init types
	p.builtinFdWriteIndex = 0
	p.builtinMainIndex = 1

	/*
		// (import "wasi_unstable" "fd_write" (func $fd_write (param i32 i32 i32 i32) (result i32)))

		struct iov_t {
			const char* base;
			int len;
		};

		WASM_IMPORT("wasi_unstable")
		int fd_write(
			int fd, const struct iov_t* iovs, int iovs_len, int* nwritten
		);
	*/
	p.module.Type.Functions = []module.FunctionType{
		// func __wa_builtin_read() int32
		{
			Params: []types.ValueType{
				types.I32,
				types.I32,
				types.I32,
				types.I32,
			},
			Results: []types.ValueType{
				types.I32,
			},
		},
		// func _start
		{},
	}

	// import
	p.module.Import.Imports = []module.Import{
		{

			Module: "wasi_unstable",
			Name:   "fd_write",
			Descriptor: module.FunctionImport{
				Func: p.builtinFdWriteIndex,
			},
		},
	}

	// _start func
	p.module.Function.TypeIndices = []uint32{
		p.builtinMainIndex,
	}
	p.module.Names.Functions = append(p.module.Names.Functions, module.NameMap{
		Index: p.builtinMainIndex,
		Name:  "_start",
	})

	// _start func body
	{
		var entry = &module.CodeEntry{
			Func: module.Function{
				Locals: []module.LocalDeclaration{},
				Expr: module.Expr{
					Instrs: []instruction.Instruction{
						// instruction.I32Const{Value: 42},
						// instruction.Call{Index: p.builtinFdWriteIndex},
						instruction.Return{},
					},
				},
			},
		}

		var buf bytes.Buffer
		if err := encoding.WriteCodeEntry(&buf, entry); err != nil {
			return err
		}

		p.module.Code.Segments = append(p.module.Code.Segments, module.RawCodeSegment{
			Code: buf.Bytes(),
		})
	}

	// export _start
	p.module.Export.Exports = append(p.module.Export.Exports, module.Export{
		Name: "_start",
		Descriptor: module.ExportDescriptor{
			Type:  module.FunctionExportType,
			Index: p.builtinMainIndex,
		},
	})

	return nil
}

func (p *Compiler) emitGlobals() error {
	if len(p.module.Global.Globals) > 0 {
		return nil
	}

	// default global
	defauleGlobal := module.Global{
		Type: types.I32,
		Init: module.Expr{
			Instrs: []instruction.Instruction{
				instruction.I32Const{Value: 0},
			},
		},
	}

	// global[0] is nil
	p.module.Global.Globals = []module.Global{
		defauleGlobal,
	}

	return nil
}

func (p *Compiler) emitMainBody() error {
	p.mainBody = &module.CodeEntry{
		Func: module.Function{
			Locals: []module.LocalDeclaration{},
			Expr: module.Expr{
				Instrs: []instruction.Instruction{
					// instruction.I32Const{Value: 42 + 1},
					// instruction.Call{Index: p.builtinWriteIndex},
				},
			},
		},
	}

	if err := p.compileNone(&p.mainBody.Func.Expr, nil); err != nil {
		return err
	}

	var buf bytes.Buffer
	if err := encoding.WriteCodeEntry(&buf, p.mainBody); err != nil {
		return err
	}

	// replace main body
	p.module.Code.Segments = []module.RawCodeSegment{
		{Code: buf.Bytes()},
	}

	return nil
}

func (p *Compiler) compileNone(ctx *module.Expr, node ast.Node) error {
	if node == nil {
		return nil
	}

	panic("TODO: support WASM")
}

func (p *Compiler) globalIndexByName(name string) uint32 {
	for i, s := range p.globalNames {
		if s == name {
			return uint32(i) + 1
		}
	}
	return 0
}

func (p *Compiler) emit(ctx *module.Expr, x ...instruction.Instruction) {
	ctx.Instrs = append(ctx.Instrs, x...)
}
