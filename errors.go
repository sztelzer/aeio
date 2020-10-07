package aeio

import (
	"fmt"
	"log"
	"runtime"
)

type Error struct {
	Reference  string `json:"reference"`
	Original   string  `json:"error"`
	Where      string `json:"file"`
	HttpStatus int    `json:"-"`
}

func (e Error) Error() string {
	return fmt.Sprintf("%s, %s, %s, %d", e.Reference, e.Original, e.Where, e.HttpStatus)
}

func NewError(reference string, original error, status int) error {
	_, file, line, _ := runtime.Caller(1)
	err := Error{
		Reference:  reference,
		Original:   original.Error(),
		Where:      fmt.Sprintf("%s:%d", file, line),
		HttpStatus: status,
	}
	log.Print(err)
	return err
}