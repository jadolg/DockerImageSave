package dockerimagesave

import (
	"net/http"

	log "github.com/sirupsen/logrus"
)

func CloseResponse(resp *http.Response) {
	if resp != nil {
		err := resp.Body.Close()
		if err != nil {
			log.Debug(err)
		}
	}
}
