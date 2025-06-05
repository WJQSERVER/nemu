package errpage

type HTTPErrors struct {
	StatusCode   int
	StatusDesc   string
	StatusTextZH string
}

var (
	ErrBadRequest = &HTTPErrors{
		StatusCode:   400,
		StatusDesc:   "Bad Request",
		StatusTextZH: "错误的请求",
	}
	ErrUnauthorized = &HTTPErrors{
		StatusCode:   401,
		StatusDesc:   "Unauthorized",
		StatusTextZH: "未授权",
	}
	ErrForbidden = &HTTPErrors{
		StatusCode:   403,
		StatusDesc:   "Forbidden",
		StatusTextZH: "禁止访问",
	}
	ErrNotFound = &HTTPErrors{
		StatusCode:   404,
		StatusDesc:   "Not Found",
		StatusTextZH: "未找到",
	}
	ErrMethodNotAllowed = &HTTPErrors{
		StatusCode:   405,
		StatusDesc:   "Method Not Allowed",
		StatusTextZH: "方法不允许",
	}
	ErrRequestTimeout = &HTTPErrors{
		StatusCode:   408,
		StatusDesc:   "Request Timeout",
		StatusTextZH: "请求超时",
	}
	ErrConflict = &HTTPErrors{
		StatusCode:   409,
		StatusDesc:   "Conflict",
		StatusTextZH: "冲突",
	}
	ErrGone = &HTTPErrors{
		StatusCode:   410,
		StatusDesc:   "Gone",
		StatusTextZH: "已移除",
	}
	ErrLengthRequired = &HTTPErrors{
		StatusCode:   411,
		StatusDesc:   "Length Required",
		StatusTextZH: "需要长度",
	}
	ErrPreconditionFailed = &HTTPErrors{
		StatusCode:   412,
		StatusDesc:   "Precondition Failed",
		StatusTextZH: "预处理失败",
	}
	ErrPayloadTooLarge = &HTTPErrors{
		StatusCode:   413,
		StatusDesc:   "Payload Too Large",
		StatusTextZH: "请求实体过大",
	}
	ErrURITooLong = &HTTPErrors{
		StatusCode:   414,
		StatusDesc:   "URI Too Long",
		StatusTextZH: "URI 过长",
	}
	ErrUnsupportedMediaType = &HTTPErrors{
		StatusCode:   415,
		StatusDesc:   "Unsupported Media Type",
		StatusTextZH: "不支持的媒体类型",
	}
	ErrRangeNotSatisfiable = &HTTPErrors{
		StatusCode:   416,
		StatusDesc:   "Range Not Satisfiable",
		StatusTextZH: "范围请求无法满足",
	}
	ErrExpectationFailed = &HTTPErrors{
		StatusCode:   417,
		StatusDesc:   "Expectation Failed",
		StatusTextZH: "期望失败",
	}
	ErrImATeapot = &HTTPErrors{
		StatusCode:   418,
		StatusDesc:   "I'm a teapot",
		StatusTextZH: "我是一个茶壶",
	}
	ErrMisdirectedRequest = &HTTPErrors{
		StatusCode:   421,
		StatusDesc:   "Misdirected Request",
		StatusTextZH: "错误的请求",
	}
	ErrUnprocessableEntity = &HTTPErrors{
		StatusCode:   422,
		StatusDesc:   "Unprocessable Entity",
		StatusTextZH: "无法处理的实体",
	}
	ErrLocked = &HTTPErrors{
		StatusCode:   423,
		StatusDesc:   "Locked",
		StatusTextZH: "已锁定",
	}
	ErrFailedDependency = &HTTPErrors{
		StatusCode:   424,
		StatusDesc:   "Failed Dependency",
		StatusTextZH: "依赖失败",
	}
	ErrTooEarly = &HTTPErrors{
		StatusCode:   425,
		StatusDesc:   "Too Early",
		StatusTextZH: "过早",
	}
	ErrUpgradeRequired = &HTTPErrors{
		StatusCode:   426,
		StatusDesc:   "Upgrade Required",
		StatusTextZH: "需要升级",
	}
	ErrPreconditionRequired = &HTTPErrors{
		StatusCode:   428,
		StatusDesc:   "Precondition Required",
		StatusTextZH: "需要预处理",
	}
	ErrTooManyRequests = &HTTPErrors{
		StatusCode:   429,
		StatusDesc:   "Too Many Requests",
		StatusTextZH: "请求过多",
	}
	ErrRequestHeaderFieldsTooLarge = &HTTPErrors{
		StatusCode:   431,
		StatusDesc:   "Request Header Fields Too Large",
		StatusTextZH: "请求头字段过大",
	}
	ErrUnavailableForLegalReasons = &HTTPErrors{
		StatusCode:   451,
		StatusDesc:   "Unavailable For Legal Reasons",
		StatusTextZH: "由于法律原因不可用",
	}
	ErrInternalServerError = &HTTPErrors{
		StatusCode:   500,
		StatusDesc:   "Internal Server Error",
		StatusTextZH: "服务器内部错误",
	}
	ErrNotImplemented = &HTTPErrors{
		StatusCode:   501,
		StatusDesc:   "Not Implemented",
		StatusTextZH: "未实现",
	}
	ErrBadGateway = &HTTPErrors{
		StatusCode:   502,
		StatusDesc:   "Bad Gateway",
		StatusTextZH: "错误的网关",
	}
	ErrServiceUnavailable = &HTTPErrors{
		StatusCode:   503,
		StatusDesc:   "Service Unavailable",
		StatusTextZH: "服务不可用",
	}
	ErrGatewayTimeout = &HTTPErrors{
		StatusCode:   504,
		StatusDesc:   "Gateway Timeout",
		StatusTextZH: "网关超时",
	}
	ErrHTTPVersionNotSupported = &HTTPErrors{
		StatusCode:   505,
		StatusDesc:   "HTTP Version Not Supported",
		StatusTextZH: "HTTP 版本不支持",
	}
	ErrVariantAlsoNegotiates = &HTTPErrors{
		StatusCode:   506,
		StatusDesc:   "Variant Also Negotiates",
		StatusTextZH: "变体也协商",
	}
	ErrInsufficientStorage = &HTTPErrors{
		StatusCode:   507,
		StatusDesc:   "Insufficient Storage",
		StatusTextZH: "存储空间不足",
	}
	ErrLoopDetected = &HTTPErrors{
		StatusCode:   508,
		StatusDesc:   "Loop Detected",
		StatusTextZH: "检测到循环",
	}
	ErrNotExtended = &HTTPErrors{
		StatusCode:   510,
		StatusDesc:   "Not Extended",
		StatusTextZH: "未扩展",
	}
	ErrNetworkAuthenticationRequired = &HTTPErrors{
		StatusCode:   511,
		StatusDesc:   "Network Authentication Required",
		StatusTextZH: "需要网络认证",
	}
	ErrUnknownError = &HTTPErrors{
		StatusCode:   520,
		StatusDesc:   "Unknown Error",
		StatusTextZH: "未知错误",
	}
	ErrWebServerIsDown = &HTTPErrors{
		StatusCode:   521,
		StatusDesc:   "Web Server Is Down",
		StatusTextZH: "Web 服务器已关闭",
	}
	ErrConnectionTimedOut = &HTTPErrors{
		StatusCode:   522,
		StatusDesc:   "Connection Timed Out",
		StatusTextZH: "连接超时",
	}
	ErrOriginIsUnreachable = &HTTPErrors{
		StatusCode:   523,
		StatusDesc:   "Origin Is Unreachable",
		StatusTextZH: "无法访问源站",
	}
	ErrATimeoutOccurred = &HTTPErrors{
		StatusCode:   524,
		StatusDesc:   "A Timeout Occurred",
		StatusTextZH: "发生超时",
	}
	ErrSSLHandshakeFailed = &HTTPErrors{
		StatusCode:   525,
		StatusDesc:   "SSL Handshake Failed",
		StatusTextZH: "SSL 握手失败",
	}
	ErrInvalidSSLCertificate = &HTTPErrors{
		StatusCode:   526,
		StatusDesc:   "Invalid SSL Certificate",
		StatusTextZH: "无效的 SSL 证书",
	}
	ErrRailgunError = &HTTPErrors{
		StatusCode:   527,
		StatusDesc:   "Railgun Error",
		StatusTextZH: "Railgun 错误",
	}
	ErrSiteOverloaded = &HTTPErrors{
		StatusCode:   529,
		StatusDesc:   "Site Overloaded",
		StatusTextZH: "站点过载",
	}
	ErrSiteIsFrozen = &HTTPErrors{
		StatusCode:   530,
		StatusDesc:   "Site Is Frozen",
		StatusTextZH: "站点已冻结",
	}
	ErrGenericError = &HTTPErrors{
		StatusCode:   599,
		StatusDesc:   "Generic Error",
		StatusTextZH: "通用错误",
	}
)

