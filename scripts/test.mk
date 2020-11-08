# Copyright 2020 The arhat.dev Authors.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

GO_TEST := GOOS=$(shell go env GOHOSTOS) GOARCH=$(shell go env GOHOSTARCH) CGO_ENABLED=1 \
	go test -timeout 10m -mod=readonly -v -failfast -covermode=atomic -coverpkg=./... -race -cpu 2,4

GO_BENCH := GOOS=$(shell go env GOHOSTOS) GOARCH=$(shell go env GOHOSTARCH) CGO_ENABLED=1 \
	go test -timeout 30m -mod=readonly -bench '^Benchmark.*' -benchmem -covermode=atomic \
	-coverpkg=./... -benchtime=100000x -run '^Benchmark.*' -v

test.unit.codec:
	${GO_TEST} -coverprofile=coverage.codec.txt ./codec

test.unit.extperipheral:
	${GO_TEST} -coverprofile=coverage.extperipheral.txt ./extperipheral

test.unit.extruntime:
	${GO_TEST} -coverprofile=coverage.extruntime.txt ./extruntime

test.unit.server:
	${GO_TEST} -coverprofile=coverage.server.txt ./server

test.unit.libext:
	${GO_TEST} -coverprofile=coverage.libext.txt .

test.unit: \
	test.unit.codec \
	test.unit.extperipheral \
	test.unit.extruntime \
	test.unit.server \
	test.unit.libext
	${GO_TEST} -coverprofile=coverage.txt ./...

test.benchmark:
	${GO_BENCH} -coverprofile=coverage.bench.txt ./

install.fuzz:
	sh scripts/fuzz.sh install

test.fuzz:
	sh scripts/fuzz.sh run
