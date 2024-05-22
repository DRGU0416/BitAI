package middleware

import (
	"net/http"
	"regexp"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
)

const (
	HeaderAccept         = "Accept"
	HeaderAcceptEncoding = "Accept-Encoding"

	HeaderAllow               = "Allow"
	HeaderAuthorization       = "Authorization"
	HeaderContentDisposition  = "Content-Disposition"
	HeaderContentEncoding     = "Content-Encoding"
	HeaderContentLength       = "Content-Length"
	HeaderContentType         = "Content-Type"
	HeaderCookie              = "Cookie"
	HeaderSetCookie           = "Set-Cookie"
	HeaderIfModifiedSince     = "If-Modified-Since"
	HeaderLastModified        = "Last-Modified"
	HeaderLocation            = "Location"
	HeaderRetryAfter          = "Retry-After"
	HeaderUpgrade             = "Upgrade"
	HeaderVary                = "Vary"
	HeaderWWWAuthenticate     = "WWW-Authenticate"
	HeaderXForwardedFor       = "X-Forwarded-For"
	HeaderXForwardedProto     = "X-Forwarded-Proto"
	HeaderXForwardedProtocol  = "X-Forwarded-Protocol"
	HeaderXForwardedSsl       = "X-Forwarded-Ssl"
	HeaderXUrlScheme          = "X-Url-Scheme"
	HeaderXHTTPMethodOverride = "X-HTTP-Method-Override"
	HeaderXRealIP             = "X-Real-Ip"
	HeaderXRequestID          = "X-Request-Id"
	HeaderXCorrelationID      = "X-Correlation-Id"
	HeaderXRequestedWith      = "X-Requested-With"
	HeaderServer              = "Server"
	HeaderOrigin              = "Origin"
	HeaderCacheControl        = "Cache-Control"
	HeaderConnection          = "Connection"

	// Access control
	HeaderAccessControlRequestMethod    = "Access-Control-Request-Method"
	HeaderAccessControlRequestHeaders   = "Access-Control-Request-Headers"
	HeaderAccessControlAllowOrigin      = "Access-Control-Allow-Origin"
	HeaderAccessControlAllowMethods     = "Access-Control-Allow-Methods"
	HeaderAccessControlAllowHeaders     = "Access-Control-Allow-Headers"
	HeaderAccessControlAllowCredentials = "Access-Control-Allow-Credentials"
	HeaderAccessControlExposeHeaders    = "Access-Control-Expose-Headers"
	HeaderAccessControlMaxAge           = "Access-Control-Max-Age"

	// Security
	HeaderStrictTransportSecurity         = "Strict-Transport-Security"
	HeaderXContentTypeOptions             = "X-Content-Type-Options"
	HeaderXXSSProtection                  = "X-XSS-Protection"
	HeaderXFrameOptions                   = "X-Frame-Options"
	HeaderContentSecurityPolicy           = "Content-Security-Policy"
	HeaderContentSecurityPolicyReportOnly = "Content-Security-Policy-Report-Only"
	HeaderXCSRFToken                      = "X-CSRF-Token"
	HeaderReferrerPolicy                  = "Referrer-Policy"
)

