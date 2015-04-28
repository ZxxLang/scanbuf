package scanbuf

import (
	"bytes"
	"io"
	"io/ioutil"
	"os"
	"testing"
)

func openfile() (f *os.File, total int64, err error) {
	var fi os.FileInfo
	f, err = os.Open("LICENSE")

	if err == nil {
		fi, err = f.Stat()
		if err != nil {
			f.Close()
			return
		}
		total = fi.Size()
	}
	return
}

func TestAdvance(T *testing.T) {
	f, total, err := openfile()
	if err != nil {
		T.Fatal(err)
	}
	defer f.Close()
	ExampleScanbuf_Advance(T, New(nil).Source(f), total)
}

func TestWriteTo(T *testing.T) {
	f, total, err := openfile()
	if err != nil {
		T.Fatal(err)
	}
	defer f.Close()

	ExampleScanbuf_WriteTo(T, New(nil).Source(f), total)
}

func TestBufAdvance(T *testing.T) {
	f, total, err := openfile()
	if err != nil {
		T.Fatal(err)
	}
	defer f.Close()

	b, err := ioutil.ReadAll(f)
	if err != nil {
		T.Fatal(err)
	}
	if int64(len(b)) != total {
		T.Fatalf("ioutil.ReadAll returns %d bytes but want %d", len(b), total)
	}
	ExampleScanbuf_Advance(T, New(b), total)
}

func TestBufWriteTo(T *testing.T) {
	f, total, err := openfile()
	if err != nil {
		T.Fatal(err)
	}
	defer f.Close()

	b, err := ioutil.ReadAll(f)
	if err != nil {
		T.Fatal(err)
	}
	if int64(len(b)) != total {
		T.Fatalf("ioutil.ReadAll returns %d bytes but want %d", len(b), total)
	}
	ExampleScanbuf_WriteTo(T, New(b), total)
}

func ExampleScanbuf_Advance(T *testing.T, buf *Scanbuf, total int64) {

	var (
		n, off int
		s      []byte
		err    error
		stop   bool
	)
	lines := 0
	for !stop {

		// 尽可能的消耗缓冲, 减少缓冲更新次数
		s, err = buf.Advance(off)

		if len(s) == 0 {
			break // EOF 或者发生错误
		}

		off = bytes.IndexByte(s, '\n')

		if off == -1 {
			if err != io.EOF {
				off = 0
				continue
			}
			off = len(s)
			stop = true
		} else {
			off++
		}
		lines++
		n += off
	}

	if err != io.EOF {
		T.Fatal("want io.EOF but got", err)
	}

	if lines != 23 {
		T.Fatalf("want lines 23 but got %d", lines)
	}
	// Unix Line Endings (LF)
	if int64(n) != total {
		T.Fatalf("want total bytes %d but got %d", total, n)
	}
}

func ExampleScanbuf_WriteTo(T *testing.T, buf *Scanbuf, total int64) {
	var n int
	lines := 0
	c, err := buf.WriteTo(WriterFunc(func(s []byte) (int, error) {
		if len(s) == 0 {
			T.Fatal("unexpected got len(s) == 0")
		}

		off := bytes.IndexByte(s, '\n')
		if off == -1 {
			if !buf.IsEOF() {
				return 0, nil
			}
			off = len(s)
		} else {
			off++
		}

		lines++
		n += off
		return off, nil
	}))

	if err != io.EOF {
		T.Fatal("want io.EOF but got", err)
	}

	if lines != 23 {
		T.Fatalf("want lines 23 but got %d", lines)
	}
	// Unix Line Endings (LF)
	if int64(n) != total {
		T.Fatalf("want total bytes %d but got %d", total, n)
	}
	if c != total {
		T.Fatalf("want total bytes %d but got %d", total, c)
	}
}
