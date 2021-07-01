package parse

import (
	"errors"
	"fmt"

	"github.com/genshinsim/gsim/pkg/def"
)

func parseAction(p *Parser) (parseFn, error) {
	var err error
	var item item
	var r def.Action
	r.Raw = tokensToStringArray(p.tokens)
	err = p.parseActionDetails(&r)
	if err != nil {
		return nil, err
	}
loop:
	for n := p.next(); n.typ != itemEOF; n = p.next() {
		switch n.typ {
		case itemLabel:
			item, err = p.acceptSeqReturnLast(itemAssign, itemIdentifier)
			r.Name = item.val
		case itemOnce:
			item, err = p.acceptSeqReturnLast(itemAssign, itemBool)
			if item.val == "true" {
				r.Once = true
			}
		case itemActionLock:
			item, err = p.acceptSeqReturnLast(itemAssign, itemNumber)
			if err == nil {
				r.ActionLock, err = itemNumberToInt(item)
			}
		case itemTarget:
			item, err = p.acceptSeqReturnLast(itemAssign, itemIdentifier)
			r.Target = item.val
		case itemExec:
			r.Exec, err = p.parseExec()
		case itemLock:
			item, err = p.acceptSeqReturnLast(itemAssign, itemNumber)
			if err == nil {
				r.SwapLock, err = itemNumberToInt(item)
			}
		case itemIf:
			r.Conditions, err = p.parseIf()
		case itemSwap:
			item, err = p.acceptSeqReturnLast(itemAssign, itemIdentifier)
			r.SwapTo = item.val
		case itemPost:
			r.PostAction, err = p.parsePostAction()
		case itemActive:
			item, err = p.acceptSeqReturnLast(itemAssign, itemIdentifier)
			r.ActiveCond = item.val
		case itemTerminateLine:
			break loop
		default:
			err = fmt.Errorf("bad token at line %v - %v: %v", n.line, n.pos, n)
		}
		if err != nil {
			return nil, err
		}
	}

	if err := isActionValid(r); err != nil {
		return nil, fmt.Errorf("bad action: %v", err)
	}

	p.result.Rotation = append(p.result.Rotation, r)

	return parseRows, nil
}

func (p *Parser) parseActionDetails(next *def.Action) error {
	//next should be a keyword
	n, err := p.consume(itemIdentifier)
	if err != nil {
		return err
	}

	t, ok := actionKeys[n.val]
	if !ok {
		return fmt.Errorf("<action> invalid identifier at line %v: %v", n.line, n)
	}
	a := def.ActionItem{}
	a.Param = make(map[string]int)
	switch {
	case t == def.ActionSequence:
		next.IsSeq = true
	case t == def.ActionSequenceStrict:
		next.IsSeq = true
		next.IsStrict = true
	case t > def.ActionDelimiter:
		a.Typ = t
		//check for params
		n = p.next()
		if n.typ != itemLeftSquareParen {
			p.backup()
			next.Exec = append(next.Exec, a)
			return nil
		}
		// log.Println(n)

		err = p.parseActionItemParams(&a)
		if err != nil {
			return err
		}

		// log.Println(n)
		next.Exec = append(next.Exec, a)

	}

	return nil
}

func (p *Parser) parseActionItemParams(a *def.ActionItem) error {
	for {
		//we're expecting ident = int
		i, err := p.consume(itemIdentifier)
		if err != nil {
			return err
		}

		item, err := p.acceptSeqReturnLast(itemAssign, itemNumber)
		if err != nil {
			return err
		}

		a.Param[i.val], err = itemNumberToInt(item)
		if err != nil {
			return err
		}

		//if we hit ], return; if we hit , keep going, other wise error
		n := p.next()
		switch n.typ {
		case itemRightSquareParen:
			return nil
		case itemComma:
			//do nothing, keep going
		default:
			return fmt.Errorf("<action item param> bad token at line %v - %v: %v", n.line, n.pos, n)
		}
	}
}

func (p *Parser) parseExec() ([]def.ActionItem, error) {
	var r []def.ActionItem

	_, err := p.consume(itemAssign)
	if err != nil {
		return nil, err
	}

LOOP:
	for {
		n, err := p.consume(itemIdentifier)
		if err != nil {
			return nil, err
		}
		t, ok := actionKeys[n.val]
		if !ok {
			return nil, fmt.Errorf("<exec> bad token at line %v - %v: %v", n.line, n.pos, n)
		}
		if t <= def.ActionDelimiter {
			return nil, fmt.Errorf("<exec> bad token at line %v - %v: %v", n.line, n.pos, n)
		}

		a := def.ActionItem{}
		a.Typ = t
		a.Param = make(map[string]int)
		//check for params
		n = p.next()

		if n.typ == itemLeftSquareParen {

			err := p.parseActionItemParams(&a)
			if err != nil {
				return nil, err
			}
		} else {
			p.backup()
		}

		r = append(r, a)
		n = p.next()
		if n.typ != itemComma {
			p.backup()
			break LOOP
		}
	}

	return r, nil
}

