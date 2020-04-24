package common

import (
	"context"
)

// NotCancelContext Cancel 되지 않도록 정의된 Context
type NotCancelContext struct {
	context.Context
}

// NotCancel 전달된 Context를 취소가 호출되어도 취소되지 않도록 변경한다.
func NotCancel(ctx context.Context) context.Context {
	return &NotCancelContext{ctx}
}

// Done Cancel 시 완료 정보를 보낼 채널을 반환한다.
func (r *NotCancelContext) Done() <-chan struct{} {
	return nil
}
