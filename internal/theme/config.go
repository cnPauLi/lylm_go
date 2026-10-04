package theme

import (
	"regexp"
	"strings"
)

// EnumOption 下拉/单选/多选的枚举项
type EnumOption struct {
	Value string
	Label string
}

// Field 主题自定义配置项（对应 config.php 中 $theme_config 的一个元素）
type Field struct {
	Type        string
	Name        string
	Title       string
	Description string
	Verify      string
	Placeholder string
	Value       any // string 或 []string
	Enum        []EnumOption
}

// DefaultValue 返回默认值字符串形式
func (f Field) DefaultValue() string {
	switch v := f.Value.(type) {
	case string:
		return v
	case []string:
		return strings.Join(v, ",")
	default:
		return ""
	}
}

// DefaultValueList 返回多选默认值
func (f Field) DefaultValueList() []string {
	if v, ok := f.Value.([]string); ok {
		return v
	}
	if s, ok := f.Value.(string); ok && s != "" {
		return strings.Split(s, ",")
	}
	return nil
}

// phpValue 解析后的 PHP 值类型
type phpKV struct {
	key any
	val any
}

type phpArray struct {
	items []phpKV
}

type phpExpr struct {
	raw string
}

// ParseConfig 解析主题 config.php，提取 $theme_config 数组定义
func ParseConfig(src string, conf map[string]string) []Field {
	loc := assignRe.FindStringSubmatchIndex(src)
	if loc == nil {
		return nil
	}
	// loc[2]:loc[3] 为数组起始位置（'[' 或 'array('）
	toks, err := tokenize(src[loc[2]:])
	if err != nil {
		return nil
	}
	p := &parser{toks: toks}
	v := p.parseValue()
	arr, ok := v.(*phpArray)
	if !ok {
		return nil
	}
	fields := make([]Field, 0, len(arr.items))
	for _, kv := range arr.items {
		if sub, ok := kv.val.(*phpArray); ok {
			fields = append(fields, arrayToField(sub, conf))
		}
	}
	return fields
}

// assignRe 精确匹配 $theme_config = [...] 或 $theme_config = array(...)
var assignRe = regexp.MustCompile(`\$theme_config\s*=\s*(\[|array\s*\()`)

// ===== 词法分析 =====

const (
	tokStr = iota
	tokNum
	tokLBracket
	tokRBracket
	tokLParen
	tokRParen
	tokArrow
	tokComma
	tokExpr
	tokIdent
)

type token struct {
	kind int
	text string
}

