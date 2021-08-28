package handlers

import "fmt"

type MyResponse struct {
	Code int
	Body string
	Err  error
}

func (res MyResponse) String() string {
	if err := res.Err; err != nil {
		return fmt.Sprintf("------\nHeader: \n%d\n------\nBody: \n%s\n------", res.Code, err.Error())
	}
	return fmt.Sprintf("------\nHeader: \n%d\n------\nBody: \n%s\n------", res.Code, res.Body)
}
