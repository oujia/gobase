package gobase

import (
	"net/http"
	"net/url"
	"time"
	"errors"
	"fmt"
	"io/ioutil"
	"strings"
	"io"
	"github.com/gin-gonic/gin"
)

type Reader struct {
	Reader io.ReaderAt
	Offset int64
}

func (p *Reader) Read(val []byte) (n int, err error) {
	n, err = p.Reader.ReadAt(val, p.Offset)
	p.Offset += int64(n)
	return
}

func (p *Reader) Close() error {
	if rc, ok := p.Reader.(io.ReadCloser); ok {
		return rc.Close()
	}
	return nil
}

const (
	tryTimes = 3
	defaultTimeout = 5
)

type DwHttp struct {
	Ctx *gin.Context
	Log *DwLog
}


func (dwHttp *DwHttp)NewGet(api string,timeoutSecond int, header map[string]string, proxy string) (string, error)  {
	client := newClient(timeoutSecond, proxy)
	return dwHttp.doRequest(client, http.MethodGet, api, nil, header)
}

func (dwHttp *DwHttp)SimpleGet(api string) (string, error) {
	client := newClient(defaultTimeout, "")
	return dwHttp.doRequest(client, http.MethodGet, api, nil, nil)
}

func (dwHttp *DwHttp)NewPost(api string, data map[string]string, timeoutSecond int, header map[string]string, proxy string) (string, error) {
	client := newClient(timeoutSecond, proxy)
	return dwHttp.doRequest(client, http.MethodPost, api, data, header)
}

func (dwHttp *DwHttp)SimplePost(api string, data map[string]string) (string, error) {
	client := newClient(defaultTimeout, "")
	return dwHttp.doRequest(client, http.MethodPost, api, data, nil)
}

func newClient(timeoutSecond int, proxy string) *http.Client {
	client := &http.Client{
		Timeout:   time.Duration(timeoutSecond) * time.Second,
	}

	if proxyURL, err := url.Parse(proxy); proxy != "" && err == nil {
		client.Transport = &http.Transport{Proxy: http.ProxyURL(proxyURL)}
	}

	return client
}

func defaultHeader(header *http.Header)  {
	header.Add("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,image/webp,image/apng,*/*;q=0.8");
	header.Add("Accept-Language", "zh-CN,zh;q=0.9,en;q=0.8,zh-TW;q=0.7")
	header.Add("Connection", "keep-alive")
	header.Add("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_13_4) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/66.0.3359.139 Safari/537.36")
}

func (dwHttp *DwHttp)doRequest(client *http.Client, method string, api string, data map[string]string, header map[string]string) (string, error) {

	var err error
	var req *http.Request
	var resp *http.Response
	var reader *strings.Reader
	var val *url.Values
	var _param string

	if method == http.MethodPost {
		val = &url.Values{}
		for k, v := range data {
			val.Set(k, v)
		}
		_param = val.Encode()
		reader = strings.NewReader(_param)
		req, err = http.NewRequest(http.MethodPost, api, &Reader{reader, 0})
		req.Header.Add("Content-Type", "application/x-www-form-urlencoded")
	} else {
		req, err = http.NewRequest(http.MethodGet, api, nil)
		_param = ""
	}

	if err != nil {
		return "", err
	}

	if header == nil {
		defaultHeader(&req.Header)
	} else {
		for k, v := range header {
			req.Header.Add(k, v)
		}
	}

	_start := time.Now()
	for i := 0; i < tryTimes; i++ {
		r, e := client.Do(req)
		if e == nil {
			resp = r
			break
		}

		if req.Method == http.MethodPost {
			req.Body = &Reader{reader, 0} //重新提交重置偏移
		}

		err = e
	}

	if resp != nil {
		defer resp.Body.Close()
	}

	if err != nil {
		return "", err
	}

	if (resp.StatusCode != http.StatusOK) {
		return "", errors.New(fmt.Sprintf("request %s error, code:%d", req.URL.String(), resp.StatusCode))
	}

	body , _ := ioutil.ReadAll(resp.Body)
	strBody := string(body)

	delay := time.Since(_start).Seconds()
	dwHttp.Log.NewModuleLog(dwHttp.Ctx, method, api, _param, delay, strBody)
	return strBody, nil
}