package main

import (
	"fmt"
)

type Bar struct {
	total   float64
	current float64
	width   int
	head    string
	empty   string
	fill    string
	left    string
	right   string
}

func New(total float64) *Bar {
	return &Bar{
		total:   total,
		current: 0,
		width:   70,
		head:    ">",
		empty:   "-",
		fill:    "=",
		left:    "[",
		right:   "]",
	}
}

func (b *Bar) Increment() (finished bool) {
	b.current++
	if b.current >= b.total {
		return true
	}

	return false
}

func (b *Bar) GetString() string {
	step := float64(b.width) / b.total
	place := int(b.current * step)

	out := b.left
	for n := 0; n < place; n++ {
		out += b.fill
	}
	out += b.head

	for n := 0; n < b.width-place; n++ {
		out += b.empty
	}
	out += b.right

	return out
}

func (b *Bar) OutputDone() {
	out := b.left
	for n := 0; n < b.width; n++ {
		out += b.fill
	}
	out += b.head
	out += b.right
	out += "\n"

	b.Flush()
	fmt.Print(out)
}

func (b *Bar) Output() {
	b.Flush()
	fmt.Print(b.GetString())
}

func (b *Bar) Flush() {
	fmt.Print("\r\033[K")
}
