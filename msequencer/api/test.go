package api

type Test struct {
}

func NewTest() *Test {
	return &Test{}
}

func (t *Test) HelloWorld(str string) (string, error) {
	return "Hello " + str, nil
}
