package handlers

type MyRequest struct {
	path string
}

func (req *MyRequest) SetPath(path string) {
	req.path = path
}

func (req *MyRequest) GetPath() string {
	return req.path
}