var statusCodeMap = map[int]*HTTPErrors{
	400: ErrBadRequest,
	401: ErrUnauthorized,
	403: ErrForbidden,
	404: ErrNotFound,
	405: ErrMethodNotAllowed,
	408: ErrRequestTimeout,
	409: ErrConflict,
	410: ErrGone,
	411: ErrLengthRequired,
	412: ErrPreconditionFailed,
	413: ErrPayloadTooLarge,
	414: ErrURITooLong,
	415: ErrUnsupportedMediaType,
	416: ErrRangeNotSatisfiable,
	417: ErrExpectationFailed,
	418: ErrImATeapot,
	421: ErrMisdirectedRequest,
	422: ErrUnprocessableEntity,
	423: ErrLocked,
	424: ErrFailedDependency,
	425: ErrTooEarly,
	426: ErrUpgradeRequired,
	428: ErrPreconditionRequired,
	429: ErrTooManyRequests,
	431: ErrRequestHeaderFieldsTooLarge,
	451: ErrUnavailableForLegalReasons,
	500: ErrInternalServerError,
	501: ErrNotImplemented,
	502: ErrBadGateway,
	503: ErrServiceUnavailable,
	504: ErrGatewayTimeout,
	505: ErrHTTPVersionNotSupported,
	506: ErrVariantAlsoNegotiates,
	507: ErrInsufficientStorage,
	508: ErrLoopDetected,
	510: ErrNotExtended,
	511: ErrNetworkAuthenticationRequired,
	520: ErrUnknownError,
	521: ErrWebServerIsDown,
	522: ErrConnectionTimedOut,
	523: ErrOriginIsUnreachable,
	524: ErrATimeoutOccurred,
	525: ErrSSLHandshakeFailed,
	526: ErrInvalidSSLCertificate,
	527: ErrRailgunError,
	529: ErrSiteOverloaded,
	530: ErrSiteIsFrozen,
	599: ErrGenericError,
}

func FindError(statusCode int) *HTTPErrors {
	if err, ok := statusCodeMap[statusCode]; ok {
		return err
	}
	return ErrUnknownError
}
