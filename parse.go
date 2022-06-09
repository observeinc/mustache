// Copyright (c) 2014 Alex Kalyvitis

package mustache

import (
	"fmt"
)

type parser struct {
	lexer  *lexer
	escape escapeType
	buf    []token
}

// read returns the next token from the lexer and advances the cursor. This
// token will not be available by the parser after it has been read.
func (p *parser) read() token {
	if len(p.buf) > 0 {
		r := p.buf[0]
		p.buf = p.buf[1:]
		return r
	}
	return p.lexer.token()
}

// readt returns the tokens starting from the current position until the first
// match of t. Similar to readn it will return an error if a tokenEOF was
// returned by the lexer before a match was made.
func (p *parser) readt(t tokenType) ([]token, error) {
	var tokens []token
	for {
		token := p.read()
		tokens = append(tokens, token)
		switch token.typ {
		case tokenEOF:
			return tokens, fmt.Errorf("token %q not found", t)
		case t:
			return tokens, nil
		}
	}
}

// readv returns the tokens starting from the current position until the first
// match of t. A match is made only of t.typ and t.val are equal to the examined
// token.
func (p *parser) readv(t token) ([]token, error) {
	var tokens []token
	for {
		read, err := p.readt(t.typ)
		tokens = append(tokens, read...)
		if err != nil {
			return tokens, err
		}
		if len(read) > 0 && read[len(read)-1].val == t.val {
			break
		}
	}
	return tokens, nil
}

func (p *parser) errorf(t token, format string, v ...interface{}) error {
	return fmt.Errorf("%d:%d syntax error: %s", t.line, t.col, fmt.Sprintf(format, v...))
}

// parse begins parsing based on tokens read from the lexer.
func (p *parser) parse() ([]node, error) {
	var nodes []node
loop:
	for {
		token := p.read()
		switch token.typ {
		case tokenEOF:
			break loop
		case tokenError:
			return nil, p.errorf(token, "%s", token.val)
		case tokenText:
			nodes = append(nodes, textNode(token.val))
		case tokenLeftDelim:
			node, err := p.parseTag()
			if err != nil {
				return nodes, err
			}
			nodes = append(nodes, node)
		case tokenRawStart:
			node, err := p.parseRawTag()
			if err != nil {
				return nodes, err
			}
			nodes = append(nodes, node)
		case tokenSetDelim:
			nodes = append(nodes, new(delimNode))
		}
	}
	return nodes, nil
}

// parseTag parses a beginning of a mustache tag. It is assumed that a leftDelim
// was already read by the parser.
func (p *parser) parseTag() (node, error) {
	token := p.read()
	switch token.typ {
	case tokenIdentifier:
		return p.parseVar(token, p.escape)
	case tokenRawStart:
		return p.parseRawTag()
	case tokenRawAlt:
		return p.parseVar(p.read(), noEscape)
	case tokenComment:
		return p.parseComment()
	case tokenSectionInverse:
		return p.parseSection(true)
	case tokenSectionStart:
		return p.parseSection(false)
	case tokenTestValue:
		return p.parseTest()
	case tokenPartial:
		return p.parsePartial()
	}
	return nil, p.errorf(token, "unreachable code %s", token)
}

// parseRawTag parses a simple variable tag. It is assumed that the read from
// the parser should return an identifier.
func (p *parser) parseRawTag() (node, error) {
	t := p.read()
	if t.typ != tokenIdentifier {
		return nil, p.errorf(t, "unexpected token %s", t)
	}
	if next := p.read(); next.typ != tokenRawEnd {
		return nil, p.errorf(t, "unexpected token %s", t)
	}
	if next := p.read(); next.typ != tokenRightDelim {
		return nil, p.errorf(t, "unexpected token %s", t)
	}
	return &varNode{name: t.val, escape: noEscape}, nil
}

// parseVar parses a simple variable tag. It is assumed that the read from the
// parser should return an identifier.
func (p *parser) parseVar(ident token, escape escapeType) (node, error) {
	if t := p.read(); t.typ != tokenRightDelim {
		return nil, p.errorf(t, "unexpected token %s", t)
	}
	return &varNode{name: ident.val, escape: escape}, nil
}

