package tools

import (
	"bufio"
	"bytes"
	"compress/gzip"
	"compress/zlib"
	"crypto/tls"
	"errors"
	"fmt"
	"github.com/andybalholm/brotli"
	"github.com/klauspost/compress/zstd"
	"golang.org/x/net/http2"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

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
	StatusLine string              //返回的歇一天第一行数据 HTTP/1.1 200 OK
	RawHeaders string              // 格式化的头部文本
	Body       []byte              // 响应体
}

// HttpUrl HTTP请求网页函数，支持HTTP2/HTTP1.1，下载文件默认最大支持200M
//
// 参数:
//
//	urlStr          请求的URL地址
//	method          请求模式，如 GET、POST、PUT 等
//	postData        请求的POST数据，如果为空或GET请求填写 []byte("")
//	cookieGo        请求的Cookie，如 _ga_0XM0LYXGC8=GS2.1.s1755523341$o1$g1
//	headersTextGo   请求的协议头，多行请使用换行隔开，如:
//	                User-Agent: Mozilla/5.0...
//	                Accept: */*
//	allowRedirects  是否重定向，重定向填写 true
//	proxyGo         代理地址，如 http://127.0.0.1:8080 或 socks5://127.0.0.1:8080
//	timeout         最大超时时间（秒）
//	MaxResponseSize 下载最大返回包长度，如果填写 0，则最大返回长度为 200MB
//
// 返回:
//
//	err             请求错误，nil 表示请求成功，否则返回实际错误
//	ResponseHeader  返回的协议头
//	ResponseBody    返回的网页实际数据
//
// 示例:
//
//	head := `Sec-Ch-Ua-Arch: "x86"
//	Accept-Encoding: gzip
//	Cookie: PHPSESSION=55523704asafsdffg0
//	User-Agent: Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/139.0.0.0 Safari/537.36
//	Connection: keep-alive
//	`
//	//忽略证书错误
//	_, resp := tools.HttpUrl("https://baidu.com/", "GET", nil, "", head, true, "", 15, 0, true)
//	fmt.Sprintf("%+v", resp)
//
//
//	_, resp :=tools.HttpUrl("https://baidu.com/", "POST", []byte("a=1&b=2"), "", "User-Agent: GoClient/1.0", true, "", 60, 0, false)
//	fmt.Println("=== 返回协议头 ===", resp.RawHeaders)
//	fmt.Println("=== 响应内容 ===", string(resp.Body))
//
// -----------示例2:添加代理访问且Cookie设置为PHPSESSION=AAAAA--如果协议头中也包含Cookie则cookieGo字段优先级更高,优先使用cookieGo----
//
//	_, respHeader, respBody := tools.HttpUrl("https://baidu.com", "POST", []byte("a=1&b=2"), "PHPSESSION=AAAAA", head, true, "http://127.0.0.1:8080", 60, 0)
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
	ignoreCertErrors bool, // 新增参数：是否忽略证书错误
) (error, *HttpResponse) {
	// 设置最大返回包长度
	if MaxResponseSize < 1 {
		MaxResponseSize = 200 * 1024 * 1024
	}

	if postData == nil {
		postData = []byte("")
	}

	// 创建HTTP客户端
	transport := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: ignoreCertErrors}, // 根据参数决定是否忽略证书错误
	}
	if proxyGo != "" {
		proxyUrl, err := url.Parse(proxyGo)
		if err != nil {
			return fmt.Errorf("error: parsing proxy URL: %s", err), nil
		}
		transport.Proxy = http.ProxyURL(proxyUrl)
	}

	// 启用HTTP/2支持
	if err := http2.ConfigureTransport(transport); err != nil {
		return fmt.Errorf("error: http2 configure: %s", err), nil
	}

	client := &http.Client{
		Transport: transport,
		Timeout:   time.Duration(timeout) * time.Second,
	}

	// 创建请求
	req, err := http.NewRequest(method, urlStr, bytes.NewBuffer(postData))
	if err != nil {
		return fmt.Errorf("error: new request: %s", err), nil
	}

	// 设置 Cookie
	if cookieGo != "" {
		req.Header.Set("Cookie", cookieGo)
	}

	// 设置额外协议头
	scanner := bufio.NewScanner(strings.NewReader(headersTextGo))
	for scanner.Scan() {
		line := scanner.Text()
		parts := strings.SplitN(line, ":", 2)
		if len(parts) == 2 {
			req.Header.Set(strings.TrimSpace(parts[0]), strings.TrimSpace(parts[1]))
		}
	}
	if err := scanner.Err(); err != nil {
		return fmt.Errorf("error: parsing headers: %s", err), nil
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
		return fmt.Errorf("error: sending request: %s", err), nil
	}
	defer resp.Body.Close()

	// 限制响应大小
	limitReader := &io.LimitedReader{R: resp.Body, N: MaxResponseSize + 1}
	body, err := io.ReadAll(limitReader)
	if err != nil && !errors.Is(err, io.EOF) {
		return fmt.Errorf("error: reading body: %s", err), nil
	}
	if limitReader.N <= 0 {
		return fmt.Errorf("error: response exceeds max size"), nil
	}

	// 解压缩（gzip/deflate/br/zstd）
	encoding := resp.Header.Get("Content-Encoding")
	switch encoding {
	case "gzip":
		r, err := gzip.NewReader(bytes.NewReader(body))
		if err != nil {
			return fmt.Errorf("error: gzip reader: %s", err), nil
		}
		defer r.Close()
		body, err = io.ReadAll(r)
		if err != nil {
			return fmt.Errorf("error: gzip read: %s", err), nil
		}
	case "deflate":
		r, err := zlib.NewReader(bytes.NewReader(body))
		if err != nil {
			return fmt.Errorf("error: deflate reader: %s", err), nil
		}
		defer r.Close()
		body, err = io.ReadAll(r)
		if err != nil {
			return fmt.Errorf("error: deflate read: %s", err), nil
		}
	case "br":
		r := brotli.NewReader(bytes.NewReader(body))
		body, err = io.ReadAll(r)
		if err != nil {
			return fmt.Errorf("error: brotli read: %s", err), nil
		}
	case "zstd":
		dec, err := zstd.NewReader(bytes.NewReader(body))
		if err != nil {
			return fmt.Errorf("error: zstd reader: %s", err), nil
		}
		defer dec.Close()
		body, err = io.ReadAll(dec)
		if err != nil {
			return fmt.Errorf("error: zstd read: %s", err), nil
		}
	}

	// 构造 RawHeaders
	rawHeaders := ""
	statusLine := fmt.Sprintf("%s %s\r\n", resp.Proto, resp.Status)

	for k, vals := range resp.Header {
		for _, v := range vals {
			rawHeaders += fmt.Sprintf("%s: %s\r\n", k, v)
		}
	}

	// 返回结构体
	return nil, &HttpResponse{
		StatusCode: resp.StatusCode,
		Proto:      resp.Proto,
		Status:     resp.Status,
		Headers:    resp.Header,
		StatusLine: statusLine,
		RawHeaders: rawHeaders,
		Body:       body,
	}
}

// HTTP请求网络 使用结构体请求，方便一些参数来回写很麻烦，用法和HttpUrl()一样
func HttpUrlStruct(req *HttpRequest) (error, *HttpResponse) {
	if req == nil {
		// 返回一个初始化好的 HttpResponse，避免空指针
		return fmt.Errorf("error: req is nil"), &HttpResponse{
			StatusCode: 0,
			Proto:      "",
			Status:     "",
			Headers:    make(map[string][]string),
			StatusLine: "",
			RawHeaders: "",
			Body:       []byte{},
		}
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
	err, resp := HttpUrl(
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

	// 如果返回 nil，就初始化一个空结构体
	if resp == nil {
		resp = &HttpResponse{
			StatusCode: 0,
			Proto:      "",
			Status:     "",
			Headers:    make(map[string][]string),
			StatusLine: "",
			RawHeaders: "",
			Body:       []byte{},
		}
	}

	return err, resp
}
