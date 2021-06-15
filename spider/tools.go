package spider

import (
	"bytes"
	"io/ioutil"
	"unsafe"

	"golang.org/x/text/encoding/traditionalchinese"
	"golang.org/x/text/transform"
)

//convert BIG5 to UTF-8
func DecodeBig5(s []byte) (string, error) {
	I := bytes.NewReader(s)
	O := transform.NewReader(I, traditionalchinese.Big5.NewDecoder())
	d, e := ioutil.ReadAll(O)
	if e != nil {
		return "", e
	}
	return *(*string)(unsafe.Pointer(&d)), nil
}
