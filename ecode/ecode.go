/*
 * @Date: 2022-11-21 00:49:03
 * @LastEditors: lipengfei
 * @LastEditTime: 2022-11-21 01:52:37
 * @FilePath: \vlgo\ecode\ecode.go
 * @Description:
 */
package ecode

import "fmt"

//go:generate cd ecode && parse_code.sh
type VEI interface {
	error
	fmt.Stringer
}

type verr struct {
	c uint32
	s string
}

func newVError(s string, code uint32) *verr {
	return &verr{c: code, s: s}
}

func (e *verr) Error() string {
	if e != nil {
		return e.s
	}
	return "<nil>"
}

func (e *verr) String() string {
	return e.Error()
}

func CustomThirdPluginErr(e error) VEI {
	if e == nil {
		return nil
	}
	return newVError(e.Error(), 100001)
}
