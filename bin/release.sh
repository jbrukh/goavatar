cd $GOPATH/src/github.com/jbrukh/goavatar

export GORACE="log_path=stdout"
go build -v ./... && go test ./... && go install -v ./...
