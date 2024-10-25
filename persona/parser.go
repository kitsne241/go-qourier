package persona

import (
	"fmt"
	"reflect"
	"strconv"
	"strings"
)

// Syntax についてはたとえばこんな感じ。厳密性だいじに

// "%s %d:%d - %d:%d\n"%s"\n```%x\n%s\n```"

// %s は文字列。次の要素との間（あるいは終わりまで）にある文字列がヒットするまで延々と読む
// %d は数字。ただし、一度 %s として得た文字列を数字に変換するだけなのでエラーになる場合がある
// %x は無視する（変数を用意しない）場所。読み方の規則は %s と同じ

// とりあえず %s %d %x の 3 つの要素だけあれば大体の要求は満たせるはず

func (command *Command) parseExecute(ms *Message, optionOrigin string) error {
	syntax := command.Syntax + "\n"
	option := optionOrigin + "\n"
	args := []any{}

	// 与えられた文字列で最初に %s %d %x のいずれかが登場する地点を返す関数
	// なければ syntax の末尾の位置を返す

	receiving := byte('x')

	// syntax = "%s %d:%d"
	// option = "Sunday 15:00"

	for {
		posSp := nextSpecifier(syntax)
		divider := syntax[:posSp]
		posDv := strings.Index(option, divider)
		if posDv == -1 {
			return fmt.Errorf("too few arguments")
		}
		arg := option[:posDv]

		// syntax 内の次の指定子の場所を得て、そこまでの部分に一致する位置を option でも探して posDv とする
		// option 内の posDv までの部分が次の arg である。見つからなければエラーを返す

		switch receiving {
		case 's':
			args = append(args, arg)
		case 'd':
			argNum, err := strconv.Atoi(arg)
			if err != nil {
				return fmt.Errorf("%%d for non-numeric arguments: %w", err)
			}
			args = append(args, argNum)
		}

		if posSp == len(syntax) {
			break
		}

		receiving = syntax[(posSp + 1)]          // 次の指定子を取得
		syntax = syntax[(posSp + 2):]            // syntax の頭を切り落とす
		option = option[(posDv + len(divider)):] // option の頭を切り落とす
	}

	return command.action(ms, args...)
}

func varadic(command *Command) (func(*Message, ...any) error, error) {
	// 関数を受け取り、多変数引数関数を返す
	fnValue := reflect.ValueOf(command.Action)
	fnType := fnValue.Type()

	// func が関数であるか確認
	if fnType.Kind() != reflect.Func {
		return nil, fmt.Errorf("'%s' is not a function", command.name)
	}

	// func が error 型の返り値をただ 1 つ持つことを確認
	if fnType.NumOut() > 1 {
		return nil, fmt.Errorf("'%s' must have only one return", command.name)
	}
	if fnType.NumOut() < 1 || fnType.Out(0) != reflect.TypeOf((*error)(nil)).Elem() {
		// ショートサーキット評価によって、fnType.NumOut() < 1 が判明すると || 以降の条件式は読まれない
		return nil, fmt.Errorf("'%s' must have an error return", command.name)
	}

	// func が *Message 型の引数を最初に持つことを確認
	if fnType.NumIn() < 1 || fnType.In(0) != reflect.TypeOf((*Message)(nil)) {
		return nil, fmt.Errorf("first argument of '%s' must be *Message", command.name)
	}

	// command.Syntax と照合し、第二引数以降の型の合致を確認

	syntax := command.Syntax
	receiving := byte('x')

	i := 1
	for {
		posSp := nextSpecifier(syntax)
		switch receiving {
		case 's':
			if fnType.NumIn() == i {
				return nil, fmt.Errorf("'%s' does not have enough arguments", command.name)
			}
			if fnType.In(i) != reflect.TypeOf("") {
				return nil, fmt.Errorf("argument %d of '%s' must be string", i+1, command.name)
			}
			i++
		case 'd':
			if fnType.NumIn() == i {
				return nil, fmt.Errorf("'%s' does not have enough arguments", command.name)
			}
			if fnType.In(i) != reflect.TypeOf(0) {
				return nil, fmt.Errorf("argument %d of '%s' must be int", i+1, command.name)
			}
			i++
		}

		if posSp == len(syntax) {
			break
		}

		receiving = syntax[(posSp + 1)] // 次の指定子を取得
		syntax = syntax[(posSp + 2):]   // syntax の頭を切り落とす
	}

	if fnType.NumIn() != i {
		return nil, fmt.Errorf("'%s' has too many arguments", command.name)
	}

	return func(ms *Message, args ...any) error {
		if len(args) != fnType.NumIn()-1 {
			return fmt.Errorf("incorrect number of arguments provided")
		}

		callArgs := make([]reflect.Value, fnType.NumIn())
		callArgs[0] = reflect.ValueOf(ms)
		for i, arg := range args {
			callArgs[i+1] = reflect.ValueOf(arg)
		}

		result := fnValue.Call(callArgs)

		// エラーならばエラーを、nil ならば nil を返す
		if result[0].IsNil() {
			return nil
		}
		return result[0].Interface().(error)
	}, nil
}

func nextSpecifier(syntax string) int {
	pos := strings.Index(syntax, "%s")
	if strings.Contains(syntax, "%d") && ((strings.Index(syntax, "%d") < pos) || (pos == -1)) {
		pos = strings.Index(syntax, "%d")
	}
	if (strings.Contains(syntax, "%x")) && ((strings.Index(syntax, "%x") < pos) || (pos == -1)) {
		pos = strings.Index(syntax, "%x")
	}
	if pos == -1 {
		return len(syntax)
	}
	return pos
}
