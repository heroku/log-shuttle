package lpx_test

import (
	"bufio"
	"github.com/bmizerany/lpx"
	"net/http"
)

func Example() {
	h := func(w http.ResponseWriter, r *http.Request) {
		lp := lpx.NewReader(bufio.NewReader(r.Body))
		for lp.Next() {
			if string(lp.Header().Name) == "router" {
				decodeMsg(lp.Bytes())
			}
		}
	}
	http.HandleFunc("/drain", h)
	_ = http.ListenAndServe(":8080", nil)
}

func decodeMsg(b []byte) {
	// process b
}
