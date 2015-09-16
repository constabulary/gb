typedef int (*intFunc) ();

extern "C" int bridge_int_func(intFunc f);

extern "C" int fortytwo();

int
bridge_int_func(intFunc f)
{
	return f();
}

int fortytwo()
{
    return 42;
}
