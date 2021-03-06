// Copyright 2020 Marko Kungla. All rights reserved.
// Use of this source code is governed by a license
// that can be found in the LICENSE file.

package bexp

import (
	"io/ioutil"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

const (
	braceexpansion = "braceexpansion"
)

type testResource struct {
	Pattern string
	Res0    string
	Res1    string
	Res2    string
}

var testData = []testResource{
	// // PATHS
	{"data/{P1/{10..19},P2/{20..29},P3/{30..39}}", "data/P1/10", "data/P1/11", "data/P1/12"},
	{"data/{-1..-3}", "data/-1", "data/-2", "data/-3"},
	{"data/{-3..-1}", "data/-3", "data/-2", "data/-1"},
	{"data/{-1..1}", "data/-1", "data/0", "data/1"},
	{"/usr/{ucb/{ex,edit},lib/{ex?.?*,how_ex}}", "/usr/ucb/ex", "/usr/ucb/edit", "/usr/lib/ex?.?*"},
	{"/usr/local/src/bash/{old,new,dist,bugs}", "/usr/local/src/bash/old", "/usr/local/src/bash/new", "/usr/local/src/bash/dist"},
	{"{abc{def}}ghi", "{abc{def}}ghi", "", ""},
	{"{,}", "", "", ""},
	{"{,,}", "", "", ""},
	{"{,,,}", "", "", ""},
	{"\\\\", "\\", "", ""},
	{"-", "-", "", ""},
	{braceexpansion, braceexpansion, "", ""},
	{"{a,b}{1,2}", "a1", "a2", "b1"},
	{"a{d,c,b}e", "ade", "ace", "abe"},
	{"x{{a,b,c}}y", "x{a}y", "x{b}y", "x{c}y"},
	{"a", "a", "", ""},
	{"a,b", "a,b", "", ""},
	{"a,,b", "a,,b", "", ""},
	{"a,", "a,", "", ""},
	{",a", ",a", "", ""},
	{",", ",", "", ""},
	{",,", ",,", "", ""},
	{"abc,def", "abc,def", "", ""},
	{"{abc,def}", "abc", "def", ""},
	{"{}", "{}", "", ""},
	{"a{}", "a{}", "", ""},
	{"{a{}}", "{a{}}", "", ""},
	{"a{,}", "a", "a", ""},
	// working
	{"a{b,c}", "ab", "ac", ""},
	{"a{,b,c}", "a", "ab", "ac"},
	{"a{b,c,}", "ab", "ac", "a"},
	{"a{b,d{e,f}}", "ab", "ade", "adf"},
	{"a{b,{c,d}}", "ab", "ac", "ad"},
	{"{a,b}{1,2}", "a1", "a2", "b1"},
	{"{a,b}x{1,2}", "ax1", "ax2", "bx1"},
	{"{abc}", "{abc}", "", ""},
	{"{abc}def", "{abc}def", "", ""},

	{"{a,{{{b}}}}", "a", "{{{b}}}", ""},
	{"{a{1,2}b}", "{a1b}", "{a2b}", ""},
	{"{0,1,2}", "0", "1", "2"},
	{"{-0,-1,-2}", "-0", "-1", "-2"},
}

func TestParse(t *testing.T) {
	for _, test := range testData {
		t.Run(test.Pattern, func(t *testing.T) {
			v := Parse(test.Pattern)
			if len(v) > 0 {
				assert.Equal(t, test.Res0, v[0])
			}
			if len(v) > 1 {
				assert.Equal(t, test.Res1, v[1])
			}
			if len(v) > 2 {
				assert.Equal(t, test.Res2, v[2])
			}
		})
	}
}

func TestParseValid(t *testing.T) {
	empty, err := ParseValid("")
	assert.ErrorIs(t, err, ErrEmptyResult)
	assert.Equal(t, "", empty[0])

	single, err := ParseValid("a")
	assert.NoError(t, single.Err())
	assert.ErrorIs(t, err, ErrUnchangedBraceExpansion)
	assert.Equal(t, "a", single[0])

	for _, test := range testData {
		t.Run(test.Pattern, func(t *testing.T) {
			v, _ := ParseValid(test.Pattern)
			if len(v) > 0 {
				assert.Equal(t, test.Res0, v[0])
			}
			if len(v) > 1 {
				assert.Equal(t, test.Res1, v[1])
			}
			if len(v) > 2 {
				assert.Equal(t, test.Res2, v[2])
			}
		})
	}
}

func TestBashExpansion(t *testing.T) {
	dat, _ := ioutil.ReadFile("testdata/bash-results.txt")
	cases := strings.Split(string(dat), "><><><><")
	for _, v := range cases {
		lines := strings.Split(v, "\r\n")
		cs := lines[0]
		lines = lines[1:]
		expected := BraceExpansion{""}
		for _, l := range lines {
			if len(l) != 0 {
				expected = append(expected, l[1:len(l)-1])
			}
		}
		result := Parse(cs)
		assert.Equal(t, expected, result)
	}
}

func TestIgnoreDollar(t *testing.T) {
	assert.Equal(t, BraceExpansion{"${1..3}"}, Parse("${1..3}"))
	assert.Equal(t, BraceExpansion{"${a,b}${c,d}"}, Parse("${a,b}${c,d}"))
	assert.Equal(t, BraceExpansion{"x${a,b}x${c,d}x"}, Parse("x${a,b}x${c,d}x"))
}

func TestEmptyOption(t *testing.T) {
	assert.Equal(t, BraceExpansion{"-v", "-v", "-v", "-v", "-v"}, Parse("-v{,,,,}"))
}

func TestNegativeIncrement(t *testing.T) {
	assert.Equal(t, BraceExpansion{"3", "2", "1"}, Parse("{3..1}"))
	assert.Equal(t, BraceExpansion{"10", "9", "8"}, Parse("{10..8}"))
	assert.Equal(t, BraceExpansion{"10", "09", "08"}, Parse("{10..08}"))
	assert.Equal(t, BraceExpansion{"c", "b", "a"}, Parse("{c..a}"))
	assert.Equal(t, BraceExpansion{"4", "2", "0"}, Parse("{4..0..2}"))
	assert.Equal(t, BraceExpansion{"4", "2", "0"}, Parse("{4..0..-2}"))
	assert.Equal(t, BraceExpansion{"e", "c", "a"}, Parse("{e..a..2}"))
}

func TestNested(t *testing.T) {
	assert.Equal(t, BraceExpansion{"a", "b1", "b2", "b3", "c"}, Parse("{a,b{1..3},c}"))
	assert.Equal(t, BraceExpansion(strings.Split("ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz", "")), Parse("{{A..Z},{a..z}}"))
	assert.Equal(t, BraceExpansion{"ppp", "pppconfig", "pppoe", "pppoeconf"}, Parse("ppp{,config,oe{,conf}}"))
}

func TestOrder(t *testing.T) {
	assert.Equal(t, BraceExpansion{"ade", "ace", "abe"}, Parse("a{d,c,b}e"))
}

func TestPad(t *testing.T) {
	assert.Equal(t, BraceExpansion{"9", "10", "11"}, Parse("{9..11}"))
	assert.Equal(t, BraceExpansion{"09", "10", "11"}, Parse("{09..11}"))
}

func TestSameType(t *testing.T) {
	assert.Equal(t, BraceExpansion{"{a..9}"}, Parse("{a..9}"))
}

func TestSequence(t *testing.T) {
	//Numeric
	assert.Equal(t, BraceExpansion{"a1b2c", "a1b3c", "a2b2c", "a2b3c"}, Parse("a{1..2}b{2..3}c"))
	assert.Equal(t, BraceExpansion{"12", "13", "22", "23"}, Parse("{1..2}{2..3}"))
	//Numeric with step count
	assert.Equal(t, BraceExpansion{"0", "2", "4", "6", "8"}, Parse("{0..8..2}"))
	assert.Equal(t, BraceExpansion{"1", "3", "5", "7"}, Parse("{1..8..2}"))
	//Numeric with negative
	assert.Equal(t, BraceExpansion{"3", "2", "1", "0", "-1", "-2"}, Parse("{3..-2}"))

	//Alphabetic
	assert.Equal(t, BraceExpansion{"a", "c", "e", "g", "i", "k"}, Parse("{a..k..2}"))
	assert.Equal(t, BraceExpansion{"b", "d", "f", "h", "j"}, Parse("{b..k..2}"))
}

func TestString(t *testing.T) {
	r := Parse("/{dir1,dir2}")
	assert.Equal(t, "/dir1 /dir2", r.String())
}

func TestResult(t *testing.T) {
	r := Parse("/{dir1,dir2}")
	assert.Equal(t, []string{"/dir1", "/dir2"}, r.Result())
}

func TestErr(t *testing.T) {
	empty := Parse("")
	assert.ErrorIs(t, empty.Err(), ErrEmptyResult)

	single := Parse("a")
	assert.NoError(t, single.Err())
	assert.Equal(t, "a", single[0])
}
