// /home/krylon/go/src/github.com/blicero/guang/frontend/helpers_tmpl.go
// -*- mode: go; coding: utf-8; -*-
// Created on 31. 10. 2022 by Benjamin Walkenhorst
// (c) 2022 Benjamin Walkenhorst
// Time-stamp: <2022-10-31 18:54:41 krylon>

package frontend

import (
	"errors"
	"time"
)

////////////////////////////////////
// Functions for use in templates //
////////////////////////////////////

type generator struct {
	values []string
	index  int
	f      func(s []string, i int) string
}

func (seq *generator) Next() string {
	s := seq.f(seq.values, seq.index)
	seq.index++
	return s
} // func (seq *generator) Next() string

func sequenceGen(values []string, i int) string {
	if i >= len(values) {
		return values[len(values)-1]
	}

	return values[i]
} // func sequenceGen(values []string, i int) string

func cycleGen(values []string, i int) string {
	return values[i%len(values)]
} // func cycleGen(values []string, i int) string

func sequenceFunc(values ...string) (*generator, error) {
	if len(values) == 0 {
		return nil, errors.New("Sequence must have at least one element")
	}

	return &generator{
		values: values,
		index:  0,
		f:      sequenceGen,
	}, nil
} // func sequenceFunc(values ...string) (*generator, error)

func cycleFunc(values ...string) (*generator, error) {
	if len(values) == 0 {
		return nil, errors.New("Cycle must have at least one element")
	}

	return &generator{
		values: values,
		index:  0,
		f:      cycleGen,
	}, nil
} // func cycleFunc(values ...string) (*generator, error)

func nowFunc() string {
	return time.Now().Format("2006-01-02 15:04:05")
} // func nowFunc() string
