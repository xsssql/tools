package tools

import (
	"bufio"
	"bytes"
	"crypto/tls"
	"fmt"
	"golang.org/x/net/http2"
	"io"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"
)

// http请求 全局资源池
var (
	transportPool sync.Map // key: proxy|ignoreCert
	clientPool    sync.Map // key: transportKey|timeout
)

// DNS Cache Resolver
var defaultResolver = &net.Resolver{
	PreferGo: true,
}

// HttpRequest 定义请求参数
type HttpRequest struct {
	URL              string // 请求的URL
	Method           string // GET/POST/PUT...
	PostData         []byte // POST数据，GET时填nil或[]byte("")
	Cookie           string // 请求Cookie
	Headers          string // 多行协议头
	AllowRedirects   bool   // 是否允许重定向
	Proxy            string // 代理地址
	Timeout          int    // 超时秒数
	MaxResponseSize  int64  // 最大返回数据长度，0表示默认200MB
	IgnoreCertErrors bool   // 是否忽略自签证书错误
}

// HttpResponse 封装返回的内容
type HttpResponse struct {
	StatusCode int                 // 状态码 200、302、301
	Proto      string              // 协议版本，如 HTTP/1.1
	Status     string              // 状态文本，如 "200 OK" 或 "302 Found"
	Headers    map[string][]string // 原始响应头
	StatusLine string              // 返回的第一行数据 HTTP/1.1 200 OK
	RawHeaders string              // 格式化的头部文本
	Body       []byte              // 响应体
}

// Transport Key
func transportKey(proxy string, ignoreCert bool) string {
	return proxy + "|" + strconv.FormatBool(ignoreCert)
}

// 获取 Transport
func getTransport(proxyGo string, ignoreCertErrors bool) (*http.Transport, error) {
	key := transportKey(proxyGo, ignoreCertErrors)

	if v, ok := transportPool.Load(key); ok {
		return v.(*http.Transport), nil
	}

	dialer := &net.Dialer{
		Timeout:   30 * time.Second,
		KeepAlive: 30 * time.Second,
		Resolver:  defaultResolver,
	}

	tr := &http.Transport{
		DialContext:           dialer.DialContext,
		TLSHandshakeTimeout:   10 * time.Second,
		IdleConnTimeout:       90 * time.Second,
		MaxIdleConns:          1000,
		MaxIdleConnsPerHost:   100,
		ExpectContinueTimeout: 1 * time.Second,
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: ignoreCertErrors,
		},
	}

	if proxyGo != "" {
		proxyURL, err := url.Parse(proxyGo)
		if err != nil {
			return nil, err
		}
		tr.Proxy = http.ProxyURL(proxyURL)
	}

	if err := http2.ConfigureTransport(tr); err != nil {
		return nil, err
	}

	actual, _ := transportPool.LoadOrStore(key, tr)
	return actual.(*http.Transport), nil
}

// http.Client 池
func getClient(tr *http.Transport, timeout int, allowRedirects bool) *http.Client {
	key := fmt.Sprintf("%p|%d|%v", tr, timeout, allowRedirects)

	if v, ok := clientPool.Load(key); ok {
		return v.(*http.Client)
	}

	client := &http.Client{
		Transport: tr,
		Timeout:   time.Duration(timeout) * time.Second,
	}

	if !allowRedirects {
		client.CheckRedirect = func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		}
	}

	actual, _ := clientPool.LoadOrStore(key, client)
	return actual.(*http.Client)
}

// newEmptyResponse 创建一个空的响应结构体
func newEmptyResponse() *HttpResponse {
	return &HttpResponse{
		StatusCode: 0,
		Proto:      "",
		Status:     "",
		Headers:    make(map[string][]string),
		StatusLine: "",
		RawHeaders: "",
		Body:       []byte{},
	}
}

