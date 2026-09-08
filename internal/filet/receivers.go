package filet

import (
	"go/ast"
	"go/token"
)

func collectReceiver(receivers map[string]map[string]token.Pos, d *ast.FuncDecl) {
	if d.Recv == nil || len(d.Recv.List) == 0 || len(d.Recv.List[0].Names) == 0 {
		return
	}
	typeName := receiverType(d.Recv.List[0].Type)
	name := d.Recv.List[0].Names[0].Name
	if receivers[typeName] == nil {
		receivers[typeName] = map[string]token.Pos{}
	}
	if _, seen := receivers[typeName][name]; !seen {
		receivers[typeName][name] = d.Pos()
	}
}
