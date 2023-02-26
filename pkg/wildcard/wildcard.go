package wildcard

import (
	"errors"
	"regexp"
	"strings"
)

/*
这段代码是一个Golang实现的Wildcard匹配库。它定义了一个Pattern类型，用于表示一个通配符模式，并且提供了两个函数：

CompilePattern(src string) (*Pattern, error)：将通配符字符串编译成Pattern类型，如果编译失败则返回错误信息。

IsMatch(s string) bool：判断给定的字符串是否与Pattern匹配。

在这个实现中，通配符支持的特殊字符包括：

*：匹配任意数量的任意字符。

?：匹配单个任意字符。

+：匹配一个或多个任意字符。

[]：匹配指定字符集合中的任意一个字符。

{}：匹配指定的字符串中的任意一个。

这段代码首先根据通配符字符串生成一个对应的正则表达式，然后使用正则表达式匹配目标字符串。
具体实现使用了replaceMap映射表，将通配符字符串中的特殊字符映射为正则表达式中的字符。
值得注意的是，在处理转义字符时，如果通配符字符串以反斜杠结尾，将会返回错误信息，因为这是不合法的通配符格式。
*/

// Pattern represents a wildcard pattern
type Pattern struct {
	exp *regexp.Regexp
}

var replaceMap = map[byte]string{
	// characters in the wildcard that must be escaped in the regexp
	'+': `\+`,
	')': `\)`,
	'$': `\$`,
	'.': `\.`,
	'{': `\{`,
	'}': `\}`,
	'|': `\|`,
	'*': ".*",
	'?': ".",
}

var errEndWithEscape = "end with escape \\"

// CompilePattern convert wildcard string to Pattern
func CompilePattern(src string) (*Pattern, error) {
	regexSrc := strings.Builder{}
	regexSrc.WriteByte('^')
	for i := 0; i < len(src); i++ {
		ch := src[i]
		if ch == '\\' {
			if i == len(src)-1 {
				return nil, errors.New(errEndWithEscape)
			}
			regexSrc.WriteByte(ch)
			regexSrc.WriteByte(src[i+1])
			i++ // skip escaped character
		} else if ch == '^' {
			if i == 0 {
				regexSrc.WriteString(`\^`)
			} else if i == 1 {
				if src[i-1] == '[' {
					regexSrc.WriteString(`^`) // src is: [^
				} else {
					regexSrc.WriteString(`\^`)
				}
			} else {
				if src[i-1] == '[' && src[i-2] != '\\' {
					regexSrc.WriteString(`^`) // src is: [^, except \[^
				} else {
					regexSrc.WriteString(`\^`)
				}
			}
		} else if escaped, toEscape := replaceMap[ch]; toEscape {
			regexSrc.WriteString(escaped)
		} else {
			regexSrc.WriteByte(ch)
		}
	}
	regexSrc.WriteByte('$')
	re, err := regexp.Compile(regexSrc.String())
	if err != nil {
		return nil, err
	}
	return &Pattern{
		exp: re,
	}, nil
}

// IsMatch returns whether the given string matches pattern
func (p *Pattern) IsMatch(s string) bool {
	return p.exp.Match([]byte(s))
}
