package gobase

import (
	"fmt"
	"time"
	"math/rand"
	"github.com/gin-gonic/gin"
	"net/http"
	"bytes"
	"strings"
	"encoding/json"
	"github.com/gomodule/redigo/redis"
)

type bodyLogWriter struct {
	gin.ResponseWriter
	body *bytes.Buffer
}

func (w bodyLogWriter) Write(b []byte) (int, error) {
	w.body.Write(b)
	return w.ResponseWriter.Write(b)
}

func genCallId() int64 {
	seed := time.Now().UnixNano()
	rand.Seed(seed)
	suffix := rand.Intn(999 - 100) + 100

	return seed % 100000000 + int64(suffix)
}

type DwLog struct {
	LogKey string
	SelfCall string
	ModuleCall string
	RedisPool *redis.Pool
}

func (l *DwLog)NewModuleLog(ctx *gin.Context, method string, toUrl string, params string,  delay float64, response string)  {
	data := make(map[string]interface{})

	data["from_call_id"] = ctx.Request.Header.Get("call_id")
	uriList := strings.Split(ctx.Request.RequestURI, "?")
	data["from_url"] = uriList[0]
	data["to_url"] = toUrl
	data["method"] = method
	uriList2 := strings.Split(toUrl, "?")
	if len(uriList2) > 1 && method == http.MethodGet {
		data["param"] = uriList2[1]
	} else {
		data["param"] = params
	}
	data["response"] = cutString(response)
	data["code"] = genCode(response)
	data["delay"] = delay
	server_ip, _ := ExternalIP()
	data["server_ip"] = server_ip

	l.pushLog(l.ModuleCall, data)
}

func (l *DwLog)NewSelfLog() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		_start := time.Now()

		call_id := ctx.Query("call_id")
		if call_id == "" {
			call_id = fmt.Sprintf("%d", genCallId())
		}

		ctx.Request.Header.Add("call_id", call_id)

		//hook?
		blw := &bodyLogWriter{body: bytes.NewBufferString(""), ResponseWriter: ctx.Writer}
		ctx.Writer = blw

		ctx.Next()

		data := make(map[string]interface{})
		data["call_id"] = call_id
		uriList := strings.Split(ctx.Request.RequestURI, "?")
		data["url"] = uriList[0]
		data["method"] = ctx.Request.Method
		data["param"] = genParam(ctx, uriList)
		data["cookie"] = genCookie(ctx.Request.Cookies())
		data["response"] = cutString(blw.body.String())
		data["code"] = genCode(data["response"].(string))
		data["delay"] = time.Since(_start).Seconds()
		server_ip, _ := ExternalIP()
		data["server_ip"] = server_ip
		data["client_ip"] = ctx.ClientIP()
		data["useragent"] = ctx.Request.UserAgent()
		data["referer"] = ctx.Request.Referer()

		l.pushLog(l.SelfCall, data)
	}
}

func genParam(ctx *gin.Context, uriList []string) string {
	var params string
	if len(uriList) > 1 && ctx.Request.Method == http.MethodGet {
		params = uriList[1]
	} else {
		ctx.Request.ParseForm()
		params = cutString(ctx.Request.Form.Encode())
	}

	return params
}

func cutString(str string) string {
	if len(str) == 0 {
		return str
	}

	_rp := []rune(str)
	_end := len(_rp) - 1;
	if _end > 3000 {
		_end = 3000
	}

	return string(_rp[:_end])
}

func genCookie(cookieList []*http.Cookie) string {
	var buf bytes.Buffer

	for _, v := range cookieList {
		if buf.Len() > 0 {
			buf.WriteString("; ")
		}
		buf.WriteString(v.String())
	}

	return buf.String()
}

func genCode(resp string) int {
	var r Response
	var code int = http.StatusOK
	err := json.Unmarshal([]byte(resp), &r)
	if nil == err {
		code = r.Code
	}

	return code
}

func (l *DwLog)pushLog(t string, data map[string]interface{})  {
	pushData := make(map[string]interface{})
	pushData["message"] = data
	pushData["type"] = t
	pushData["time"] = FormatTime(time.Now())

	j, err := json.Marshal(pushData)
	if err == nil {
		go l.RedisPool.Get().Do("RPUSH", l.LogKey, string(j))
	}

}