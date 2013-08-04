cd $GOPATH/src/github.com/jbrukh/goavatar

export GORACE="log_path=stdout"

if [ "$1" != "--no-version-stamp" ]; then
  if [[ -n $(git status -s) ]]; then
    echo ""
    echo "WARNING: not updating version, your working tree is dirty."
    echo ""
  else
    SHA=$(git rev-parse HEAD)
    echo ""
    echo "Updating version to: $SHA"
    echo ""
    sed -i '' "s#\(GoavatarVersionSha   = \"\)\([a-f0-9]*\)\"#\1$SHA\"#g" version.go
  fi
fi

go build ./... && go install -v ./... && go test ./...

echo ""
echo "Suggestions from go-vet..."
echo ""
find . -type d -name "*" -and -not -path "./.git*" | xargs go tool vet --all

echo ""
echo "Done!"
