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
    // WriteTo 会循环调用此处的代码, 并保证 len(s) != 0
    // 以简单的查找换行为例
    off := bytes.IndexByte(s, '\n')
    if off == -1 {
        // 还有数据, 补充缓冲
        if !buf.IsEOF() {
            // 0 是特殊的, 特指补充缓冲
            // nil, 本例中不会产生错误
            return 0, nil
        }
        off = len(s) // 残余数据处理, 在其它场景中也许需要返回错误来结束
    } else {
        off++ // 消耗的字节数, '\n' 只有一个字节, +1 即可.
    }

    // 这里简单的打印示意处理数据.
    fmt.Println(string(s[:off]))

    // 返回消耗的字节数, 指示继续推进, 即便 n 为 0, 缓冲也会更新
    return off, nil
}))

// 最后判断是否有错误发生, 正常情况下总是返回 io.EOF.
if err != io.EOF {
    // 真的有错误发生
}
```

通过 *Scanbuf.Advance 消耗数据
```go
var (
    n    int
    s    []byte
    err  error
    stop bool
)
// Advance 需要循环结构
for !stop {
    // off 指示 Advance 向前推进(消耗)掉的字节数, 从 0 开始
    s, err = buf.Advance(off)

    // 自 go 1.3 可能同时返回数据和 io.EOF.
    // 只要 len(s) 不为零, 先处理数据, 忽略 err.
    if len(s) == 0 {
        break
    }

    off = bytes.IndexByte(p, '\n')

    if off == -1 {
        if !buf.IsEOF() {
            // 0 是特殊的, 特指补充缓冲
            off = 0
            continue
        }
        off = len(s) // 残余数据处理, 其它场景中也许需要退出循环
        stop = true  // 为正确处理数据和 io.EOF 同时返回, 需要此标记
    } else {
        off++
    }

    fmt.Println(string(s[:off]))
}

if err != io.EOF {
    // 真的有错误发生
}
```

WriterTo 和 Advance 的相同点:

* 推进值为 0 表示补充缓冲

区别:

1. WriterTo 返回的错误也可能来自 io.Writer. Advance 返回的错误来自 Scanbuf.
2. WriterTo 可预测 io.ErrUnexpectedEOF.
3. WriterTo 内部有一个循环, Advance 需要调用者构建循环.
4. 结束循环 WriterTo 靠调用者返回错误或数据消耗完, Advance 完全由调用者控制.

特别注意:
自 go 1.3 可能同时返回数据和 io.EOF. 但是事实上不同 io.Reader 实现可能存在差异.

内部细节
========

Scanbuf 内部维护两个游标 start 和 end, 其间的数据传递给调用者处理.
WriterTo 或 Advance 方法发出的推进(消耗)字节数 n 指示 Scanbuf 更新游标.

* n 为 0 时, 补充缓冲数据
* n 非 0 时, 如果 end == start + n, 补充缓冲数据. 否则 start += n.

补充缓冲数据时会先把缓冲数据移动到最开始处, 然后计算剩余空间 size, 如果 size < MinBuffer 扩展缓冲. 最后从 io.Reader 读取数据到 buf[:end]. 如果 io.Reader 为 nil, 设置 io.EOF.

LICENSE
=======
Copyright (c) 2014 The ZxxLang Authors. All rights reserved.
Use of this source code is governed by a BSD-style
license that can be found in the LICENSE file.

