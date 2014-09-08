Scanbuf
=======

是一个自动扩展的读缓冲, 目的是提高内存使用率.
Scanbuf 兼容 io.WriterTo 和 io.Reader 接口.
io.WriterTo 接口消耗缓冲中的数据, 根据消耗的字节数 Scanbuf 从输入源中补充数据. 并判定是否需要扩展缓冲.
io.Reader 接口可以把剩余数据输出给调用者.

使用
====

```go
import "github.com/ZxxLang/scanbuf"
```

建立 Scanbuf 需要一个 io.Reader 输入源
```go
var r io.Reader
// ...
buf := scanbuf.New(r)
```

调用者可以实现一个 io.Writer 消耗数据
```go
var w io.Writer
// ...
n, err := buf.WriteTo(w)
// ...
```

Scanbuf.WriteTo 内部持续调用 w.Write, 依据其返回值确定对缓冲更新或扩展尺寸.
直到输入源的数据被读取完或者发生错误.

调用者也可以通过 Advance 消耗数据
```go
n := 0 // n 是每次处理 p 所消耗掉的字节数, 指示 buf 向前推进更新缓冲
for {
    p, err := buf.Advance(n)
    // ... n += ...
    if err != nil {
        break
    }
}
```


LICENSE
=======
Copyright (c) 2014 The ZxxLang Authors. All rights reserved.
Use of this source code is governed by a BSD-style
license that can be found in the LICENSE file.

