package scanbuf

import (
	"io"
)

// WriterFunc 可以包裹函数使其符合 io.Writer 接口.
type WriterFunc func([]byte) (int, error)

func (w WriterFunc) Write(p []byte) (int, error) {
	return w(p)
}

/**
Scanbuf 是一个前进式读缓冲, 自动扩展缓冲大小.
*/
type Scanbuf struct {
	r     io.Reader
	buf   []byte
	start int
	end   int
	err   error
}

/**
New 新建一个 *Scanbuf.
参数:
	p   Scanbuf 初始化缓冲
*/
func New(p []byte) *Scanbuf {
	s := &Scanbuf{buf: []byte{}}
	if len(p) == 0 {
		s.err = io.EOF
	} else {
		s.buf = p
		s.end = len(p)
	}
	return s
}

/**
Advance 更新并返回缓冲数据.
参数:
	n 消耗掉(向前推进)的字节数, 初次调用应该是 0.
	advanc 会修正 n 值, 使 0 <= n <= len(buf) 范围内.
返回:
	更新后的缓冲数据, 调用者处理此数据所消耗掉的字节数就是 n.
	有可能同时返回数据和 io.EOF.
*/
func (s *Scanbuf) Advance(n int) ([]byte, error) {
	s.advance(n)
	if s.err != nil && s.err != io.EOF {
		return nil, s.err
	}
	return s.buf[s.start:s.end], s.err
}

const incBuffer = 512

func (s *Scanbuf) advance(n int) {

	// 先计算前进量, 保证推进
	s.start += n

	if s.start < 0 {
		s.start = 0
	} else if s.start > s.end {
		s.start = s.end
	}

	if s.err != nil {
		return
	}

	if s.r == nil {
		s.err = io.EOF
		return
	}

	size := len(s.buf)
	// 缓冲不够需要移动或扩充
	if size-s.end < incBuffer {
		if size-s.end+s.start >= incBuffer {

			copy(s.buf, s.buf[s.start:s.end])
		} else {

			newBuf := make([]byte, (size/incBuffer+1)*incBuffer)
			copy(newBuf, s.buf[s.start:s.end])
			s.buf = newBuf
		}
		s.end -= s.start
		s.start = 0
	}

	n = 0
	// 更新数据, 出于小心谨慎, 无错误读取 0 字节时, 多次尝试.
	for loop := 10; ; loop-- {
		n, s.err = s.r.Read(s.buf[s.end:])

		if s.err != nil {
			s.r = nil
			break
		}

		if n > 0 {
			break
		}

		if loop == 0 {
			s.err = io.ErrNoProgress
			s.r = nil
			break
		}
	}

	s.end += n
}

/**
Reset 丢弃缓冲中的数据, 清除已经发生的错误.
*/
func (s *Scanbuf) Reset() *Scanbuf {
	s.start = 0
	s.end = 0
	s.err = nil
	return s
}

/**
Source 设定 io.Reader 输入源, 保留缓冲中的数据, 清除已经发生的错误.
*/
func (s *Scanbuf) Source(r io.Reader) *Scanbuf {
	s.r = r
	if s.r == nil && len(s.buf) == 0 {
		s.err = io.EOF
	} else {
		s.err = nil
	}
	return s
}

/**
WriteTo 更新缓冲并向 w 写入数据, 直到缓冲被消耗完或者有错误发生.
参数:
	w io.Writer, Scanbuf 通过 w.Write 写入数据.
	Write 的参数: p []byte
		p Scanbuf 向 w 写入数据, len(p) 不会为 0.
	Write 的返回值: n int, err error
		n   指示 Scanbuf 向前推进的字节数.
		err 指示发生的错误, 如果非 nil, Scanbuf 将返回此错误.

返回:
	n   WriteTo 向 w 写入的字节数.
	err 发生的错误信息, EOF 用 nil 替代.
*/
func (s *Scanbuf) WriteTo(w io.Writer) (int64, error) {
	var (
		n   int64
		c   int
		err error
	)

	for {
		s.advance(c)

		// 没数据了
		if s.end == s.start {
			break
		}

		c, err = w.Write(s.buf[s.start:s.end])
		n += int64(c)

		if err != nil {
			return n, err
		}

		if s.err != nil && s.err != io.EOF {
			break
		}
	}
	return n, s.err
}