// HttpUrl HTTP请求网页函数，支持HTTP2/HTTP1.1，下载文件默认最大支持200M
//
// 参数:
//
//	urlStr          请求的URL地址
//	method          请求模式，如 GET、POST、PUT 等
//	postData        请求的POST数据，如果为空或GET请求填写 []byte("")
//	cookieGo        请求的Cookie，优先级高于请求协议头里面的Cookie 如 _ga_0XM0LYXGC8=GS2.1.s1755523341$o1$g1
//	headersTextGo   请求的协议头，多行请使用换行隔开
//	allowRedirects  是否重定向，重定向填写 true
//	proxyGo         代理地址，如 http://127.0.0.1:8080 或 socks5://127.0.0.1:8080
//	timeout         最大超时时间（秒）
//	MaxResponseSize 下载最大返回包长度，如果填写 0，则最大返回长度为 200MB
//	ignoreCertErrors 是否忽略证书错误 true=忽略 false=不忽略
//
// 返回:
//
//	error           请求错误，nil 表示请求成功
//	*HttpResponse   响应结构体，始终返回有效对象
func HttpUrl(
	urlStr string,
	method string,
	postData []byte,
	cookieGo string,
	headersTextGo string,
	allowRedirects bool,
	proxyGo string,
	timeout int,
	MaxResponseSize int64,
	ignoreCertErrors bool,
) (error, *HttpResponse) {

	respObj := &HttpResponse{
		Headers: make(http.Header),
		Body:    []byte{},
	}

	if MaxResponseSize <= 0 {
		MaxResponseSize = 200 * 1024 * 1024
	}
	if postData == nil {
		postData = []byte{}
	}

	tr, err := getTransport(proxyGo, ignoreCertErrors)
	if err != nil {
		return err, respObj
	}

	client := getClient(tr, timeout, allowRedirects)

	req, err := http.NewRequest(method, urlStr, bytes.NewReader(postData))
	if err != nil {
		return err, respObj
	}

	headers, headerCookie := parseHeaders(headersTextGo)
	for k, v := range headers {
		for _, vv := range v {
			req.Header.Add(k, vv)
		}
	}

	cookieMap := mergeCookiesToMap(headerCookie, cookieGo)

	for name, value := range cookieMap {
		req.AddCookie(&http.Cookie{
			Name:  name,
			Value: value,
		})
	}

	resp, err := client.Do(req)
	if err != nil {
		return err, respObj
	}
	defer resp.Body.Close()

	limit := &io.LimitedReader{R: resp.Body, N: MaxResponseSize + 1}
	body, _ := io.ReadAll(limit)
	if limit.N <= 0 {
		body = body[:MaxResponseSize]
	}

	respObj.StatusCode = resp.StatusCode
	respObj.Status = resp.Status
	respObj.Proto = resp.Proto
	respObj.Headers = resp.Header
	respObj.Body = body
	respObj.StatusLine = fmt.Sprintf("%s %s\r\n", resp.Proto, resp.Status)
	respObj.RawHeaders = formatHeaders(resp.Header)

	return nil, respObj
}

// normalizeCookieInput 去除cookieGo里面的cookie: 关键字
func normalizeCookieInput(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return ""
	}
	if strings.HasPrefix(strings.ToLower(s), "cookie:") {
		s = strings.TrimSpace(s[len("cookie:"):])
	}
	return s
}

// mergeCookies 将协议头里面已经有的Cookie和cookieGo 想通的进行合并,cookieGo 优先级最高
func mergeCookiesToMap(headerCookie, cookieGo string) map[string]string {
	m := make(map[string]string)

	parse := func(s string) {
		s = normalizeCookieInput(s)
		if s == "" {
			return
		}
		for _, p := range strings.Split(s, ";") {
			p = strings.TrimSpace(p)
			if p == "" {
				continue
			}
			kv := strings.SplitN(p, "=", 2)
			if len(kv) != 2 {
				continue
			}

			name := strings.TrimSpace(kv[0])
			value := strings.TrimSpace(kv[1])

			if name == "" {
				continue
			}

			// 🔥 关键：清洗 value
			m[name] = sanitizeCookieValue(value)
		}
	}

	parse(headerCookie)
	parse(cookieGo)

	return m
}

func sanitizeCookieValue(s string) string {
	var b strings.Builder
	for _, r := range s {
		// RFC + net/http 要求：不能有控制字符
		if r < 32 || r == 127 {
			continue
		}
		b.WriteRune(r)
	}
	return b.String()
}

// parseHeaders Header 解析
func parseHeaders(text string) (http.Header, string) {
	h := make(http.Header)
	var cookie string

	scanner := bufio.NewScanner(strings.NewReader(text))
	for scanner.Scan() {
		line := scanner.Text()
		kv := strings.SplitN(line, ":", 2)
		if len(kv) != 2 {
			continue
		}
		key := strings.TrimSpace(kv[0])
		val := strings.TrimSpace(kv[1])

		if strings.EqualFold(key, "Cookie") {
			cookie = val
			continue
		}
		h.Add(key, val)
	}
	return h, cookie
}

// formatHeaders 格式化响应头
func formatHeaders(headers map[string][]string) string {
	rawHeaders := ""
	for k, vals := range headers {
		for _, v := range vals {
			rawHeaders += fmt.Sprintf("%s: %s\r\n", k, v)
		}
	}
	return rawHeaders
}

