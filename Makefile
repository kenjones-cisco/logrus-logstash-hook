MAKEFLAGS += --warn-undefined-variables
SHELL := /bin/bash
.SHELLFLAGS := -o pipefail -euc
.DEFAULT_GOAL := test

export PROJECT = logrus-logstash-hook

IMPORT_PATH := github.com/kenjones-cisco/${PROJECT}

# Windows environment?
CYG_CHECK := $(shell hash cygpath 2>/dev/null && echo 1)
ifeq ($(CYG_CHECK),1)
	VBOX_CHECK := $(shell hash VBoxManage 2>/dev/null && echo 1)

	# Docker Toolbox (pre-Windows 10)
	ifeq ($(VBOX_CHECK),1)
		ROOT := /${PROJECT}
	else
		# Docker Windows
		ROOT := $(shell cygpath -m -a "$(shell pwd)")
	endif
else
	# all non-windows environments
	ROOT := $(shell pwd)
endif

DEV_IMAGE := ${PROJECT}_dev

DOCKERRUN := docker run --rm \
	-v ${ROOT}/vendor:/go/src \
	-v ${ROOT}:/${PROJECT}/src/${IMPORT_PATH} \
	-w /${PROJECT}/src/${IMPORT_PATH} \
	${DEV_IMAGE}

DOCKERNOVENDOR := docker run --rm -i \
	-v ${ROOT}:/${PROJECT}/src/${IMPORT_PATH} \
	-w /${PROJECT}/src/${IMPORT_PATH} \
	${DEV_IMAGE}

clean:
	@rm -rf cover

veryclean: clean
	@rm -rf tmp .glide vendor

## prefix before other make targets to run in your local dev environment
local: | quiet
	@$(eval DOCKERRUN= )
	@$(eval DOCKERNOVENDOR= )
quiet: # this is silly but shuts up 'Nothing to be done for `local`'
	@:

## builds the dev container
prepare: tmp/dev_image_id
tmp/dev_image_id: Dockerfile.dev
	@mkdir -p tmp
	@docker rmi -f ${DEV_IMAGE} > /dev/null 2>&1 || true
	@echo "## Building dev container"
	@docker build --quiet -t ${DEV_IMAGE} -f Dockerfile.dev .
	@docker inspect -f "{{ .ID }}" ${DEV_IMAGE} > tmp/dev_image_id

# ----------------------------------------------
# dependencies
# NOTE: glide will be replaced with `dep` when its production-ready
# ref https://github.com/golang/dep

## Install dependencies using glide if glide.yaml changed.
vendor: tmp/glide-installed
tmp/glide-installed: tmp/dev_image_id glide.yaml
	@mkdir -p vendor
	${DOCKERRUN} glide install --skip-test
	@date > tmp/glide-installed

## Install all dependencies using glide.
dep-install: prepare
	@mkdir -p vendor
	${DOCKERRUN} glide install --skip-test
	@date > tmp/glide-installed

## Update dependencies using glide.
dep-update: prepare
	${DOCKERRUN} glide up --skip-test

# usage DEP=github.com/owner/package make dep-add
## Add new dependencies to glide and install.
dep-add: prepare
ifeq ($(strip $(DEP)),)
	$(error "No dependency provided. Expected: DEP=<go import path>")
endif
	${DOCKERNOVENDOR} glide get --skip-test ${DEP}

# ----------------------------------------------
# develop and test

.PHONY: format
format: tmp/glide-installed
	${DOCKERNOVENDOR} bash ./scripts/fmt.sh

.PHONY: check
check: format
	${DOCKERNOVENDOR} bash ./scripts/check.sh

# default task
.PHONY: test
test: check
	${DOCKERRUN} bash ./scripts/test.sh

# run unit tests and write out test coverage
.PHONY: cover
cover: check
	@rm -rf cover/
	@mkdir -p cover
	${DOCKERRUN} bash ./scripts/cover.sh
	@chmod 644 cover/coverage.html
