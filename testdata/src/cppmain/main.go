package main

/*
typedef int (*intFunc) ();

int bridge_int_func(intFunc f);

int fortytwo();
*/
import "C"
import "fmt"

func main() {
	f := C.intFunc(C.fortytwo)
	fmt.Println(int(C.bridge_int_func(f)))
	// Output: 42
}
