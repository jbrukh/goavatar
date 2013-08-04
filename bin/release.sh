cd $GOPATH/src/github.com/jbrukh/goavatar

export GORACE="log_path=stdout"
go build ./... && go install -v ./... && go test ./...

if [ $? -eq 0 ] && [ "$1" != "--no-version-stamp" ]; then
  if [[ -n $(git status -s) ]]; then
    echo ""
    echo "WARNING: not updating version, your working tree is dirty."
    echo ""
  else
    SHA=$(git rev-parse HEAD)
    echo "Updating version to: $SHA"
    sed -i '' "s#\(GoavatarVersionSha   = \"\)\([a-f0-9]*\)\"#\1$SHA\"#g" version.go
    git commit -a -m "Updating latest Octopus version to $SHA."
  fi
fi

echo ""
echo "Suggestions from go-vet..."
echo ""
find . -type d -name "*" -and -not -path "./.git*" | xargs go tool vet --all

echo ""
echo "Done!"