type (
	// CORSConfig defines the config for CORS middleware.
	CORSConfig struct {
		// Skipper defines a function to skip middleware.
		Skipper Skipper

		// AllowOrigins determines the value of the Access-Control-Allow-Origin
		// response header.  This header defines a list of origins that may access the
		// resource.  The wildcard characters '*' and '?' are supported and are
		// converted to regex fragments '.*' and '.' accordingly.
		//
		// Security: use extreme caution when handling the origin, and carefully
		// validate any logic. Remember that attackers may register hostile domain names.
		// See https://blog.portswigger.net/2016/10/exploiting-cors-misconfigurations-for.html
		//
		// Optional. Default value []string{"*"}.
		//
		// See also: https://developer.mozilla.org/en-US/docs/Web/HTTP/Headers/Access-Control-Allow-Origin
		AllowOrigins []string `yaml:"allow_origins"`

		// AllowOriginFunc is a custom function to validate the origin. It takes the
		// origin as an argument and returns true if allowed or false otherwise. If
		// an error is returned, it is returned by the handler. If this option is
		// set, AllowOrigins is ignored.
		//
		// Security: use extreme caution when handling the origin, and carefully
		// validate any logic. Remember that attackers may register hostile domain names.
		// See https://blog.portswigger.net/2016/10/exploiting-cors-misconfigurations-for.html
		//
		// Optional.
		AllowOriginFunc func(origin string) (bool, error) `yaml:"allow_origin_func"`

		// AllowMethods determines the value of the Access-Control-Allow-Methods
		// response header.  This header specified the list of methods allowed when
		// accessing the resource.  This is used in response to a preflight request.
		//
		// Optional. Default value DefaultCORSConfig.AllowMethods.
		// If `allowMethods` is left empty, this middleware will fill for preflight
		// request `Access-Control-Allow-Methods` header value
		// from `Allow` header that gin.Router set into context.
		//
		// See also: https://developer.mozilla.org/en-US/docs/Web/HTTP/Headers/Access-Control-Allow-Methods
		AllowMethods []string `yaml:"allow_methods"`

		// AllowHeaders determines the value of the Access-Control-Allow-Headers
		// response header.  This header is used in response to a preflight request to
		// indicate which HTTP headers can be used when making the actual request.
		//
		// Optional. Default value []string{}.
		//
		// See also: https://developer.mozilla.org/en-US/docs/Web/HTTP/Headers/Access-Control-Allow-Headers
		AllowHeaders []string `yaml:"allow_headers"`

		// AllowCredentials determines the value of the
		// Access-Control-Allow-Credentials response header.  This header indicates
		// whether or not the response to the request can be exposed when the
		// credentials mode (Request.credentials) is true. When used as part of a
		// response to a preflight request, this indicates whether or not the actual
		// request can be made using credentials.  See also
		// [MDN: Access-Control-Allow-Credentials].
		//
		// Optional. Default value false, in which case the header is not set.
		//
		// Security: avoid using `AllowCredentials = true` with `AllowOrigins = *`.
		// See "Exploiting CORS misconfigurations for Bitcoins and bounties",
		// https://blog.portswigger.net/2016/10/exploiting-cors-misconfigurations-for.html
		//
		// See also: https://developer.mozilla.org/en-US/docs/Web/HTTP/Headers/Access-Control-Allow-Credentials
		AllowCredentials bool `yaml:"allow_credentials"`

		// UnsafeWildcardOriginWithAllowCredentials UNSAFE/INSECURE: allows wildcard '*' origin to be used with AllowCredentials
		// flag. In that case we consider any origin allowed and send it back to the client with `Access-Control-Allow-Origin` header.
		//
		// This is INSECURE and potentially leads to [cross-origin](https://portswigger.net/research/exploiting-cors-misconfigurations-for-bitcoins-and-bounties)
		// attacks. See: https://github.com/labstack/gin/issues/2400 for discussion on the subject.
		//
		// Optional. Default value is false.
		UnsafeWildcardOriginWithAllowCredentials bool `yaml:"unsafe_wildcard_origin_with_allow_credentials"`

		// ExposeHeaders determines the value of Access-Control-Expose-Headers, which
		// defines a list of headers that clients are allowed to access.
		//
		// Optional. Default value []string{}, in which case the header is not set.
		//
		// See also: https://developer.mozilla.org/en-US/docs/Web/HTTP/Headers/Access-Control-Expose-Header
		ExposeHeaders []string `yaml:"expose_headers"`

		// MaxAge determines the value of the Access-Control-Max-Age response header.
		// This header indicates how long (in seconds) the results of a preflight
		// request can be cached.
		//
		// Optional. Default value 0.  The header is set only if MaxAge > 0.
		//
		// See also: https://developer.mozilla.org/en-US/docs/Web/HTTP/Headers/Access-Control-Max-Age
		MaxAge int `yaml:"max_age"`
	}
)

var (
	// DefaultCORSConfig is the default CORS middleware config.
	DefaultCORSConfig = CORSConfig{
		Skipper:      DefaultSkipper,
		AllowOrigins: []string{"*"},
		AllowMethods: []string{http.MethodGet, http.MethodHead, http.MethodPut, http.MethodPatch, http.MethodPost, http.MethodDelete},
	}
)

