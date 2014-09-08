package scanbuf

import (
	"io"
)

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
	r   Scanbuf 输入源
*/
func New(r io.Reader) *Scanbuf {
	return &Scanbuf{r: r, buf: []byte{}}
}

/**
Scanbuf 实现了 io.Reader 接口.
事实上从 Read 中读取的数据是 Scanbuf 中剩余的数据.
*/
func (s *Scanbuf) Read(p []byte) (n int, err error) {
	if s.err == nil {
		s.update()
	}
	n = copy(s.buf[s.start:s.end], p)
	s.start += n
	err = s.err
	return
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
	s.err = nil
	return s
}

/**
Advance 更新缓冲.
参数:
	n 消耗掉的字节数, n >= 0, 初次调用应该是 0.
返回:
	更新后的缓冲数据, 调用者处理此数据所消耗掉的字节数就是 n.
*/
func (s *Scanbuf) Advance(n int) ([]byte, error) {
	s.start += n
	s.update()
	return s.buf[s.start:s.end], s.err
}

func (s *Scanbuf) update() {
	const minBuffer = 512

	if s.err != nil {
		return
	}

	if s.r == nil {
		s.err = io.EOF
		return
	}

	size := len(s.buf)

	// 缓冲不够需要移动或扩充
	if size-s.end < minBuffer {
		if size-s.end+s.start >= minBuffer {

			copy(s.buf, s.buf[s.start:s.end])
		} else {

			newBuf := make([]byte, size*2+minBuffer)
			copy(newBuf, s.buf[s.start:s.end])
			s.buf = newBuf
		}
		s.end -= s.start
		s.start = 0
	}

	var n int
	// 更新数据, 出于小心谨慎
	for loop := 10; ; {
		n, s.err = s.r.Read(s.buf[s.end:])
		loop--
		if n > 0 || s.err != nil {
			break
		}
		if loop == 0 {
			s.err = io.ErrNoProgress
			break
		}
	}
	s.end += n
}

/**
WriteTo 向 w 写入数据. Scanbuf 符合 io.WriterTo 接口, 但行为有所差异.
参数:
	w io.Writer, Scanbuf 通过 w.Write 写入数据.
	Write 的参数: p []byte
		p Scanbuf 向 w 写入数据, nil 表示 EOF.
	Write 的返回值: n int64, err error
		n   指示 Scanbuf 向前推进的字节数.
		err 指示发生的错误, 如果非 nil, Scanbuf 将返回此错误.
	WriteTo 持续更新缓冲并向 w 写入数据, 直到 EOF 或者 Write 返回错误.
返回:
	n   WriteTo 向 w 写入的字节数.
	err 发生的错误信息, EOF 用 nil 替代.
Recorder 被调用后, 如果 data 为 nil, 或者有错误发生扫描结束.
*/
func (s *Scanbuf) WriteTo(w io.Writer) (int64, error) {
	var (
		n   int64
		c   int
		err error
	)

	for {
		s.update()
		if s.end-s.start == 0 {
			break
		}
		c, err = w.Write(s.buf[s.start:s.end])
		if err != nil {
			s.err = err
			break
		}
		s.start += c
		n += int64(c)
	}

	return n, s.err
}