func isDigit(c byte) bool { return c >= '0' && c <= '9' }
func isAlpha(c byte) bool {
	return (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || c == '_'
}

func unescPhp(c byte) byte {
	switch c {
	case 'n':
		return '\n'
	case 'r':
		return '\r'
	case 't':
		return '\t'
	case '0':
		return 0
	case '\\', '\'', '"':
		return c
	default:
		return c
	}
}

func tokenize(s string) ([]token, error) {
	var toks []token
	i, n := 0, len(s)
	for i < n {
		c := s[i]
		switch {
		case c == ' ' || c == '\t' || c == '\r' || c == '\n':
			i++
		case c == '/' && i+1 < n && s[i+1] == '/':
			for i < n && s[i] != '\n' {
				i++
			}
		case c == '#':
			for i < n && s[i] != '\n' {
				i++
			}
		case c == '/' && i+1 < n && s[i+1] == '*':
			i += 2
			for i+1 < n && !(s[i] == '*' && s[i+1] == '/') {
				i++
			}
			i += 2
		case c == '\'' || c == '"':
			quote := c
			i++
			var sb strings.Builder
			for i < n {
				if s[i] == '\\' && i+1 < n {
					sb.WriteByte(unescPhp(s[i+1]))
					i += 2
					continue
				}
				if s[i] == quote {
					i++
					break
				}
				sb.WriteByte(s[i])
				i++
			}
			toks = append(toks, token{tokStr, sb.String()})
		case c == '[':
			toks = append(toks, token{tokLBracket, "["})
			i++
		case c == ']':
			toks = append(toks, token{tokRBracket, "]"})
			i++
		case c == '(':
			toks = append(toks, token{tokLParen, "("})
			i++
		case c == ')':
			toks = append(toks, token{tokRParen, ")"})
			i++
		case c == ',':
			toks = append(toks, token{tokComma, ","})
			i++
		case c == '=' && i+1 < n && s[i+1] == '>':
			toks = append(toks, token{tokArrow, "=>"})
			i += 2
		case c == '$':
			start := i
			depth := 0
			for i < n {
				ch := s[i]
				if ch == '\'' {
					i++
					for i < n {
						if s[i] == '\\' {
							i += 2
							continue
						}
						if s[i] == '\'' {
							i++
							break
						}
						i++
					}
					continue
				}
				if ch == '[' {
					depth++
					i++
					continue
				}
				if ch == ']' {
					if depth == 0 {
						break
					}
					depth--
					i++
					continue
				}
				if depth == 0 && (ch == ',' || ch == ')') {
					break
				}
				i++
			}
			toks = append(toks, token{tokExpr, strings.TrimSpace(s[start:i])})
		default:
			if isDigit(c) {
				start := i
				for i < n && (isDigit(s[i]) || s[i] == '.') {
					i++
				}
				toks = append(toks, token{tokNum, s[start:i]})
			} else if isAlpha(c) {
				start := i
				for i < n && (isAlpha(s[i]) || isDigit(s[i])) {
					i++
				}
				toks = append(toks, token{tokIdent, s[start:i]})
			} else {
				i++
			}
		}
	}
	return toks, nil
}

// ===== 语法分析 =====

type parser struct {
	toks []token
	pos  int
}

func (p *parser) peek() *token {
	if p.pos < len(p.toks) {
		return &p.toks[p.pos]
	}
	return nil
}

func (p *parser) next() *token {
	t := p.peek()
	if t != nil {
		p.pos++
	}
	return t
}

func (p *parser) parseValue() any {
	t := p.peek()
	if t == nil {
		return nil
	}
	switch t.kind {
	case tokStr:
		p.next()
		return t.text
	case tokNum:
		p.next()
		return t.text
	case tokExpr:
		p.next()
		return phpExpr{raw: t.text}
	case tokLBracket:
		p.next()
		return p.parseArray(tokRBracket)
	case tokIdent:
		if strings.EqualFold(t.text, "array") {
			p.next()
			if lp := p.peek(); lp != nil && lp.kind == tokLParen {
				p.next()
				return p.parseArray(tokRParen)
			}
		}
		p.next()
		return nil
	default:
		p.next()
		return nil
	}
}

func (p *parser) parseArray(endKind int) *phpArray {
	arr := &phpArray{}
	for {
		t := p.peek()
		if t == nil || t.kind == endKind {
			p.next()
			break
		}
		if t.kind == tokComma {
			p.next()
			continue
		}
		val := p.parseValue()
		if nxt := p.peek(); nxt != nil && nxt.kind == tokArrow {
			p.next() // =>
			v2 := p.parseValue()
			arr.items = append(arr.items, phpKV{key: val, val: v2})
		} else {
			arr.items = append(arr.items, phpKV{key: nil, val: val})
		}
	}
	return arr
}

// ===== 转换为 Field =====

func arrayToField(a *phpArray, conf map[string]string) Field {
	f := Field{}
	for _, kv := range a.items {
		key, _ := kv.key.(string)
		switch key {
		case "type":
			f.Type = toStr(kv.val, conf)
		case "name":
			f.Name = toStr(kv.val, conf)
		case "title":
			f.Title = toStr(kv.val, conf)
		case "description":
			f.Description = toStr(kv.val, conf)
		case "verify":
			f.Verify = toStr(kv.val, conf)
		case "placeholder":
			f.Placeholder = toStr(kv.val, conf)
		case "value":
			f.Value = toValue(kv.val, conf)
		case "enum":
			f.Enum = arrayToEnum(kv.val, conf)
		}
	}
	return f
}

func arrayToEnum(v any, conf map[string]string) []EnumOption {
	arr, ok := v.(*phpArray)
	if !ok {
		return nil
	}
	var opts []EnumOption
	for _, kv := range arr.items {
		value := toStr(kv.key, conf)
		label := toStr(kv.val, conf)
		opts = append(opts, EnumOption{Value: value, Label: label})
	}
	return opts
}

func toValue(v any, conf map[string]string) any {
	switch t := v.(type) {
	case *phpArray:
		var list []string
		for _, kv := range t.items {
			list = append(list, toStr(kv.val, conf))
		}
		return list
	default:
		return toStr(v, conf)
	}
}

func toStr(v any, conf map[string]string) string {
	switch t := v.(type) {
	case nil:
		return ""
	case string:
		return t
	case int64:
		return itoa(t)
	case phpExpr:
		return resolveExpr(t.raw, conf)
	case *phpArray:
		// 数组转字符串时取第一个元素
		if len(t.items) > 0 {
			return toStr(t.items[0].val, conf)
		}
		return ""
	default:
		return ""
	}
}

func itoa(n int64) string {
	if n == 0 {
		return "0"
	}
	neg := n < 0
	if neg {
		n = -n
	}
	var b [20]byte
	i := len(b)
	for n > 0 {
		i--
		b[i] = byte('0' + n%10)
		n /= 10
	}
	if neg {
		i--
		b[i] = '-'
	}
	return string(b[i:])
}

// resolveExpr 解析 PHP 表达式，仅支持 $GLOBALS['conf']['xxx'] 形式
func resolveExpr(raw string, conf map[string]string) string {
	if !strings.Contains(raw, "$GLOBALS") {
		return ""
	}
	parts := extractQuoted(raw)
	for i := 0; i+1 < len(parts); i++ {
		if parts[i] == "conf" {
			if v, ok := conf[parts[i+1]]; ok {
				return v
			}
			return ""
		}
	}
	return ""
}

func extractQuoted(s string) []string {
	var out []string
	i, n := 0, len(s)
	for i < n {
		if s[i] == '\'' || s[i] == '"' {
			quote := s[i]
			i++
			start := i
			for i < n && s[i] != quote {
				i++
			}
			out = append(out, s[start:i])
			i++
			continue
		}
		i++
	}
	return out
}