// Errors
var (
	ErrCORSContext = gin.H{"code": 0, "message": "invalid context"}
)

// CORS returns a Cross-Origin Resource Sharing (CORS) middleware.
// See also [MDN: Cross-Origin Resource Sharing (CORS)].
//
// Security: Poorly configured CORS can compromise security because it allows
// relaxation of the browser's Same-Origin policy.  See [Exploiting CORS
// misconfigurations for Bitcoins and bounties] and [Portswigger: Cross-origin
// resource sharing (CORS)] for more details.
//
// [MDN: Cross-Origin Resource Sharing (CORS)]: https://developer.mozilla.org/en/docs/Web/HTTP/Access_control_CORS
// [Exploiting CORS misconfigurations for Bitcoins and bounties]: https://blog.portswigger.net/2016/10/exploiting-cors-misconfigurations-for.html
// [Portswigger: Cross-origin resource sharing (CORS)]: https://portswigger.net/web-security/cors
func CORS() gin.HandlerFunc {
	return CORSWithConfig(DefaultCORSConfig)
}

// CORSWithConfig returns a CORS middleware with config.
// See: [CORS].
func CORSWithConfig(config CORSConfig) gin.HandlerFunc {
	// Defaults
	if config.Skipper == nil {
		config.Skipper = DefaultCORSConfig.Skipper
	}
	if len(config.AllowOrigins) == 0 {
		config.AllowOrigins = DefaultCORSConfig.AllowOrigins
	}
	hasCustomAllowMethods := true
	if len(config.AllowMethods) == 0 {
		hasCustomAllowMethods = false
		config.AllowMethods = DefaultCORSConfig.AllowMethods
	}

	allowOriginPatterns := []string{}
	for _, origin := range config.AllowOrigins {
		pattern := regexp.QuoteMeta(origin)
		pattern = strings.Replace(pattern, "\\*", ".*", -1)
		pattern = strings.Replace(pattern, "\\?", ".", -1)
		pattern = "^" + pattern + "$"
		allowOriginPatterns = append(allowOriginPatterns, pattern)
	}

	allowMethods := strings.Join(config.AllowMethods, ",")
	allowHeaders := strings.Join(config.AllowHeaders, ",")
	exposeHeaders := strings.Join(config.ExposeHeaders, ",")
	maxAge := strconv.Itoa(config.MaxAge)

	return func(c *gin.Context) {
		if config.Skipper(c) {
			c.Next()
			return
		}

		req := c.Request
		origin := req.Header.Get(HeaderOrigin)
		allowOrigin := ""
		// fmt.Fprintf(gin.DefaultWriter, origin)

		c.Header(HeaderVary, HeaderOrigin)

		// Preflight request is an OPTIONS request, using three HTTP request headers: Access-Control-Request-Method,
		// Access-Control-Request-Headers, and the Origin header. See: https://developer.mozilla.org/en-US/docs/Glossary/Preflight_request
		// For simplicity we just consider method type and later `Origin` header.
		preflight := req.Method == http.MethodOptions

		// Although router adds special handler in case of OPTIONS method we avoid calling next for OPTIONS in this middleware
		// as CORS requests do not have cookies / authentication headers by default, so we could get stuck in auth
		// middlewares by calling next(c).
		// But we still want to send `Allow` header as response in case of Non-CORS OPTIONS request as router default
		// handler does.
		routerAllowMethods := ""
		if preflight {
			// TODO: Echo代码，临时注释
			// ContextKeyHeaderAllow = "echo_header_allow"
			// tmpAllowMethods, ok := c.Get(ContextKeyHeaderAllow).(string)
			// if ok && tmpAllowMethods != "" {
			// 	routerAllowMethods = tmpAllowMethods
			// 	c.Header(HeaderAllow, routerAllowMethods)
			// }
		}

		// No Origin provided. This is (probably) not request from actual browser - proceed executing middleware chain
		if origin == "" {
			if !preflight {
				c.Next()
				return
			}
			c.JSON(http.StatusNoContent, ErrCORSContext)
			return
		}

		if config.AllowOriginFunc != nil {
			allowed, err := config.AllowOriginFunc(origin)
			if err != nil {
				c.JSON(http.StatusNoContent, ErrCORSContext)
				return
			}
			if allowed {
				allowOrigin = origin
			}
		} else {
			// Check allowed origins
			for _, o := range config.AllowOrigins {
				if o == "*" && config.AllowCredentials && config.UnsafeWildcardOriginWithAllowCredentials {
					allowOrigin = origin
					break
				}
				if o == "*" || o == origin {
					allowOrigin = o
					break
				}
				if matchSubdomain(origin, o) {
					allowOrigin = origin
					break
				}
			}

			checkPatterns := false
			if allowOrigin == "" {
				// to avoid regex cost by invalid (long) domains (253 is domain name max limit)
				if len(origin) <= (253+3+5) && strings.Contains(origin, "://") {
					checkPatterns = true
				}
			}
			if checkPatterns {
				for _, re := range allowOriginPatterns {
					if match, _ := regexp.MatchString(re, origin); match {
						allowOrigin = origin
						break
					}
				}
			}
		}

		// Origin not allowed
		if allowOrigin == "" {
			if !preflight {
				c.Next()
				return
			}
			c.JSON(http.StatusNoContent, ErrCORSContext)
			return
		}

		c.Header(HeaderAccessControlAllowOrigin, allowOrigin)
		if config.AllowCredentials {
			c.Header(HeaderAccessControlAllowCredentials, "true")
		}

		// Simple request
		if !preflight {
			if exposeHeaders != "" {
				c.Header(HeaderAccessControlExposeHeaders, exposeHeaders)
			}
			c.Next()
			return
		}

		// Preflight request
		c.Header(HeaderVary, HeaderAccessControlRequestMethod)
		c.Header(HeaderVary, HeaderAccessControlRequestHeaders)

		if !hasCustomAllowMethods && routerAllowMethods != "" {
			c.Header(HeaderAccessControlAllowMethods, routerAllowMethods)
		} else {
			c.Header(HeaderAccessControlAllowMethods, allowMethods)
		}

		if allowHeaders != "" {
			c.Header(HeaderAccessControlAllowHeaders, allowHeaders)
		} else {
			h := req.Header.Get(HeaderAccessControlRequestHeaders)
			if h != "" {
				c.Header(HeaderAccessControlAllowHeaders, h)
			}
		}
		if config.MaxAge > 0 {
			c.Header(HeaderAccessControlMaxAge, maxAge)
		}

		c.JSON(http.StatusNoContent, ErrCORSContext)
		return
	}
}

