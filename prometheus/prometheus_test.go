package prometheus

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestQueryParsing(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("query") == "bad" {
			w.WriteHeader(http.StatusBadRequest)
			_, _ = w.Write([]byte(`{"status":"error","errorType":"bad_data","error":"parse error"}`))
			return
		}
		_, _ = w.Write([]byte(`{"status":"success","data":{"resultType":"vector","result":[
			{"metric":{"if_name":"et-5/5/0"},"value":[1761127200.5,"1250000"]},
			{"metric":{"if_name":"et-5/5/1"},"value":[1761127200.5,"NaN"]}]}}`))
	}))
	defer srv.Close()
	old := prometheusURL
	prometheusURL = srv.URL
	defer func() { prometheusURL = old }()

	series, err := Query(context.Background(), "x", time.Time{})
	if err != nil {
		t.Fatal(err)
	}
	if len(series) != 2 || *series[0].Points[0].Value != 1250000 || series[1].Points[0].Value != nil {
		t.Errorf("unexpected result: %+v", series)
	}
	if _, err := Query(context.Background(), "bad", time.Time{}); err == nil {
		t.Errorf("expected prometheus error")
	}
}
