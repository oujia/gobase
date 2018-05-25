package gobase

type BaseError struct {
	errno int
	msg string
}

func (e *BaseError)Error() string {
	return e.msg
}

func (e *BaseError)Code() int {
	return e.errno
}

var CODE_SUCCESS *BaseError = &BaseError{errno: 0, msg: "成功"}
var CODE_UNKNOW_ERROR *BaseError = &BaseError{errno: -1, msg: "未知错误"}
var CODE_PARAM_ERROR *BaseError = &BaseError{errno: -3, msg: "参数错误"}
var CODE_SIGN_ERROR *BaseError = &BaseError{errno: -6, msg: "签名错误"}
var CODE_UNLOGIN_ERROR *BaseError = &BaseError{errno: -7, msg: "没有登录"}
var CODE_NO_PERMITION *BaseError = &BaseError{errno: -8, msg: "没有权限"}
var CODE_NORMAL_ERROR *BaseError = &BaseError{errno: -9, msg: "常规错误"}
var CODE_DB_ERROR *BaseError = &BaseError{errno: -10, msg: "系统繁忙，请稍后再试"}
var CODE_REQUEST_TIMEOUT *BaseError = &BaseError{errno: -11, msg: "网络请求超时，请重试"}
var CODE_CACHE_ERROR *BaseError = &BaseError{errno: -13, msg: "系统繁忙，缓存错误，请稍后再"}