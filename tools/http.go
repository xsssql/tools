package tools

import (
	"bufio"
	"bytes"
	"compress/gzip"
	"compress/zlib"
	"crypto/tls"
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
//	_, respHeader, respBody := tools.HttpUrl("https://baidu.com", "GET", []byte(""), "", head, true, "", 60, 0)
//	fmt.Println("=== 返回协议头 ===", respHeader)
//	fmt.Println("=== 响应内容 ===", string(respBody))
//
// -----------示例2:添加代理访问且Cookie设置为PHPSESSION=AAAAA--如果协议头中也包含Cookie则cookieGo字段优先级更高,优先使用cookieGo----
//
//	_, respHeader, respBody := tools.HttpUrl("https://baidu.com", "POST", []byte("a=1&b=2"), "PHPSESSION=AAAAA", head, true, "http://127.0.0.1:8080", 60, 0)
func HttpUrl(urlStr string, method string, postData []byte, cookieGo string, headersTextGo string, allowRedirects bool, proxyGo string, timeout int, MaxResponseSize int64) (err error, ResponseHeader string, ResponseBody []byte) {
	// 设置最大返回包长度
	errCode := fmt.Errorf("")
	ResponseDataBody := []byte("") //返回的数据
	maxResponseSize := MaxResponseSize
	if maxResponseSize < 1 {
		maxResponseSize = 200 * 1024 * 1024
	}

	// 创建HTTP客户端
	transport := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
	if proxyGo != "" {
		proxyUrl, err := url.Parse(proxyGo)
		if err != nil {
			errCode = fmt.Errorf("error:Error parsing proxy URL: %s", err)
			return errCode, "", ResponseDataBody
		}

		transport.Proxy = http.ProxyURL(proxyUrl)
	}

	// 启用HTTP/2支持
	err = http2.ConfigureTransport(transport) // 这一步启用HTTP/2支持
	if err != nil {
		errCode = fmt.Errorf("error:http2 Error: %s", err)
		return errCode, "", ResponseDataBody
	}

	client := &http.Client{
		Transport: transport,
		Timeout:   time.Duration(timeout) * time.Second,
	}

	// 创建实际请求
	req, err := http.NewRequest(method, urlStr, bytes.NewBuffer(postData))
	if err != nil {
		errCode = fmt.Errorf("error: %s", err)
		return errCode, "", ResponseDataBody
	}

	// 设置Cookie和附加协议头
	if cookieGo != "" {
		req.Header.Set("Cookie", cookieGo)
	}
	// 设置附加协议头
	scanner := bufio.NewScanner(strings.NewReader(headersTextGo))
	for scanner.Scan() {
		header := scanner.Text()
		parts := strings.SplitN(header, ":", 2)
		if len(parts) == 2 {
			req.Header.Set(strings.TrimSpace(parts[0]), strings.TrimSpace(parts[1]))
			//强制只使用gzip解压
			/*	if strings.TrimSpace(parts[0]) == "Accept-Encoding" {
					req.Header.Set("Accept-Encoding", "gzip")
				} else {
					req.Header.Set(strings.TrimSpace(parts[0]), strings.TrimSpace(parts[1]))
				}*/

		}
	}
	if err = scanner.Err(); err != nil {
		errCode = fmt.Errorf("error: Error headers %s", err)
		return errCode, "", ResponseDataBody
	}

	// 如果不允许重定向，则禁用重定向
	if !allowRedirects {
		client.CheckRedirect = func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		}
	}

	// 发送请求
	resp, err := client.Do(req)
	if err != nil {
		errCode = fmt.Errorf("error: Error sending request %s", err)
		return errCode, "", ResponseDataBody
	}
	defer resp.Body.Close()

	// 检查响应的长度是否超过限制
	limitReader := &io.LimitedReader{R: resp.Body, N: maxResponseSize + 1}
	responseBody, err := io.ReadAll(limitReader)
	if err != nil && err != io.EOF {

		errCode = fmt.Errorf("error: Error reading response body %s", err)
		return errCode, "", ResponseDataBody
	}

	// 检查是否超过了最大响应大小
	if limitReader.N <= 0 {
		errCode = fmt.Errorf("error: File size exceeds limit  %s", err)
		return errCode, "", ResponseDataBody
	}
	// 解压数据（如果服务器返回的数据是gzip压缩的）
	encoding := resp.Header.Get("Content-Encoding")
	var reader io.ReadCloser
	/*	if resp.Header.Get("Content-Encoding") == "gzip" {
		reader, err = gzip.NewReader(bytes.NewReader(responseBody))
		if err != nil {
			errStr := fmt.Sprintf("-1Error: creating gzip reader: %s", err)
			BackDataBody := []byte("")
			return 1, errStr, BackDataBody
		}
		defer reader.Close()
		responseBody, err = io.ReadAll(reader)
		if err != nil {
			errStr := fmt.Sprintf("-1Error: Error reading gzip response body: %s", err)
			BackDataBody := []byte("")
			return 1, errStr, BackDataBody
		}
	}*/

	switch encoding {
	case "gzip":
		reader, err = gzip.NewReader(bytes.NewReader(responseBody))
		if err != nil {
			errCode = fmt.Errorf("error: creating gzip reader %s", err)
			return errCode, "", ResponseDataBody
		}
		defer reader.Close()
		responseBody, err = io.ReadAll(reader)
		if err != nil {
			errCode = fmt.Errorf("error: reading gzip body %s", err)
			return errCode, "", ResponseDataBody
		}

	case "deflate":
		reader, err = zlib.NewReader(bytes.NewReader(responseBody))
		if err != nil {
			errCode = fmt.Errorf("error: creating deflate reader %s", err)
			return errCode, "", ResponseDataBody
		}
		defer reader.Close()
		responseBody, err = io.ReadAll(reader)
		if err != nil {
			errCode = fmt.Errorf("error: reading deflate body %s", err)
			return errCode, "", ResponseDataBody
		}

	case "br":
		reader = io.NopCloser(brotli.NewReader(bytes.NewReader(responseBody)))
		defer reader.Close()
		responseBody, err = io.ReadAll(reader)
		if err != nil {
			errCode = fmt.Errorf("error: reading brotli body %s", err)
			return errCode, "", ResponseDataBody
		}

	case "zstd":
		dec, err := zstd.NewReader(bytes.NewReader(responseBody))
		if err != nil {
			errCode = fmt.Errorf("error: creating zstd reader %s", err)
			return errCode, "", ResponseDataBody
		}
		defer dec.Close()
		responseBody, err = io.ReadAll(dec)
		if err != nil {
			errCode = fmt.Errorf("error: reading zstd body %s", err)
			return errCode, "", ResponseDataBody
		}
	default:
		// 没压缩或者暂不支持的压缩方式
		// 什么都不做，直接用原始 responseBody
	}

	// 构建 返回结果
	// 构建头部 字符串
	headerStr := fmt.Sprintf("%s %s\r\n", resp.Proto, resp.Status)
	for key, values := range resp.Header {
		for _, value := range values {
			headerStr += fmt.Sprintf("%s: %s\r\n", key, value)
		}
	}

	// 构建返回结果
	return nil, headerStr, responseBody

}
