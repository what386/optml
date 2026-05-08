default:
    just --list

fmt:
    go fmt ./...

lint:
    test -z "$(gofmt -l .)"
    go vet ./...

test:
    go test ./...

run *args:
    go run .

srun *args:
    rm ./optml
    go build .
    sudo ./optml {{args}}



prepare version:
    lash run scripts/release/prepare.lash {{version}}

promote:
    just lint
    just test
    lash run scripts/release/promote.lash

publish version:
    lash run scripts/release/publish.lash {{version}}
    git switch dev
