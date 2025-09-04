package httpclient

import (
	"gin-example/internal/pkg/core"
	"gin-example/internal/pkg/trace"

	"github.com/go-resty/resty/v2"
)

type customInterceptor struct {
	ctx core.StdContext
}

func (i *customInterceptor) OnResponse(client *resty.Client, response *resty.Response) error {
	requestInfo := map[string]interface{}{
		"request-id":   i.ctx.Trace.ID(),
		"url":          response.Request.URL,
		"method":       response.Request.Method,
		"header":       response.Request.Header,
		"path_params":  response.Request.PathParams,
		"body":         response.Request.Body,
		"request_time": response.Request.Time.Format("2006-01-02 15:04:05"),

		"ti-DNSLookup":      response.Request.TraceInfo().DNSLookup,
		"ti-ConnTime":       response.Request.TraceInfo().ConnTime.String(),
		"ti-TCPConnTime":    response.Request.TraceInfo().TCPConnTime.String(),
		"ti-TLSHandshake":   response.Request.TraceInfo().TLSHandshake.String(),
		"ti-ServerTime":     response.Request.TraceInfo().ServerTime.String(),
		"ti-IsConnReused":   response.Request.TraceInfo().IsConnReused,
		"ti-IsConnWasIdle":  response.Request.TraceInfo().IsConnWasIdle,
		"ti-ConnIdleTime":   response.Request.TraceInfo().ConnIdleTime.String(),
		"ti-RequestAttempt": response.Request.TraceInfo().RequestAttempt,
		"ti-RemoteAddr":     response.Request.TraceInfo().RemoteAddr.String(),
	}

	responseInfo := map[string]interface{}{
		"status_code": response.StatusCode(),
		"status":      response.Status(),
		"proto":       response.Proto(),
		"header":      response.Header(),
		"total_time":  response.Request.TraceInfo().TotalTime.String(),
		"received_at": response.ReceivedAt().Format("2006-01-02 15:04:05"),
		"body":        string(response.Body()),
	}

	httpLog := new(trace.HttpLog)
	httpLog.Request = requestInfo
	httpLog.Response = responseInfo
	httpLog.CostSeconds = response.Time().Seconds()

	i.ctx.Trace.AppendThirdPartyRequests(httpLog)

	return nil
}

func GetHttpClientWithContext(ctx core.StdContext) *resty.Client {
	client := resty.New().EnableTrace()

	interceptor := &customInterceptor{
		ctx: ctx,
	}

	client.OnAfterResponse(interceptor.OnResponse)

	return client
}

func GetHttpClient() *resty.Client {
	return resty.New().EnableTrace()
}
