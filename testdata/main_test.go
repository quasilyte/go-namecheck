package main

import "testing"

func TestBadParam1(test *testing.T) {}
func TestBadParam2(tst *testing.T)  {}

func BenchmarkBadParam(bench *testing.B) {}
