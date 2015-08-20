package cgoonly

/*
int add(int x, int y) {
        return x+y;
};
*/
import "C"

func Add(x, y int) int { 
	return int(C.add(_Ctype_int(x), _Ctype_int(y)))
}
