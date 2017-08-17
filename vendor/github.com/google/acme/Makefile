# Copyright 2016 Google Inc. All Rights Reserved.
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

RELEASES=bin/acme-darwin-amd64 \
	 bin/acme-linux-amd64 \
	 bin/acme-linux-386 \
	 bin/acme-linux-arm \
	 bin/acme-windows-amd64.exe \
	 bin/acme-windows-386.exe \
	 bin/acme-solaris-amd64 

all: $(RELEASES)

bin/acme-%: GOOS=$(firstword $(subst -, ,$*))
bin/acme-%: GOARCH=$(subst .exe,,$(word 2,$(subst -, ,$*)))
bin/acme-%: $(wildcard *.go)
	GOOS=$(GOOS) GOARCH=$(GOARCH) CGO_ENABLED=0 go build \
	     -ldflags "-X main.osarch=$(GOOS)/$(GOARCH) -s -w" \
	     -buildmode=exe \
	     -tags release \
	     -o $@

clean:
	rm -rf bin
