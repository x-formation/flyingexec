#!/usr/bin/env bash

case "$(uname)" in
  Darwin)
    type -a greadlink &>/dev/null && {
      TOP=$(dirname $(greadlink -f "${BASH_SOURCE[0]}"))
    } || {
      TOP=$(dirname "${BASH_SOURCE[0]}")
    }
    ;;
  *)
    TOP=$(dirname $(readlink -f "${BASH_SOURCE[0]}"))
    ;;
esac

flyingexec-env() {
  unset GOBIN
  test -z "${GOPATH}" && {
    GOPATH=${TOP}
  } || {
    echo "${GOPATH}" | tr ':' '\n' | grep -x "${TOP}" &>/dev/null || {
      GOPATH=${TOP}:${GOPATH}
    }
  }
  local src=${TOP}/src/github.com/rjeczalik
  local alias=${TOP}/src
  test -d "${src}/flyingexec" || {
    mkdir -p "${src}"; (
      cd "${src}"; ln -sf ../../../ flyingexec
    )
  }
  test -d "${alias}/flyingexec" || (
    cd "${alias}"; ln -sf ../ flyingexec
  )
  export GOPATH
}

flyingexec-env
