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

gpff-env() {
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

  test -d "${src}/gpff" || {
    mkdir -p "${src}"; (
      cd "${src}"; ln -sf ../../../ gpff
    )
  }

  test -d "${alias}/gpff" || (
    cd "${alias}"; ln -sf ../ gpff
  )

  export GOPATH
}

gpff-env

