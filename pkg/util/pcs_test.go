package util

import (
	"context"
	"fmt"
	"testing"
)

func TestMd5(t *testing.T) {
	md5, err := Md5(context.TODO(), []byte{})
	fmt.Printf("%+v\n%+v", md5, err)
}
