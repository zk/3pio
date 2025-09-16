package main

import (
    "testing"
    "time"
)

func TestExample(t *testing.T) {
    time.Sleep(100 * time.Millisecond)
    if 1+1 != 2 {
        t.Fatal("math is broken")
    }
}

func TestAnother(t *testing.T) {
    time.Sleep(100 * time.Millisecond)
    if 2+2 != 4 {
        t.Fatal("more math is broken")
    }
}
