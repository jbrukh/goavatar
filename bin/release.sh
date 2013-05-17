cd $GOPATH/src/github.com/jbrukh/goavatar

export GORACE="log_path=stdout"
go build ./... && go install -v ./... && go test ./...

echo ""
echo "Suggestions from go-vet..."
echo ""
find . -type d -name "*" -and -not -path "./.git*" | xargs go tool vet --all

echo ""
echo "Done!"
