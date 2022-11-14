package util

import (
	"io"
	"log"
)

func CloseAndLogErrors(source string, closer io.Closer) {
	if err := closer.Close(); err != nil {
		log.Println(source, err)
	}
}