// HttpUrlStruct HTTP请求网络 使用结构体请求
func HttpUrlStruct(req *HttpRequest) (error, *HttpResponse) {
	if req == nil {
		return fmt.Errorf("error: req is nil"), newEmptyResponse()
	}

	// 对结构体参数做默认值处理
	if req.PostData == nil {
		req.PostData = []byte("")
	}
	if req.Headers == "" {
		req.Headers = ""
	}
	if req.Timeout <= 0 {
		req.Timeout = 30 // 默认超时30秒
	}
	if req.MaxResponseSize < 1 {
		req.MaxResponseSize = 200 * 1024 * 1024
	}

	// 直接调用原来的 HttpUrl 函数
	return HttpUrl(
		req.URL,
		req.Method,
		req.PostData,
		req.Cookie,
		req.Headers,
		req.AllowRedirects,
		req.Proxy,
		req.Timeout,
		req.MaxResponseSize,
		req.IgnoreCertErrors,
	)
}

/*func HttpUrl_old(
	urlStr string,
	method string,
	postData []byte,
	cookieGo string,
	headersTextGo string,
	allowRedirects bool,
	proxyGo string,
	timeout int,
	MaxResponseSize int64,
	ignoreCertErrors bool,
) (error, *HttpResponse) {
	// 初始化响应结构体，确保始终返回有效对象
	response := &HttpResponse{
		StatusCode: 0,
		Proto:      "",
		Status:     "",
		Headers:    make(http.Header),
		StatusLine: "",
		RawHeaders: "",
		Body:       []byte{},
	}

	// 设置最大返回包长度
	if MaxResponseSize < 1 {
		MaxResponseSize = 200 * 1024 * 1024
	}

	if postData == nil {
		postData = []byte("")
	}

	// 创建HTTP客户端
	transport := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: ignoreCertErrors},
	}
	if proxyGo != "" {
		proxyUrl, err := url.Parse(proxyGo)
		if err != nil {
			return fmt.Errorf("error: parsing proxy URL: %s", err), response
		}
		transport.Proxy = http.ProxyURL(proxyUrl)
	}

	// 启用HTTP/2支持
	if err := http2.ConfigureTransport(transport); err != nil {
		return fmt.Errorf("error: http2 configure: %s", err), response
	}

	client := &http.Client{
		Transport: transport,
		Timeout:   time.Duration(timeout) * time.Second,
	}

	// 创建请求
	req, err := http.NewRequest(method, urlStr, bytes.NewBuffer(postData))
	if err != nil {
		return fmt.Errorf("error: new request: %s", err), response
	}

	// 设置额外协议头
	scanner := bufio.NewScanner(strings.NewReader(headersTextGo))
	var headerCookie string
	for scanner.Scan() {
		line := scanner.Text()
		parts := strings.SplitN(line, ":", 2)
		if len(parts) != 2 {
			continue
		}

		key := strings.TrimSpace(parts[0])
		val := strings.TrimSpace(parts[1])

		if strings.EqualFold(key, "Cookie") {
			headerCookie = val
			continue
		}

		req.Header.Set(key, val)
	}

	// 合并并设置 Cookie
	mergedCookie := mergeCookies(headerCookie, cookieGo)
	if mergedCookie != "" {
		req.Header.Set("Cookie", mergedCookie)
	}

	if err := scanner.Err(); err != nil {
		return fmt.Errorf("error: parsing headers: %s", err), response
	}

	// 禁止重定向时，返回原始响应
	if !allowRedirects {
		client.CheckRedirect = func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		}
	}

	// 发送请求
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("error: sending request: %s", err), response
	}
	defer resp.Body.Close()

	// 限制响应大小
	limitReader := &io.LimitedReader{R: resp.Body, N: MaxResponseSize + 1}
	body, err := io.ReadAll(limitReader)
	if err != nil && !errors.Is(err, io.EOF) {
		// 即使读取失败，也返回部分响应信息
		return fmt.Errorf("error: reading body: %s", err), &HttpResponse{
			StatusCode: resp.StatusCode,
			Proto:      resp.Proto,
			Status:     resp.Status,
			Headers:    resp.Header,
			StatusLine: fmt.Sprintf("%s %s\r\n", resp.Proto, resp.Status),
			RawHeaders: formatHeaders(resp.Header),
			Body:       body,
		}
	}
	if limitReader.N <= 0 {
		// 超过大小限制，但返回部分响应信息
		return fmt.Errorf("error: response exceeds max size"), &HttpResponse{
			StatusCode: resp.StatusCode,
			Proto:      resp.Proto,
			Status:     resp.Status,
			Headers:    resp.Header,
			StatusLine: fmt.Sprintf("%s %s\r\n", resp.Proto, resp.Status),
			RawHeaders: formatHeaders(resp.Header),
			Body:       body[:MaxResponseSize],
		}
	}

	// 解压缩（gzip/deflate/br/zstd）
	encoding := resp.Header.Get("Content-Encoding")
	switch encoding {
	case "gzip":
		r, err := gzip.NewReader(bytes.NewReader(body))
		if err != nil {
			return fmt.Errorf("error: gzip reader: %s", err), &HttpResponse{
				StatusCode: resp.StatusCode,
				Proto:      resp.Proto,
				Status:     resp.Status,
				Headers:    resp.Header,
				StatusLine: fmt.Sprintf("%s %s\r\n", resp.Proto, resp.Status),
				RawHeaders: formatHeaders(resp.Header),
				Body:       body,
			}
		}
		defer r.Close()
		body, err = io.ReadAll(r)
		if err != nil {
			return fmt.Errorf("error: gzip read: %s", err), &HttpResponse{
				StatusCode: resp.StatusCode,
				Proto:      resp.Proto,
				Status:     resp.Status,
				Headers:    resp.Header,
				StatusLine: fmt.Sprintf("%s %s\r\n", resp.Proto, resp.Status),
				RawHeaders: formatHeaders(resp.Header),
				Body:       []byte{},
			}
		}
	case "deflate":
		r, err := zlib.NewReader(bytes.NewReader(body))
		if err != nil {
			return fmt.Errorf("error: deflate reader: %s", err), &HttpResponse{
				StatusCode: resp.StatusCode,
				Proto:      resp.Proto,
				Status:     resp.Status,
				Headers:    resp.Header,
				StatusLine: fmt.Sprintf("%s %s\r\n", resp.Proto, resp.Status),
				RawHeaders: formatHeaders(resp.Header),
				Body:       body,
			}
		}
		defer r.Close()
		body, err = io.ReadAll(r)
		if err != nil {
			return fmt.Errorf("error: deflate read: %s", err), &HttpResponse{
				StatusCode: resp.StatusCode,
				Proto:      resp.Proto,
				Status:     resp.Status,
				Headers:    resp.Header,
				StatusLine: fmt.Sprintf("%s %s\r\n", resp.Proto, resp.Status),
				RawHeaders: formatHeaders(resp.Header),
				Body:       []byte{},
			}
		}
	case "br":
		r := brotli.NewReader(bytes.NewReader(body))
		body, err = io.ReadAll(r)
		if err != nil {
			return fmt.Errorf("error: brotli read: %s", err), &HttpResponse{
				StatusCode: resp.StatusCode,
				Proto:      resp.Proto,
				Status:     resp.Status,
				Headers:    resp.Header,
				StatusLine: fmt.Sprintf("%s %s\r\n", resp.Proto, resp.Status),
				RawHeaders: formatHeaders(resp.Header),
				Body:       []byte{},
			}
		}
	case "zstd":
		dec, err := zstd.NewReader(bytes.NewReader(body))
		if err != nil {
			return fmt.Errorf("error: zstd reader: %s", err), &HttpResponse{
				StatusCode: resp.StatusCode,
				Proto:      resp.Proto,
				Status:     resp.Status,
				Headers:    resp.Header,
				StatusLine: fmt.Sprintf("%s %s\r\n", resp.Proto, resp.Status),
				RawHeaders: formatHeaders(resp.Header),
				Body:       body,
			}
		}
		defer dec.Close()
		body, err = io.ReadAll(dec)
		if err != nil {
			return fmt.Errorf("error: zstd read: %s", err), &HttpResponse{
				StatusCode: resp.StatusCode,
				Proto:      resp.Proto,
				Status:     resp.Status,
				Headers:    resp.Header,
				StatusLine: fmt.Sprintf("%s %s\r\n", resp.Proto, resp.Status),
				RawHeaders: formatHeaders(resp.Header),
				Body:       []byte{},
			}
		}
	}

	// 构造 StatusLine
	statusLine := fmt.Sprintf("%s %s\r\n", resp.Proto, resp.Status)

	// 返回结构体
	return nil, &HttpResponse{
		StatusCode: resp.StatusCode,
		Proto:      resp.Proto,
		Status:     resp.Status,
		Headers:    resp.Header,
		StatusLine: statusLine,
		RawHeaders: formatHeaders(resp.Header),
		Body:       body,
	}
}
*/