// parseComment parses a comment block. It is assumed that the next read should
// return a t_comment token.
func (p *parser) parseComment() (node, error) {
	var comment string
	for {
		t := p.read()
		switch t.typ {
		case tokenEOF:
			return nil, p.errorf(t, "unexpected token %s", t)
		case tokenError:
			return nil, p.errorf(t, t.val)
		case tokenRightDelim:
			return commentNode(comment), nil
		default:
			comment += t.val
		}
	}
}

// parseSection parses a section block. It is assumed that the next read should
// return a t_section token.
func (p *parser) parseSection(inverse bool) (node, error) {
	t := p.read()
	if t.typ != tokenIdentifier {
		return nil, p.errorf(t, "unexpected token %s", t)
	}

	nodes, err := p.parseSectionInternal(t)
	if err != nil {
		return nil, err
	}

	section := &sectionNode{
		name:     t.val,
		inverted: inverse,
		elems:    nodes,
	}
	return section, nil
}

// parsePartial parses a partial block. It is assumed that the next read should
// return a t_ident token.
func (p *parser) parsePartial() (node, error) {
	t := p.read()
	if t.typ != tokenIdentifier {
		return nil, p.errorf(t, "unexpected token %s", t)
	}
	if next := p.read(); next.typ != tokenRightDelim {
		return nil, p.errorf(t, "unexpected token %s", t)
	}
	return &partialNode{t.val}, nil
}

func (p *parser) parseSectionInternal(t token) ([]node, error) {

	if next := p.read(); next.typ != tokenRightDelim {
		return nil, p.errorf(next, "unexpected token %s", next)
	}

	var (
		tokens []token
		stack  = 1
	)
	for {
		read, err := p.readv(t)
		if err != nil {
			return nil, err
		}
		tokens = append(tokens, read...)
		if len(read) > 1 {
			// Check the token that preceded the matching identifier. For
			// section start and inverse tokens we increase the stack for sections or testValue for those special sections, otherwise
			// decrease.
			tt := read[len(read)-2]
			switch {
			case tt.typ == tokenSectionStart || tt.typ == tokenTestValue || tt.typ == tokenSectionInverse:
				stack++
			case tt.typ == tokenSectionEnd:
				stack--
			}
		}
		if stack == 0 {
			break
		}
	}
	nodes, err := subParser(tokens[:len(tokens)-3], p.escape).parse()
	if err != nil {
		return nil, err
	}

	return nodes, nil
}

// parseSection parses a test_Value block. It is assumed that the next read should
// return a t_section token.
func (p *parser) parseTest() (node, error) {
	t := p.read()
	if t.typ != tokenIdentifier {
		return nil, p.errorf(t, "unexpected token %s", t)
	}
	if next := p.read(); next.typ != tokenLeftDelim {
		return nil, p.errorf(next, "unexpected token %s", next)
	}
	i := p.read()
	if i.typ != tokenIdentifier {
		return nil, p.errorf(i, "unexpected token %s", i)
	}
	if next := p.read(); next.typ != tokenRightDelim {
		return nil, p.errorf(next, "unexpected token %s", next)
	}

	v := p.read()
	if v.typ != tokenText {
		return nil, p.errorf(v, "unexpected token %s", v)
	}

	nodes, err := p.parseSectionInternal(t)
	if err != nil {
		return nil, err
	}

	section := &testNode{
		testIdent: i.val,
		testVal:   v.val,
		elems:     nodes,
	}
	return section, nil
}

// newParser creates a new parser using the suppliad lexer.
func newParser(l *lexer, escape escapeType) *parser {
	return &parser{lexer: l, escape: escape}
}

// subParser creates a new parser with a pre-defined token buffer.
func subParser(b []token, escape escapeType) *parser {
	return &parser{buf: append(b, token{typ: tokenEOF}), escape: escape}
}
