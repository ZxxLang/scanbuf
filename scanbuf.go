package scanbuf

import (
	"errors"
	"io"
)

var ErrOutOfBuffer = errors.New("out of buffer limited")

/**
最小缓冲尺寸也是缓冲递增量.
现实中实际的最小缓冲尺寸会达到 MinBuffer * 2.
*/
const MinBuffer = 512

/**
最大缓冲尺寸限制, 如果已经达到此尺寸, 并且仍然需要补充缓冲数据,
将产生 ErrOutOfBuffer 错误.
*/
const MaxBuffer = int(^uint(0)>>2) + 1

// WriterFunc 可以包裹函数使其符合 io.Writer 接口.
type WriterFunc func([]byte) (int, error)

func (w WriterFunc) Write(p []byte) (int, error) {
	return w(p)
}

/**
Scanbuf 是一个前进式读缓冲, 自动扩展缓冲大小.
Scanbuf 内部维护两个游标 start, end 表示有效数据范围.
当调用者需要补充缓冲数据时, Scanbuf 根据缓冲使用状态确定补充方式:
扩展缓冲或者移动数据到头部然后补充缓冲数据.
如果无法补充缓冲, 调用者得到尺寸为 0 的数据和相关错误信息.
*/
type Scanbuf struct {
	r     io.Reader
	buf   []byte
	start int
	end   int
	max   int
	err   error
}

/**
New 新建一个 *Scanbuf.
参数:
	p   Scanbuf 初始化缓冲.
*/
func New(p []byte) *Scanbuf {
	s := &Scanbuf{buf: p, max: MaxBuffer}
	if len(p) == 0 {
		s.err = io.EOF
	} else {
		//s.buf = p
		s.end = len(p)
	}
	return s
}

/**
Advance 更新并返回缓冲数据.
参数:
	n 消耗掉(向前推进)的字节数, 初次调用应该是 0.
	内部以 s.start += n 进行计算, 且自动调整 s.start >= 0.
	只有 n 为 0 时, Advance 才会补充缓冲数据.
返回:
	更新后的缓冲数据, 调用者处理此数据所消耗掉的字节数就是 n.
	有可能同时返回数据和 io.EOF.
*/
func (s *Scanbuf) Advance(n int) ([]byte, error) {
	s.advance(n)
	return s.buf[s.start:s.end], s.err
}

func (s *Scanbuf) advance(n int) {
	size := s.end - s.start
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

	if n != 0 && s.start != s.end {
		return
	}

	if s.r == nil {
		s.err = io.EOF
		return
	}

	size = len(s.buf)
	// 缓冲不够需要移动或扩充
	if size-s.end < MinBuffer {
		if size-s.end+s.start >= MinBuffer {
			if s.start != 0 {
				copy(s.buf, s.buf[s.start:s.end])
			}
		} else {
			size = (size/MinBuffer + 1) * MinBuffer
			if size > s.max {
				if s.start == 0 && s.end == size {
					s.end = 0
					s.err = ErrOutOfBuffer
					return
				}
				if s.start != 0 {
					copy(s.buf, s.buf[s.start:s.end])
				}
			} else {
				newBuf := make([]byte, size)
				copy(newBuf, s.buf[s.start:s.end])
				s.buf = newBuf
			}
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
			if s.err != io.EOF {
				s.end = s.start
				return
			}
			break
		}

		if n > 0 {
			break
		}

		if loop == 0 {
			s.err = io.ErrNoProgress
			s.r = nil
			s.end = s.start
			return
		}
	}
	s.end += n
}

/**
Limit 限定缓冲最大尺寸.
参数:
	size 期望设置的限定尺寸
返回:
	实际限定的尺寸
Limit 会自动调整 size 为 MinBuffer 的整数倍.
*/
func (s *Scanbuf) Limit(size int) int {
	size = size / MinBuffer * MinBuffer
	if size > 0 && size <= MaxBuffer {
		s.max = size
	}
	return s.max
}

/**
IsEOF 返回是否遇到 io.EOF.
*/
func (s *Scanbuf) IsEOF() bool {
	return s.err == io.EOF
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
Size 返回当前缓冲尺寸.
通常补充缓冲时, 缓冲中有待处理数据, 使尺寸最小会达到 MinBuffer * 2.
*/
func (s *Scanbuf) Size() int {
	return len(s.buf)
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

行为:
	很明显 w 不知道缓冲是否已经读取完毕, 如果 w.Write 返回 (0, nil),
	而更新缓冲遇到 io.EOF, w 无法消耗缓冲数据, 即数据未预期 EOF.
	返回 (n, io.ErrUnexpectedEOF).
*/
func (s *Scanbuf) WriteTo(w io.Writer) (n int64, err error) {
	var (
		c int
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

		if c == 0 && s.err == io.EOF {
			return n, io.ErrUnexpectedEOF
		}

		if s.err != nil && s.err != io.EOF {
			break
		}
	}
	return n, s.err
}
