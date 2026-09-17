package main

// JSON-RPC 2.0 over stdio（LSP 传输层，自实现零依赖）。
//
// 帧格式：`Content-Length: N\r\n\r\n<JSON>`（长度按字节计，UTF-8）。

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"strconv"
	"strings"
)

type rpcMessage struct {
	JSONRPC string           `json:"jsonrpc"`
	ID      *json.RawMessage `json:"id,omitempty"`
	Method  string           `json:"method,omitempty"`
	Params  json.RawMessage  `json:"params,omitempty"`
	Result  interface{}      `json:"result,omitempty"`
	Error   *rpcError        `json:"error,omitempty"`
}

type rpcError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

// isRequest 判断是否是需要回包的请求（有 id）。
func (m rpcMessage) isRequest() bool { return m.ID != nil }

type rpcConn struct {
	r *bufio.Reader
	w io.Writer
}

func newRPCConn(r io.Reader, w io.Writer) *rpcConn {
	return &rpcConn{r: bufio.NewReader(r), w: w}
}

// read 读取一条消息；EOF 时返回 io.EOF。
func (c *rpcConn) read() (rpcMessage, error) {
	var msg rpcMessage
	length := -1
	for {
		line, err := c.r.ReadString('\n')
		if err != nil {
			if line == "" {
				return msg, err
			}
		}
		line = strings.TrimRight(line, "\r\n")
		if line == "" {
			break // 头部结束
		}
		if k, v, ok := strings.Cut(line, ":"); ok {
			if strings.EqualFold(strings.TrimSpace(k), "Content-Length") {
				n, cerr := strconv.Atoi(strings.TrimSpace(v))
				if cerr != nil {
					return msg, fmt.Errorf("qklsp: 非法 Content-Length %q", v)
				}
				length = n
			}
		}
		if err != nil {
			return msg, err
		}
	}
	if length < 0 {
		return msg, fmt.Errorf("qklsp: 缺少 Content-Length 头")
	}
	body := make([]byte, length)
	if _, err := io.ReadFull(c.r, body); err != nil {
		return msg, err
	}
	if err := json.Unmarshal(body, &msg); err != nil {
		return msg, fmt.Errorf("qklsp: JSON 解析失败: %w", err)
	}
	return msg, nil
}

func (c *rpcConn) write(v interface{}) error {
	data, err := json.Marshal(v)
	if err != nil {
		return err
	}
	if _, err := fmt.Fprintf(c.w, "Content-Length: %d\r\n\r\n", len(data)); err != nil {
		return err
	}
	_, err = c.w.Write(data)
	return err
}

// reply 回包（请求）。
func (c *rpcConn) reply(id *json.RawMessage, result interface{}) error {
	if id == nil {
		return nil
	}
	return c.write(rpcMessage{JSONRPC: "2.0", ID: id, Result: result})
}

// replyErr 回错误。
func (c *rpcConn) replyErr(id *json.RawMessage, code int, format string, args ...interface{}) error {
	if id == nil {
		return nil
	}
	return c.write(rpcMessage{JSONRPC: "2.0", ID: id, Error: &rpcError{Code: code, Message: fmt.Sprintf(format, args...)}})
}

// notify 发送通知（无 id）。
func (c *rpcConn) notify(method string, params interface{}) error {
	raw, err := json.Marshal(params)
	if err != nil {
		return err
	}
	return c.write(rpcMessage{JSONRPC: "2.0", Method: method, Params: raw})
}
