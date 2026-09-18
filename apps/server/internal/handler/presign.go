package handler

import (
	"errors"
	"net/http"
)

// errTestPresign 仅用于测试注入的哨兵错误（生产路径不会用到）。
var errTestPresign = errors.New("presign failed")

// writePresignResult 统一写出预签名结果。
//
// 预签名并非「不可能失败」：AWS SDK 在取凭证、序列化输入等阶段都会返回错误
// （实测空 key、context 取消均可触发）。原实现三处写成 `u, _ := client.PresignXxx(...)`，
// 失败时返回 `200 {"url":""}`——调用方拿到空 URL 却看到成功状态，无从判断是
// 服务端出错还是自身渲染问题（todolist #23 / ASSESSMENT L1）。
//
// 返回 ok=false 表示已写出错误响应（调用方应直接 return）。
func (h *Handler) writePresignResult(w http.ResponseWriter, err error, body map[string]any) bool {
	if err != nil {
		h.writeErr(w, http.StatusInternalServerError, "failed to create presigned url")
		return false
	}
	h.writeJSON(w, http.StatusOK, body)
	return true
}
