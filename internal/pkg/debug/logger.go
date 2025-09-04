package debug

import (
	"runtime"
	"strconv"

	"gin-example/internal/pkg/core"
	"gin-example/internal/pkg/trace"
)

type debug struct {
	ctx core.StdContext
}

func WithContext(ctx core.StdContext) *debug {
	return &debug{
		ctx: ctx,
	}
}

func (d *debug) Logger(value ...any) {
	debugInfo := new(trace.Debug)

	defer func() {
		if d.ctx.Trace != nil {
			debugInfo.Stack = fileWithLineNum()
			debugInfo.Value = value
			d.ctx.Trace.AppendDebug(debugInfo)
		}
	}()
}

func fileWithLineNum() string {
	_, file, line, ok := runtime.Caller(3)
	if ok {
		return file + ":" + strconv.FormatInt(int64(line), 10)
	}

	return ""
}
