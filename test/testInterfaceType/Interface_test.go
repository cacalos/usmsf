package main

import (
	"fmt"
	"testing"
)

type InterfaceType struct {
	aa string
}

type inter interface {
	testFunc() string
}

func (r *InterfaceType) testFunc() string {
	return "test"
}

func TestInterfaceType(t *testing.T) {
	test := &InterfaceType{}

	CheckType(test)
}

func CheckType(v interface{}) {
	switch a := v.(type) {
	case inter:
		fmt.Println(a.testFunc())
	default:
		fmt.Println("why default")
	}
}
