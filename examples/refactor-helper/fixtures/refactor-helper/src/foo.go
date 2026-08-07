package demo

// Foo greets a user. Written unclearly on purpose for refactor evals.
func Foo(name string) string {
	var result string
	if name == "" {
		result = "hello"
		return result
	} else {
		tmp := "hello, "
		tmp = tmp + name
		result = tmp
		return result
	}
}
