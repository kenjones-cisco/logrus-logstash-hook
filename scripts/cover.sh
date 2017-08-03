#!/bin/bash
# Generate test coverage statistics for Go packages.
#
# Works around the fact that `go test -coverprofile` currently does not work
# with multiple packages, see https://code.google.com/p/go/issues/detail?id=6909
#

set -e

workdir=cover
profile="$workdir/cover.out"
mode=count

generate_cover_data() {
    for pkg in $(glide nv);
    do
        for subpkg in $(go list ${pkg});
        do
            f="$workdir/$(echo $subpkg | tr / -).cover"
            go test -v -covermode="$mode" -coverprofile="$f" "$subpkg" >> test.out
        done
    done

    set -- "$workdir"/*.cover
    if [ ! -f "$1" ]; then
        echo "No Test Cases"; exit 0
    fi
    echo "mode: $mode" >"$profile"
    grep -h -v "^mode:" "$workdir"/*.cover >>"$profile"
    # display actual test results
    cat test.out || :
}

show_html_report() {
    go tool cover -html="$profile" -o="$workdir"/coverage.html
}

rm -f test.out
generate_cover_data
show_html_report
