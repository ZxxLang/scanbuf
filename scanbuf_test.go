package scanbuf

import (
	"bytes"
	"io"
	"os"
	"testing"
)

func TestAdvance(T *testing.T) {
	ExampleScanbuf_Advance(T)
}

func TestWriteTo(T *testing.T) {
	ExampleScanbuf_WriteTo(T)
}

func ExampleScanbuf_Advance(T *testing.T) {

	var (
		n, off int
		s      []byte
		err    error
	)

	f, err := os.Open("LICENSE")
	if err != nil {
		T.Fatal(err)
	}

	defer f.Close()
	buf := New(nil).Source(f)

	lines := 0
	for {
		s, err = buf.Advance(n)

		if len(s) == 0 {
			break // EOF 或者发生错误
		}

		// 尽可能的消耗缓冲, 减少缓冲更新次数
		n = 0
		for {
			off = bytes.IndexByte(s, '\n')
			if off == -1 {
				break
			}
			lines++

			n += off + 1
			s = s[off+1:]
		}

		if err != nil {
			break
		}
	}

	if err != io.EOF {
		T.Fatal("want io.EOF but got", err)
	}

	if lines != 23 {
		T.Fatalf("want lines 23 but got %d", lines)
	}
}

func ExampleScanbuf_WriteTo(T *testing.T) {

	f, err := os.Open("LICENSE")
	if err != nil {
		T.Fatal(err)
	}

	defer f.Close()
	buf := New(nil).Source(f)

	lines := 0
	n, err := buf.WriteTo(WriterFunc(func(s []byte) (int, error) {
		if len(s) == 0 {
			T.Fatal("unexpected got len(s) == 0")
		}

		// 尽可能的消耗缓冲, 减少缓冲更新次数
		n := 0
		for {

			off := bytes.IndexByte(s, '\n')
			if off == -1 {
				break
			}

			lines++
			n += off + 1
			s = s[off+1:]
		}

		return n, nil
	}))

	if err != io.EOF {
		T.Fatal("want io.EOF but got", err)
	}

	if lines != 23 {
		T.Fatalf("want lines 23 but got %d", lines)
	}

	// Unix Line Endings (LF)
	if n != 1299 {
		T.Fatalf("want total bytes 1299 but got %d", n)
	}
}
