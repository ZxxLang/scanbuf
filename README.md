Scanbuf
=======

是一个读缓冲, 自动扩展空间尺寸.
缓冲扩展尺寸公式:
```
(len(buf)/512 + 1) * 512
```
当可用缓冲空间小于 512 字节时进行扩展, 并以 512 字节为增量.
否则移动数据使可用空间连续, 然后更新缓冲.
Scanbuf 适用于大量文本扫描的场景, 有利于内存使用率.

使用
====

```go
import "github.com/ZxxLang/scanbuf"
```

Scanbuf 需要输入源, 可提供两个输入源
```go
var p []byte
var r io.Reader
// ... 给 p, r 绑定数据
buf := scanbuf.New(p) // p 作为输入源初始化缓冲数据
buf.Source(r) // 给 buf 绑定 io.Reader 输入源.
```
很明显 len(p) 为 0 且 r 为 nil 是无意义的.

Scanbuf 实现了 io.WriterTo 接口. 获取(消耗)缓冲数据的方法:

* 调用者实现 io.Writer 消耗数据
* 调用者循环调用 Acvance 消耗数据

实现 io.Writer 通过 *Scanbuf.WriterTo 消耗数据
```go
type MyWriter struct{}
func (m *MyWriter) Write (p []byte) (int, error) {
    // 在 Write 执行体中消耗数据, 返回消耗掉的字节数和发生的错误.
    // 代码见下文
}
w := new(MyWriter)
n, err := buf.WriteTo(w)
```

类型 WriterFunc 为纯函数转换到 io.Writer 提供了便利.
```go
n, err := buf.WriteTo(scanbuf.WriterFunc(func(s []byte) (int, error) {
    // WriteTo 保证 len(s) != 0

    // 因为更新缓冲, 要对缓冲内的数据进行内存(移动)复制.
    // 所以应该尽可能的消耗缓冲, 减少更新次数, 有效使用内存.
    n := 0
    for {
        // 以简单的查找换行为例
        off := bytes.IndexByte(s, '\n')
        if off == -1 {
            break
        }

        // 这里简单的 print 示意消耗数据.
        fmt.Println(string(s[:off]))

        // 计算消耗的字节数, '\n' 只有一个字节, +1 即可.
        n += off + 1
        s = s[off+1:]
    }
    // 返回消耗的字节数, 指示继续推进, 即便 n 为 0, 缓冲也会更新
    return n, nil
}))

// 最后判断是否有错误发生, 正常情况下总是返回 io.EOF.
if err != io.EOF {
    // 真的有错误发生
}
```

通过 *Scanbuf.Advance 消耗数据
```go
var (
    n   int
    s   []byte
    err error
)
for {
    // n 指示 Advance 向前推进(消耗)掉的字节数, 从 0 开始
    s, err = buf.Advance(n)

    // 自 go 1.3 可能同时返回数据和 io.EOF.
    // 只要 len(s) 不为零, 先处理数据, 忽略 err.
    if len(s) == 0 {
        break
    }

    n = 0
    for {
        off := bytes.IndexByte(p, '\n')
        if off == -1 {
            break
        }

        fmt.Println(string(s[:off]))

        n += off + 1
        s = s[off+1:]
    }

    if err != nil {
        break
    }
}

if err != io.EOF {
    // 真的有错误发生
}
```

WriterTo 和 Advance 的区别在于 WriterTo 返回的错误可能来自 io.Writer.

LICENSE
=======
Copyright (c) 2014 The ZxxLang Authors. All rights reserved.
Use of this source code is governed by a BSD-style
license that can be found in the LICENSE file.

