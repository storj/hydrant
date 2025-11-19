package filter

import "fmt"

type inst struct {
	_ [0]func() // no equality

	op  byte
	arg uint32
}

const (
	// nop
	instNop = iota

	// function calls
	instPushStr   // push(strs[arg])
	instPushFloat // push(floats[arg])
	instCall      // push(call(funcs[i.arg1]))

	// conjunctions
	instAnd // push(pop() && pop())
	instOr  // push(pop() || pop())
)

func (i inst) String() string {
	switch i.op {
	case instNop:
		return "nop"
	case instPushStr:
		return fmt.Sprintf("(pushLit %v)", token(i.arg))
	case instPushFloat:
		return fmt.Sprintf("(pushFloat %v)", token(i.arg))
	case instCall:
		return fmt.Sprintf("(call %d)", i.arg)
	case instAnd:
		return "and"
	case instOr:
		return "or"
	default:
		return fmt.Sprintf("(op%d %d)", i.op, i.arg)
	}
}