func (p *Parser) parseIf() (*def.ExprTreeNode, error) {
	n, err := p.consume(itemAssign)
	if err != nil {
		return nil, err
	}

	parenDepth := 0
	var queue []*def.ExprTreeNode
	var stack []*def.ExprTreeNode
	var x *def.ExprTreeNode
	var root *def.ExprTreeNode

	//operands are conditions
	//operators are &&, ||, (, )
LOOP:
	for {
		//we expect either brackets, or a field
		n = p.next()
		switch {
		case n.typ == itemLeftParen:
			parenDepth++
			stack = append(stack, &def.ExprTreeNode{
				Op: "(",
			})
			//expecting a condition after a paren
			c, err := p.parseCondition()
			if err != nil {
				return nil, err
			}
			queue = append(queue, &def.ExprTreeNode{
				Expr:   c,
				IsLeaf: true,
			})
		case n.typ == itemRightParen:

			parenDepth--
			if parenDepth < 0 {
				return nil, fmt.Errorf("unmatched right paren")
			}
			/**
			Else if token is a right parenthesis
				Until the top token (from the stack) is left parenthesis, pop from the stack to the output buffer
				Also pop the left parenthesis but don’t include it in the output buffe
			**/

			for {
				x, stack = stack[len(stack)-1], stack[:len(stack)-1]
				if x.Op == "(" {
					break
				}
				queue = append(queue, x)
			}

		case n.typ == itemField:
			p.backup()
			//scan for fields
			c, err := p.parseCondition()
			if err != nil {
				return nil, err
			}
			queue = append(queue, &def.ExprTreeNode{
				Expr:   c,
				IsLeaf: true,
			})
		}

		//check if any logical ops
		n = p.next()
		switch {
		case n.typ > itemLogicOP && n.typ < itemCompareOp:
			//check if top of stack is an operator
			if len(stack) > 0 && stack[len(stack)-1].Op != "(" {
				//pop and add to output
				x, stack = stack[len(stack)-1], stack[:len(stack)-1]
				queue = append(queue, x)
			}
			//append current operator to stack
			stack = append(stack, &def.ExprTreeNode{
				Op: n.val,
			})
		case n.typ == itemRightParen:
			p.backup()
		default:
			p.backup()
			break LOOP
		}
	}

	if parenDepth > 0 {
		return nil, fmt.Errorf("unmatched left paren")
	}

	for i := len(stack) - 1; i >= 0; i-- {
		queue = append(queue, stack[i])
	}

	var ts []*def.ExprTreeNode
	//convert this into a tree
	for _, v := range queue {
		if v.Op != "" {
			// fmt.Printf("%v ", v.Op)
			//pop 2 nodes from tree
			if len(ts) < 2 {
				return nil, errors.New("tree stack less than 2 before operator")
			}
			v.Left, ts = ts[len(ts)-1], ts[:len(ts)-1]
			v.Right, ts = ts[len(ts)-1], ts[:len(ts)-1]
			ts = append(ts, v)
		} else {
			// fmt.Printf("%v ", v.Expr)
			ts = append(ts, v)
		}
	}
	// fmt.Printf("\n")

	root = ts[0]
	return root, nil
}

func (p *Parser) parseCondition() (def.Condition, error) {
	var c def.Condition
	var n item
LOOP:
	for {
		//look for a field
		n = p.next()
		if n.typ != itemField {
			return c, fmt.Errorf("<if - field> bad token at line %v - %v: %v", n.line, n.pos, n)
		}
		c.Fields = append(c.Fields, n.val)
		//see if any more fields
		n = p.peek()
		if n.typ != itemField {
			break LOOP
		}
	}

	//scan for comparison op
	n = p.next()
	if n.typ <= itemCompareOp || n.typ >= itemKeyword {
		return c, fmt.Errorf("<if - comp> bad token at line %v - %v: %v", n.line, n.pos, n)
	}
	c.Op = n.val
	//scan for value
	n, err := p.consume(itemNumber)
	if err != nil {
		return c, err
	}
	c.Value, err = itemNumberToInt(n)

	return c, err
}

func (p *Parser) parsePostAction() (def.ActionType, error) {
	var t def.ActionType

	n, err := p.acceptSeqReturnLast(itemAssign, itemIdentifier)
	if err != nil {
		return t, err
	}

	t, ok := actionKeys[n.val]
	if !ok {
		return t, fmt.Errorf("<post - val id> bad token at line %v - %v: %v", n.line, n.pos, n)
	}
	if t <= def.ActionCancellable {
		return t, fmt.Errorf("<post - cancel> invalid post action at line %v - %v: %v", n.line, n.pos, n)
	}
	return t, nil
}

func parseActiveChar(p *Parser) (parseFn, error) {
	n, err := p.consume(itemIdentifier)
	if err != nil {
		return nil, err
	}
	p.result.Characters.Initial = n.val
	_, err = p.consume(itemTerminateLine)
	if err != nil {
		return nil, err
	}
	return parseRows, nil
}

func isActionValid(a def.Action) error {
	if a.Target == "" {
		return errors.New("missing target")
	}
	if len(a.Exec) == 0 {
		return errors.New("missing actions")
	}
	return nil
}