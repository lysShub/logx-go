package logx

import (
	"fmt"
	"os"
	"testing"

	"github.com/lysShub/logx-go/internal"
)

func Test_ErrStack(t *testing.T) {
	// {
	// 	st := ErrStack(nil)
	// 	if st != nil {
	// 		t.Fatal("rquire nil")
	// 	}
	// }
	// {
	// 	st := ErrStack(errors.New("xxx"))
	// 	if len(st) == 0 {
	// 		t.Fatal("rquire not nil")
	// 	}
	// }
	// {
	// 	st := ErrStack(errors.WithMessage(errors.New("xxx"), "xxx"))
	// 	if len(st) == 0 {
	// 		t.Fatal("rquire not nil")
	// 	}
	// }
}

func TestXxx(t *testing.T) {

	fh, err := os.Open("go.mod")
	if err != nil {
		panic(err.Error())
	}
	raw, err := fh.SyscallConn()
	if err != nil {
		panic(err.Error())
	}

	var b = make([]byte, 64)
	var e internal.Error

	err = raw.Write(func(fd uintptr) (done bool) {

		_, e = fh.Read(b)

		return true
	})
	if err != nil {
		panic(err.Error())
	}
	if e != nil {
		panic(e.Error())
	}

	fmt.Println(b)

}
