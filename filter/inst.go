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
	instPushStr // push(token(arg).literal(filter))
	instPushVal // push(vals[arg])
	instCall    // push(call(funcs[i.arg1]))

	// optimized instructions
	instKey // push(token(arg).literal(filter)); push(call(key))
	instHas // push(token(arg).literal(filter)); push(call(has))

	// conjunctions
	instAnd       // push(pop() && pop())
	instJumpFalse // if !peek() { pc += arg }
	instOr        // push(pop() || pop())
	instJumpTrue  // if peek() { pc += arg }
)

func (i inst) String() string {
	switch i.op {
	case instNop:
		return "nop"
	case instPushStr:
		return fmt.Sprintf("(pushStr %v)", token(i.arg))
	case instPushVal:
		return fmt.Sprintf("(pushVal %v)", i.arg)
	case instCall:
		return fmt.Sprintf("(call %d)", i.arg)
	case instKey:
		return fmt.Sprintf("(key %v)", token(i.arg))
	case instHas:
		return fmt.Sprintf("(has %v)", token(i.arg))
	case instAnd:
		return "and"
	case instJumpFalse:
		return fmt.Sprintf("(jumpFalse %d)", i.arg)
	case instOr:
		return "or"
	case instJumpTrue:
		return fmt.Sprintf("(jumpTrue %d)", i.arg)
	default:
		return fmt.Sprintf("(op%d %d)", i.op, i.arg)
	}
}
