package task

import (
	"context"
	"github.com/injoyai/notice/pkg/push"
	"github.com/injoyai/notice/pkg/push/serverchan"
	"github.com/injoyai/tdx"
)

func NewNoticeServerChan(sendKey, message string) *NoticeServerChan {
	return &NoticeServerChan{
		ServerChan: serverchan.New(sendKey),
		message:    message,
	}
}

type NoticeServerChan struct {
	*serverchan.ServerChan
	message string
}

func (this *NoticeServerChan) Name() string {
	return "通知到Server酱"
}

func (this *NoticeServerChan) Run(ctx context.Context, m *tdx.Manage) error {
	if len(this.DefaultSendKey) == 0 {
		return nil
	}
	return this.ServerChan.Push(&push.Message{
		Title:   "pull-tdx",
		Content: this.message,
	})
}
