package main

import (
	"fmt"

	"github.com/aglyzov/go-ds/critbit/dict"
)

func main() {
	d := dict.NewDict()
	d.Set([]byte("c"), 1)
	//d.Set([]byte("a"),  2)
	d.Set([]byte("a1"), 3)
	d.Set([]byte("a2"), 4)
	d.Set([]byte("a3"), 5)
	d.Set([]byte("a22"), 6)
	d.Set([]byte("bb"), 7)

	d.DebugDump()

	var a, b *dict.RefPath

	//a,b = d.FindPathRange([]byte(""));		fmt.Printf("R()     -> %v .. %v\n", a, b)
	a, b = d.FindPathRange([]byte("a"))
	fmt.Printf("R(a)    -> %v .. %v\n", a, b)

	if a != nil {
		cur, end := a.GetLeaf(), b.GetLeaf()
		for {
			fmt.Printf("%s\n", cur.Key)
			if cur == end {
				break
			}
			cur = a.TrackNext()
		}
	}

	println("------")

	visitor := func(item dict.Item) bool {
		fmt.Printf("%s\n", item.Key)
		return true
	}
	d.Iter([]byte("a"), visitor)

	//fmt.Printf("GE()    -> %v\n", d.FindPathGE([]byte("")))
	//fmt.Printf("GE(000) -> %v\n", d.FindPathGE([]byte("000")))
	//fmt.Printf("GE(e)   -> %v\n", d.FindPathGE([]byte("e")))
	//fmt.Printf("GE(c)   -> %v\n", d.FindPathGE([]byte("c")))
	//fmt.Printf("GE(b)   -> %v\n", d.FindPathGE([]byte("b")))
	//fmt.Printf("GE(bb)  -> %v\n", d.FindPathGE([]byte("bb")))
	//fmt.Printf("GE(b1)  -> %v\n", d.FindPathGE([]byte("b1")))
	//fmt.Printf("GE(bbb) -> %v\n", d.FindPathGE([]byte("bbb")))

	//fmt.Printf("LE()    -> %v\n", d.FindPathLE([]byte("")))
	//fmt.Printf("LE(000) -> %v\n", d.FindPathLE([]byte("000")))
	//fmt.Printf("LE(e)   -> %v\n", d.FindPathLE([]byte("e")))
	//fmt.Printf("LE(c)   -> %v\n", d.FindPathLE([]byte("c")))
	//fmt.Printf("LE(b)   -> %v\n", d.FindPathLE([]byte("b")))
	//fmt.Printf("LE(bb)  -> %v\n", d.FindPathLE([]byte("bb")))
	//fmt.Printf("LE(b1)  -> %v\n", d.FindPathLE([]byte("b1")))
	//fmt.Printf("LE(bbb) -> %v\n", d.FindPathLE([]byte("bbb")))
}
