package testhelpers

import (
	"fmt"
	"reflect"
	"testing"
)

type TestRunner[T any, A any] struct {
	focused []func(*testing.T)
	regular []func(*testing.T)
	t       *testing.T
	assert  A
}

func NewTestRunner[T any, A any](t *testing.T, assert A) *TestRunner[T, A] {
	return &TestRunner[T, A]{
		focused: make([]func(*testing.T), 0),
		regular: make([]func(*testing.T), 0),
		t:       t,
		assert:  assert,
	}
}

func GenerateCases[T any, A any](t *testing.T, assert A) (T, T, T, func()) {
	runner := NewTestRunner[T, A](t, assert)

	var zero T
	testType := reflect.TypeOf(zero)

	if testType.Kind() != reflect.Func {
		panic("T must be a function type")
	}

	makeTestFunc := func(isFocused bool, isSkipped bool) T {
		f := reflect.MakeFunc(testType, func(args []reflect.Value) []reflect.Value {
			if len(args) < 1 || args[0].Kind() != reflect.String {
				panic("First argument must be a string (test name)")
			}

			testName := args[0].String()

			if isSkipped {
				fmt.Printf("Skipping Test %s\n", testName)
				return nil
			}

			testFunc := func(t *testing.T) {
				assertArgs := make([]reflect.Value, len(args))
				copy(assertArgs, args)
				assertArgs[0] = reflect.ValueOf(t)
				reflect.ValueOf(assert).Call(assertArgs)
			}

			if isFocused {
				runner.focused = append(runner.focused, testFunc)
			} else {
				runner.regular = append(runner.regular, testFunc)
			}

			return nil
		})

		return f.Interface().(T)
	}

	run := func() {
		tests := runner.regular
		if len(runner.focused) > 0 {
			tests = runner.focused
		}

		for i, testFunc := range tests {
			t.Run(fmt.Sprintf("Test_%d", i+1), func(t *testing.T) {
				testFunc(t)
			})
		}
	}

	return makeTestFunc(false, false), makeTestFunc(true, false), makeTestFunc(false, true), run
}