func matchScheme(domain, pattern string) bool {
	didx := strings.Index(domain, ":")
	pidx := strings.Index(pattern, ":")
	return didx != -1 && pidx != -1 && domain[:didx] == pattern[:pidx]
}

// matchSubdomain compares authority with wildcard
func matchSubdomain(domain, pattern string) bool {
	if !matchScheme(domain, pattern) {
		return false
	}
	didx := strings.Index(domain, "://")
	pidx := strings.Index(pattern, "://")
	if didx == -1 || pidx == -1 {
		return false
	}
	domAuth := domain[didx+3:]
	// to avoid long loop by invalid long domain
	if len(domAuth) > 253 {
		return false
	}
	patAuth := pattern[pidx+3:]

	domComp := strings.Split(domAuth, ".")
	patComp := strings.Split(patAuth, ".")
	for i := len(domComp)/2 - 1; i >= 0; i-- {
		opp := len(domComp) - 1 - i
		domComp[i], domComp[opp] = domComp[opp], domComp[i]
	}
	for i := len(patComp)/2 - 1; i >= 0; i-- {
		opp := len(patComp) - 1 - i
		patComp[i], patComp[opp] = patComp[opp], patComp[i]
	}

	for i, v := range domComp {
		if len(patComp) <= i {
			return false
		}
		p := patComp[i]
		if p == "*" {
			return true
		}
		if p != v {
			return false
		}
	}
	return false
}
